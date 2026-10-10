package recording

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"abel/src/backend/lib/config"
)

type ProcessingJob struct {
	ID               string  `json:"id"`
	Source           string  `json:"source"`
	Output           string  `json:"output"`
	Stage            string  `json:"stage"`
	Progress         float64 `json:"progress"`
	ProcessedSeconds float64 `json:"processedSeconds"`
	TotalSeconds     float64 `json:"totalSeconds"`
	Error            string  `json:"error,omitempty"`
	AutoPush         bool    `json:"autoPush"`
	Start            float64 `json:"-"`
	End              float64 `json:"-"`
}

type RecordingExport struct {
	Name            string    `json:"name"`
	Size            int64     `json:"size"`
	ModTime         time.Time `json:"modTime"`
	Pushed          bool      `json:"pushed"`
	CloudPath       string    `json:"cloudPath"`
	CloudTargetPath string    `json:"cloudTargetPath"`
	Duration        float64   `json:"duration"`
}

type RecordingEntry struct {
	Name         string            `json:"name"`
	Display      string            `json:"display"`
	Size         int64             `json:"size"`
	ModTime      time.Time         `json:"modTime"`
	Duration     float64           `json:"duration"`
	RawCloudPath string            `json:"rawCloudPath"`
	RawPushed    bool              `json:"rawPushed"`
	Exports      []RecordingExport `json:"exports"`
	Jobs         []ProcessingJob   `json:"jobs"`
}

// RecordingProcessor coordinates post-processing, trimming, MP3 conversion, and cloud delivery.
type RecordingProcessor struct {
	cfg              *config.Config
	mu               sync.RWMutex
	cloudMu          sync.Mutex
	cloudNames       map[string]string
	cloudNamesLoaded bool
	jobs             map[string]ProcessingJob
	cancels          map[string]context.CancelFunc
	queue            chan string
	info             map[string]cachedDuration
	onUpdate         func()
}

type cachedDuration struct {
	size     int64
	modTime  time.Time
	duration float64
}

var ErrInvalidExport = errors.New("invalid audio filename")
var ErrQueueFull = errors.New("processing queue is full")
var ErrJobNotCancellable = errors.New("processing job is not active")

func NewRecordingProcessor(cfg *config.Config) *RecordingProcessor {
	p := &RecordingProcessor{
		cfg:     cfg,
		jobs:    make(map[string]ProcessingJob),
		cancels: make(map[string]context.CancelFunc),
		info:    make(map[string]cachedDuration),
		queue:   make(chan string, 32),
	}
	go p.work()
	go p.reconcileCloudLoop()
	return p
}

// StorageLocation returns the configured local recording storage directory.
func (p *RecordingProcessor) StorageLocation() string {
	return p.cfg.StorageLocation
}

// CloudLocation returns the configured cloud drive destination directory.
func (p *RecordingProcessor) CloudLocation() string {
	return p.cfg.CloudDriveLocation
}

// SetOnUpdate registers a callback to be invoked when processing state changes.
func (p *RecordingProcessor) SetOnUpdate(fn func()) {
	p.mu.Lock()
	p.onUpdate = fn
	p.mu.Unlock()
}

func (p *RecordingProcessor) notifyUpdate() {
	p.mu.RLock()
	fn := p.onUpdate
	p.mu.RUnlock()
	if fn != nil {
		fn()
	}
}

func (p *RecordingProcessor) reconcileCloudLoop() {
	if err := p.reconcileCloudExports(); err != nil {
		slog.Warn("cloud export reconciliation failed", "error", err)
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		if err := p.reconcileCloudExports(); err != nil {
			slog.Warn("cloud export reconciliation failed", "error", err)
		}
	}
}

func (p *RecordingProcessor) reconcileCloudExports() error {
	entries, err := p.Library()
	if err != nil {
		return err
	}
	var failures []error
	for _, entry := range entries {
		for _, export := range entry.Exports {
			if export.Pushed {
				continue
			}
			latestStage := ""
			for _, job := range entry.Jobs {
				if job.Output == export.Name {
					latestStage = job.Stage
				}
			}
			if activeProcessingStage(latestStage) || latestStage == "cancelled" {
				continue
			}
			if err := p.Push(export.Name); err != nil {
				failures = append(failures, fmt.Errorf("%s: %w", export.Name, err))
			}
		}
	}
	return errors.Join(failures...)
}

func validFileName(name, extension string) bool {
	return name != "" && name == filepath.Base(name) && name != "." &&
		!strings.ContainsAny(name, `/\`) && !strings.HasPrefix(name, ".") &&
		strings.EqualFold(filepath.Ext(name), extension)
}

func ValidAudioName(name string) bool {
	for _, extension := range []string{".wav", ".mp3", ".m4a", ".flac", ".aac", ".ogg"} {
		if validFileName(name, extension) {
			return true
		}
	}
	return false
}

func outputStem(name string) string {
	if strings.EqualFold(filepath.Ext(name), ".wav") {
		return name[:len(name)-4]
	}
	return name
}

func regularFile(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("not a regular file")
	}
	return info, nil
}

func (p *RecordingProcessor) validateManualTrim(stem string, start, end float64) error {
	processed := stem + "-processed.mp3"
	if _, err := regularFile(filepath.Join(p.cfg.StorageLocation, processed)); err != nil {
		return errors.New("process this recording before trimming")
	}
	duration, err := p.duration(processed, nil)
	if err != nil || duration <= 0 {
		return errors.New("processed MP3 is not readable")
	}
	if end > duration+0.05 || start >= duration || (end > 0 && end-start < 0.5) {
		return errors.New("trim range is outside processed MP3 or shorter than 0.5 seconds")
	}
	return nil
}

func (p *RecordingProcessor) validateEnqueueSource(source string, start, end float64) error {
	if !ValidAudioName(source) {
		return errors.New("invalid audio filename")
	}
	sourcePath := filepath.Join(p.cfg.StorageLocation, source)
	if _, err := regularFile(sourcePath); err != nil {
		return fmt.Errorf("recording unavailable: %w", err)
	}
	if duration, err := p.duration(source, nil); err != nil || duration <= 0 {
		return errors.New("recording is not finalized or has no audio")
	}
	if math.IsNaN(start) || math.IsInf(start, 0) || start < 0 || math.IsNaN(end) || math.IsInf(end, 0) || end < 0 {
		return errors.New("invalid trim range")
	}
	if end > 0 && end <= start {
		return errors.New("trim end must exceed start")
	}
	return nil
}

func (p *RecordingProcessor) Enqueue(source string, start, end float64, autoPush bool) (ProcessingJob, error) {
	if err := p.validateEnqueueSource(source, start, end); err != nil {
		return ProcessingJob{}, err
	}
	stem := outputStem(source)
	if !autoPush {
		if err := p.validateManualTrim(stem, start, end); err != nil {
			return ProcessingJob{}, err
		}
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	output := stem + "-processed.mp3"
	if !autoPush {
		output = stem + "-trimmed-" + id + ".mp3"
	}
	job := ProcessingJob{ID: id, Source: source, Output: output, Stage: "queued", AutoPush: autoPush, Start: start, End: end}
	p.mu.Lock()
	defer p.mu.Unlock()
	if autoPush {
		if _, err := regularFile(filepath.Join(p.cfg.StorageLocation, output)); err == nil {
			return ProcessingJob{}, errors.New("processed MP3 already exists")
		}
		for _, existing := range p.jobs {
			if existing.Source == source && existing.AutoPush && activeProcessingStage(existing.Stage) {
				return existing, nil
			}
		}
	}
	select {
	case p.queue <- id:
		p.jobs[id] = job
		return job, nil
	default:
		return ProcessingJob{}, ErrQueueFull
	}
}

func (p *RecordingProcessor) update(id, stage string, progress float64, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	job := p.jobs[id]
	if job.Stage == "cancelled" {
		return
	}
	if stage == "encoding" && job.Stage != "encoding" {
		job.ProcessedSeconds = 0
	}
	job.Stage = stage
	job.Progress = progress
	if err != nil {
		job.Error = err.Error()
	}
	p.jobs[id] = job
}

func activeProcessingStage(stage string) bool {
	return stage == "queued" || stage == "analyzing" || stage == "encoding" || stage == "pushing"
}

func (p *RecordingProcessor) Cancel(id string) error {
	p.mu.Lock()
	job, ok := p.jobs[id]
	if !ok || !activeProcessingStage(job.Stage) {
		p.mu.Unlock()
		return ErrJobNotCancellable
	}
	job.Stage = "cancelled"
	job.Error = "Processing stopped by user"
	p.jobs[id] = job
	cancel := p.cancels[id]
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	p.notifyUpdate()
	slog.Info("recording processing cancelled", "file", job.Source, "job", id)
	return nil
}

func (p *RecordingProcessor) work() {
	for id := range p.queue {
		p.mu.Lock()
		job := p.jobs[id]
		if job.Stage == "cancelled" {
			p.mu.Unlock()
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		p.cancels[id] = cancel
		p.mu.Unlock()
		err := p.processWithContext(ctx, job)
		p.mu.Lock()
		delete(p.cancels, id)
		cancelled := p.jobs[id].Stage == "cancelled" || ctx.Err() != nil
		p.mu.Unlock()
		cancel()
		if cancelled {
			continue
		}
		if err != nil {
			p.update(id, "failed", 0, err)
			slog.Error("recording processing failed", "file", job.Source, "job", id, "error", err)
			p.notifyUpdate()
		} else {
			p.update(id, "completed", 100, nil)
			slog.Info("recording processing complete", "file", job.Source, "output", job.Output, "job", id)
			p.notifyUpdate()
		}
	}
}

func probeDuration(path string) (float64, error) {
	return probeDurationWithContext(context.Background(), path)
}

func probeDurationWithContext(parent context.Context, path string) (float64, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w: %s", err, strings.TrimSpace(string(out)))
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil || duration <= 0 || math.IsNaN(duration) || math.IsInf(duration, 0) {
		return 0, errors.New("invalid recording duration")
	}
	return duration, nil
}

func ValidateAudioFile(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "a:0", "-show_entries", "stream=codec_type", "-of", "csv=p=0", path).CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "audio" {
		return errors.New("no readable audio stream")
	}
	_, err = probeDuration(path)
	return err
}

func (p *RecordingProcessor) duration(name string, info os.FileInfo) (float64, error) {
	path := filepath.Join(p.cfg.StorageLocation, name)
	if info == nil {
		var err error
		info, err = regularFile(path)
		if err != nil {
			return 0, err
		}
	}
	if strings.EqualFold(filepath.Ext(name), ".wav") {
		if strings.HasPrefix(name, "rec_") {
			return wavDuration(path), nil
		}
	}
	p.mu.RLock()
	cached, ok := p.info[name]
	p.mu.RUnlock()
	if ok && cached.size == info.Size() && cached.modTime.Equal(info.ModTime()) {
		return cached.duration, nil
	}
	value, err := probeDuration(path)
	if err != nil {
		return 0, err
	}
	p.mu.Lock()
	if p.info != nil {
		p.info[name] = cachedDuration{info.Size(), info.ModTime(), value}
	}
	p.mu.Unlock()
	return value, nil
}

var silenceStart = regexp.MustCompile(`silence_start: ([0-9.]+)`)
var silenceEnd = regexp.MustCompile(`silence_end: ([0-9.]+)`)
var exportName = regexp.MustCompile(`^(.+)-(?:processed|trimmed-[0-9]+)\.mp3$`)
var importedName = regexp.MustCompile(`^import-[0-9]+-(.+)$`)

func displayName(name string) string {
	if match := importedName.FindStringSubmatch(name); match != nil {
		return match[1]
	}
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func detectEdgeTrim(path string, duration float64) (float64, float64, error) {
	return detectEdgeTrimWithProgress(path, duration, nil)
}

func parseFFmpegTime(line string) (float64, bool) {
	if !strings.HasPrefix(line, "out_time=") {
		return 0, false
	}
	parts := strings.Split(strings.TrimPrefix(line, "out_time="), ":")
	if len(parts) != 3 {
		return 0, false
	}
	hours, hErr := strconv.ParseFloat(parts[0], 64)
	minutes, mErr := strconv.ParseFloat(parts[1], 64)
	seconds, sErr := strconv.ParseFloat(parts[2], 64)
	if hErr != nil || mErr != nil || sErr != nil {
		return 0, false
	}
	return hours*3600 + minutes*60 + seconds, true
}

func detectEdgeTrimWithProgress(path string, duration float64, progress func(float64)) (float64, float64, error) {
	return detectEdgeTrimWithContext(context.Background(), path, duration, progress)
}

func parseSilenceEdges(stderr string, duration float64) (float64, float64) {
	start, end := 0.0, duration
	var pending *float64
	for _, line := range strings.Split(stderr, "\n") {
		if m := silenceStart.FindStringSubmatch(line); m != nil {
			value, _ := strconv.ParseFloat(m[1], 64)
			pending = &value
		}
		if m := silenceEnd.FindStringSubmatch(line); m != nil && pending != nil {
			value, _ := strconv.ParseFloat(m[1], 64)
			if *pending <= 0.05 && value >= 1 && value <= 300 {
				start = value
			}
			if duration-value <= 0.05 && value-*pending >= 2 && value-*pending <= 300 {
				end = *pending
			}
			pending = nil
		}
	}
	if pending != nil && duration-*pending >= 2 && duration-*pending <= 300 {
		end = *pending
	}
	if end-start < 0.5 {
		return 0, duration
	}
	return start, end
}

func detectEdgeTrimWithContext(parent context.Context, path string, duration float64, progress func(float64)) (float64, float64, error) {
	ctx, cancel := context.WithTimeout(parent, 4*time.Hour)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-nostdin", "-i", path,
		"-af", "silencedetect=noise=-60dB:d=1", "-progress", "pipe:1", "-nostats", "-f", "null", "-")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, duration, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return 0, duration, fmt.Errorf("start silence analysis: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if elapsed, ok := parseFFmpegTime(scanner.Text()); ok && progress != nil {
			progress(math.Min(duration, elapsed))
		}
	}
	scanErr := scanner.Err()
	waitErr := cmd.Wait()
	if err := parent.Err(); err != nil {
		return 0, duration, err
	}
	if scanErr != nil {
		return 0, duration, fmt.Errorf("read silence analysis progress: %w", scanErr)
	}
	if waitErr != nil {
		return 0, duration, fmt.Errorf("silence analysis: %w: %s", waitErr, tail(stderr.Bytes()))
	}
	start, end := parseSilenceEdges(stderr.String(), duration)
	return start, end, nil
}

func tail(data []byte) string {
	if len(data) > 2048 {
		data = data[len(data)-2048:]
	}
	return strings.TrimSpace(string(data))
}

func (p *RecordingProcessor) process(job ProcessingJob) error {
	return p.processWithContext(context.Background(), job)
}

func (p *RecordingProcessor) resolveTrimBounds(ctx context.Context, job ProcessingJob, sourcePath string, duration float64) (float64, float64, error) {
	start, end := job.Start, job.End
	if job.AutoPush {
		p.mu.Lock()
		current := p.jobs[job.ID]
		current.TotalSeconds = duration
		p.jobs[job.ID] = current
		p.mu.Unlock()
		var err error
		start, end, err = detectEdgeTrimWithContext(ctx, sourcePath, duration, func(elapsed float64) {
			p.mu.Lock()
			current := p.jobs[job.ID]
			current.ProcessedSeconds = elapsed
			current.Progress = math.Min(99, math.Max(0, elapsed/duration*100))
			p.jobs[job.ID] = current
			p.mu.Unlock()
		})
		if err != nil {
			return 0, 0, err
		}
	} else if end == 0 {
		end = duration
	}
	if end > duration+0.05 || start >= duration || end-start < 0.5 {
		return 0, 0, errors.New("trim range is outside recording or shorter than 0.5 seconds")
	}
	return start, end, nil
}

func (p *RecordingProcessor) trackEncodeProgress(stdout io.Reader, job ProcessingJob, total float64) error {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if elapsed, ok := parseFFmpegTime(line); ok {
			p.mu.Lock()
			current := p.jobs[job.ID]
			current.ProcessedSeconds = math.Min(total, math.Max(0, elapsed))
			current.Progress = math.Min(99, math.Max(0, elapsed/total*100))
			p.jobs[job.ID] = current
			p.mu.Unlock()
		}
	}
	return scanner.Err()
}

func (p *RecordingProcessor) executeFFmpegEncode(ctx context.Context, job ProcessingJob, sourcePath, tmpPath string, start, end float64) error {
	encodeCtx, cancel := context.WithTimeout(ctx, 4*time.Hour)
	defer cancel()
	filter := fmt.Sprintf("atrim=start=%.3f:end=%.3f,asetpts=PTS-STARTPTS", start, end)
	if job.AutoPush {
		filter += ",highpass=f=80,dynaudnorm=f=150:g=15:p=0.95:m=10,loudnorm=I=-16:LRA=11:TP=-1.5,alimiter=limit=0.95"
	}
	args := []string{"-hide_banner", "-nostdin", "-y", "-i", sourcePath, "-vn", "-af", filter,
		"-ar", "44100", "-ac", "1", "-codec:a", "libmp3lame", "-b:a", "96k",
		"-progress", "pipe:1", "-nostats", "-f", "mp3", tmpPath}
	cmd := exec.CommandContext(encodeCtx, "ffmpeg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}
	p.update(job.ID, "encoding", 0, nil)
	p.mu.Lock()
	current := p.jobs[job.ID]
	current.TotalSeconds = end - start
	p.jobs[job.ID] = current
	p.mu.Unlock()
	scanErr := p.trackEncodeProgress(stdout, job, end-start)
	waitErr := cmd.Wait()
	if err := ctx.Err(); err != nil {
		return err
	}
	if scanErr != nil {
		return fmt.Errorf("read ffmpeg progress: %w", scanErr)
	}
	if waitErr != nil {
		return fmt.Errorf("ffmpeg: %w: %s", waitErr, tail(stderr.Bytes()))
	}
	return nil
}

func finalizeMP3File(tmpPath, outputPath string) error {
	info, err := regularFile(tmpPath)
	if err != nil || info.Size() == 0 {
		return errors.New("ffmpeg produced no MP3")
	}
	if err := os.Chmod(tmpPath, 0644); err != nil {
		return fmt.Errorf("set MP3 permissions: %w", err)
	}
	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("publish MP3: %w", err)
	}
	return nil
}

func (p *RecordingProcessor) resolveProcessingSource(job ProcessingJob) (string, error) {
	inputName := job.Source
	if !job.AutoPush {
		inputName = outputStem(job.Source) + "-processed.mp3"
	}
	sourcePath := filepath.Join(p.cfg.StorageLocation, inputName)
	if _, err := regularFile(sourcePath); err != nil {
		return "", err
	}
	return sourcePath, nil
}

func (p *RecordingProcessor) cleanupCancelledExport(ctx context.Context, output, outputPath string) {
	if ctx.Err() != nil {
		if err := os.Remove(outputPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("could not remove cancelled export", "file", output, "error", err)
		}
		if err := p.releaseUnusedCloudName(output); err != nil {
			slog.Warn("could not release cancelled cloud name", "file", output, "error", err)
		}
	}
}

func (p *RecordingProcessor) processWithContext(ctx context.Context, job ProcessingJob) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sourcePath, err := p.resolveProcessingSource(job)
	if err != nil {
		return err
	}
	p.update(job.ID, "analyzing", 0, nil)
	duration, err := probeDurationWithContext(ctx, sourcePath)
	if err != nil {
		return err
	}
	start, end, err := p.resolveTrimBounds(ctx, job, sourcePath, duration)
	if err != nil {
		return err
	}
	outputPath := filepath.Join(p.cfg.StorageLocation, job.Output)
	defer p.cleanupCancelledExport(ctx, job.Output, outputPath)

	tmp, err := os.CreateTemp(p.cfg.StorageLocation, ".abel-processing-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	if err := p.executeFFmpegEncode(ctx, job, sourcePath, tmpPath, start, end); err != nil {
		return err
	}
	if err := finalizeMP3File(tmpPath, outputPath); err != nil {
		return err
	}
	p.update(job.ID, "pushing", 100, nil)
	return p.pushWithContext(ctx, job.Output)
}

func (p *RecordingProcessor) Push(name string) error {
	err := p.pushWithContext(context.Background(), name)
	if err == nil {
		p.notifyUpdate()
	}
	return err
}

func streamToFile(ctx context.Context, src io.Reader, tmp *os.File) error {
	buffer := make([]byte, 1024*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := src.Read(buffer)
		if n > 0 {
			if _, err := tmp.Write(buffer[:n]); err != nil {
				return fmt.Errorf("copy audio: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read audio: %w", readErr)
		}
	}
	return nil
}

func copyToCloudLocation(ctx context.Context, srcPath, cloudDir, target string) error {
	if err := os.MkdirAll(cloudDir, 0755); err != nil {
		return fmt.Errorf("create cloud directory: %w", err)
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	tmp, err := os.CreateTemp(cloudDir, ".abel-push-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := streamToFile(ctx, src, tmp); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), filepath.Join(cloudDir, target)); err != nil {
		return fmt.Errorf("publish cloud copy: %w", err)
	}
	return nil
}

func (p *RecordingProcessor) pushWithContext(ctx context.Context, name string) error {
	if !ValidAudioName(name) {
		return ErrInvalidExport
	}
	source := name
	if mapped, err := p.sourceForExport(name); err == nil {
		source = mapped
	} else {
		if _, sourceErr := regularFile(filepath.Join(p.cfg.StorageLocation, name)); sourceErr != nil {
			return fmt.Errorf("recording unavailable: %w", sourceErr)
		}
	}
	srcPath := filepath.Join(p.cfg.StorageLocation, name)
	if _, err := regularFile(srcPath); err != nil {
		return fmt.Errorf("audio file unavailable: %w", err)
	}
	sourceInfo, err := regularFile(filepath.Join(p.cfg.StorageLocation, source))
	if err != nil {
		return fmt.Errorf("source recording unavailable: %w", err)
	}
	target, err := p.cloudNameFor(source, name, sourceInfo.ModTime(), true)
	if err != nil {
		return err
	}
	if err := copyToCloudLocation(ctx, srcPath, p.cfg.CloudDriveLocation, target); err != nil {
		return err
	}
	slog.Info("audio pushed to cloud", "file", name, "target", target, "cloud", p.cfg.CloudDriveLocation)
	return nil
}

func (p *RecordingProcessor) sourceForExport(name string) (string, error) {
	matches := exportName.FindStringSubmatch(name)
	if matches == nil {
		return "", ErrInvalidExport
	}
	for _, candidate := range []string{matches[1] + ".wav", matches[1] + ".WAV", matches[1]} {
		if ValidAudioName(candidate) {
			if _, err := regularFile(filepath.Join(p.cfg.StorageLocation, candidate)); err == nil {
				return candidate, nil
			}
		}
	}
	files, err := os.ReadDir(p.cfg.StorageLocation)
	if err == nil {
		for _, file := range files {
			if ValidAudioName(file.Name()) && outputStem(file.Name()) == matches[1] {
				if _, err := regularFile(filepath.Join(p.cfg.StorageLocation, file.Name())); err == nil {
					return file.Name(), nil
				}
			}
		}
	}
	return "", fmt.Errorf("source recording unavailable: %w", os.ErrNotExist)
}

func wavDuration(path string) float64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	header := make([]byte, 44)
	if _, err := io.ReadFull(f, header); err != nil || string(header[:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return 0
	}
	byteRate := binary.LittleEndian.Uint32(header[28:32])
	if byteRate == 0 {
		return 0
	}
	return float64(binary.LittleEndian.Uint32(header[40:44])) / float64(byteRate)
}

func (p *RecordingProcessor) scanLibraryFiles(dir []os.DirEntry) ([]RecordingEntry, []RecordingExport, error) {
	entries := make([]RecordingEntry, 0)
	exports := make([]RecordingExport, 0)
	for _, file := range dir {
		if !file.Type().IsRegular() {
			continue
		}
		info, err := file.Info()
		if err != nil {
			continue
		}
		_, isExport := p.sourceForExport(file.Name())
		if ValidAudioName(file.Name()) && isExport != nil {
			duration, _ := p.duration(file.Name(), info)
			rawName, err := p.cloudNameFor(file.Name(), file.Name(), info.ModTime(), false)
			if err != nil {
				return nil, nil, err
			}
			rawCloudPath, rawPushed := cloudCopyPath(p.cfg.CloudDriveLocation,
				rawName, file.Name(), info.Size(), info.ModTime())
			entries = append(entries, RecordingEntry{
				Name:         file.Name(),
				Display:      displayName(file.Name()),
				Size:         info.Size(),
				ModTime:      info.ModTime(),
				Duration:     duration,
				RawCloudPath: rawCloudPath,
				RawPushed:    rawPushed,
				Exports:      []RecordingExport{},
				Jobs:         []ProcessingJob{},
			})
		} else if validFileName(file.Name(), ".mp3") {
			duration, _ := p.duration(file.Name(), info)
			exports = append(exports, RecordingExport{Name: file.Name(), Size: info.Size(), ModTime: info.ModTime(), Duration: duration})
		}
	}
	return entries, exports, nil
}

func (p *RecordingProcessor) populateExportsAndJobs(entries []RecordingEntry, exports []RecordingExport) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for i := range entries {
		stem := outputStem(entries[i].Name)
		for _, export := range exports {
			if export.Name == stem+"-processed.mp3" || strings.HasPrefix(export.Name, stem+"-trimmed-") {
				cloudName, err := p.cloudNameFor(entries[i].Name, export.Name, entries[i].ModTime, false)
				if err != nil {
					return err
				}
				path, pushed := cloudCopyPath(p.cfg.CloudDriveLocation, cloudName, export.Name, export.Size, export.ModTime)
				export.CloudPath = path
				export.CloudTargetPath = filepath.Join(p.cfg.CloudDriveLocation, cloudName)
				export.Pushed = pushed
				entries[i].Exports = append(entries[i].Exports, export)
			}
		}
		for _, job := range p.jobs {
			if job.Source == entries[i].Name {
				entries[i].Jobs = append(entries[i].Jobs, job)
			}
		}
		sort.Slice(entries[i].Exports, func(a, b int) bool { return entries[i].Exports[a].ModTime.After(entries[i].Exports[b].ModTime) })
		sort.Slice(entries[i].Jobs, func(a, b int) bool { return entries[i].Jobs[a].ID < entries[i].Jobs[b].ID })
	}
	return nil
}

func (p *RecordingProcessor) Library() ([]RecordingEntry, error) {
	dir, err := os.ReadDir(p.cfg.StorageLocation)
	if err != nil {
		if os.IsNotExist(err) {
			return []RecordingEntry{}, nil
		}
		return nil, err
	}
	entries, exports, err := p.scanLibraryFiles(dir)
	if err != nil {
		return nil, err
	}
	if err := p.populateExportsAndJobs(entries, exports); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(a, b int) bool { return entries[a].ModTime.After(entries[b].ModTime) })
	return entries, nil
}
