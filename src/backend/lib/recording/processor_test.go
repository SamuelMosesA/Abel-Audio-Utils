package recording

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"abel/src/backend/lib/audioengine/audio_processing"
	"abel/src/backend/lib/config"
)

func testRecording(t *testing.T, dir string) string {
	t.Helper()
	name := "rec_test.wav"
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	const rate = 44100
	const frames = rate * 6
	if err := audio_processing.WritePlaceholderWavHeader(f, 2, rate); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, frames*4)
	for i := 0; i < frames; i++ {
		seconds := float64(i) / rate
		sample := int16(0)
		if seconds >= 1.2 && seconds < 3.2 {
			sample = int16(math.Sin(seconds*2*math.Pi*440) * 12000)
		}
		binary.LittleEndian.PutUint16(data[i*4:], uint16(sample))
		binary.LittleEndian.PutUint16(data[i*4+2:], uint16(sample))
	}
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
	dataBytes := int64(frames * 2 * audio_processing.WavBytesPerSample)
	if err := audio_processing.FinalizeWavHeader(f, 2, dataBytes, rate); err != nil {
		t.Fatal(err)
	}
	return name
}

func TestRecordingProcessingAndPush(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe unavailable")
	}
	dir := t.TempDir()
	cloud := filepath.Join(dir, "cloud")
	name := testRecording(t, dir)
	p := &RecordingProcessor{cfg: &config.Config{StorageLocation: dir, CloudDriveLocation: cloud}, jobs: map[string]ProcessingJob{}}
	progressSeconds := 0.0
	start, end, err := detectEdgeTrimWithProgress(filepath.Join(dir, name), 6, func(seconds float64) { progressSeconds = seconds })
	if err != nil {
		t.Fatal(err)
	}
	if progressSeconds < 5.9 {
		t.Fatalf("silence analysis did not report processed duration: %.2f", progressSeconds)
	}
	if math.Abs(start-1.2) > 0.1 || math.Abs(end-3.2) > 0.1 {
		t.Fatalf("unexpected automatic trim: %.2f to %.2f", start, end)
	}
	automatic := ProcessingJob{ID: "1", Source: name, Output: "rec_test-processed.mp3", AutoPush: true}
	p.jobs[automatic.ID] = automatic
	if err := p.process(automatic); err != nil {
		t.Fatal(err)
	}
	sourceInfo, err := os.Stat(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	automaticCloud := cloudFilename(name, automatic.Output, sourceInfo.ModTime())
	if _, err := os.Stat(filepath.Join(cloud, automaticCloud)); err != nil {
		t.Fatalf("automatic push: %v", err)
	}
	duration, err := probeDuration(filepath.Join(dir, automatic.Output))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(duration-2) > 0.2 {
		t.Fatalf("automatic duration: %.2f", duration)
	}
	output, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "a:0", "-show_entries", "stream=sample_rate,channels", "-of", "csv=p=0", filepath.Join(dir, automatic.Output)).Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "44100,1\n" {
		t.Fatalf("unexpected MP3 format: %q", output)
	}

	manual := ProcessingJob{ID: "2", Source: name, Output: "rec_test-trimmed-2.mp3", Start: 0.2, End: 1.2}
	p.jobs[manual.ID] = manual
	originalPath := filepath.Join(dir, name)
	backupPath := filepath.Join(dir, "original.backup")
	if err := os.Rename(originalPath, backupPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(originalPath, []byte("not readable audio"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(originalPath, sourceInfo.ModTime(), sourceInfo.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := p.process(manual); err != nil {
		t.Fatalf("manual trim should source the processed MP3 when available: %v", err)
	}
	if err := os.Rename(backupPath, originalPath); err != nil {
		t.Fatal(err)
	}
	manualCloud := cloudFilename(name, manual.Output, sourceInfo.ModTime())
	if _, err := os.Stat(filepath.Join(cloud, manualCloud)); err != nil {
		t.Fatalf("trimmed MP3 was not pushed: %v", err)
	}
	duration, err = probeDuration(filepath.Join(dir, manual.Output))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(duration-1) > 0.2 {
		t.Fatalf("manual duration: %.2f", duration)
	}
	if err := p.Push(manual.Output); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Fatal(err)
	}
	library, err := p.Library()
	if err != nil || len(library) != 1 || len(library[0].Exports) != 2 {
		t.Fatalf("library: %+v, %v", library, err)
	}
	for _, exported := range library[0].Exports {
		if !exported.Pushed {
			t.Errorf("export not marked pushed: %s", exported.Name)
		}
	}
	if err := os.WriteFile(filepath.Join(cloud, manualCloud), []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}
	library, err = p.Library()
	if err != nil {
		t.Fatal(err)
	}
	for _, exported := range library[0].Exports {
		if exported.Name == manual.Output && exported.Pushed {
			t.Error("stale cloud copy marked current")
		}
	}
	oldCopy, err := os.ReadFile(filepath.Join(dir, manual.Output))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cloud, manual.Output), oldCopy, 0644); err != nil {
		t.Fatal(err)
	}
	library, err = p.Library()
	if err != nil {
		t.Fatal(err)
	}
	for _, exported := range library[0].Exports {
		if exported.Name == manual.Output && (!exported.Pushed || exported.CloudPath != filepath.Join(cloud, manual.Output) || exported.CloudTargetPath != filepath.Join(cloud, manualCloud)) {
			t.Errorf("legacy cloud copy was not recognized: %+v", exported)
		}
	}
}

func TestProcessingRejectsUnsafeInputAndInvalidRanges(t *testing.T) {
	dir := t.TempDir()
	name := testRecording(t, dir)
	p := NewRecordingProcessor(&config.Config{StorageLocation: dir, CloudDriveLocation: filepath.Join(dir, "cloud")})
	if _, err := p.Enqueue("../"+name, 0, 0, true); err == nil {
		t.Error("accepted traversal")
	}
	if _, err := p.Enqueue(name, 2, 1, false); err == nil {
		t.Error("accepted reversed range")
	}
	if err := p.Push("../rec_test-processed.mp3"); err == nil {
		t.Error("accepted unsafe push")
	}
	if err := p.Push(name); err != nil {
		t.Fatalf("original audio could not be pushed: %v", err)
	}
	library, err := p.Library()
	if err != nil || len(library) != 1 || !library[0].RawPushed || !strings.HasSuffix(library[0].RawCloudPath, "-original.wav") {
		t.Fatalf("raw cloud status: %+v, %v", library, err)
	}
	if _, err := p.Enqueue(name, 0, 1, false); err == nil {
		t.Error("accepted trim before processed MP3 exists")
	}
}

func TestCloudFilenameUsesReadableRecordingDateAndTrimSuffix(t *testing.T) {
	recordedAt := time.Date(2026, 4, 12, 10, 30, 0, 0, time.Local)
	name := cloudFilename("R_20260412-103000.wav", "R_20260412-103000-processed.mp3", recordedAt.Add(2*time.Hour))
	if name != "2026-04-12_10-30-00.mp3" {
		t.Fatalf("unexpected cloud filename: %s", name)
	}
	trimmed := cloudFilename("R_20260412-103000.wav", "R_20260412-103000-trimmed-123.mp3", recordedAt.Add(2*time.Hour))
	if trimmed != "2026-04-12_10-30-00-trimmed.mp3" {
		t.Fatalf("unexpected trimmed cloud filename: %s", trimmed)
	}
	original := cloudFilename("R_20260412-103000.wav", "R_20260412-103000.wav", recordedAt.Add(2*time.Hour))
	if original != "2026-04-12_10-30-00-original.wav" {
		t.Fatalf("unexpected original cloud filename: %s", original)
	}
}

func TestCancelQueuedAndRunningProcessing(t *testing.T) {
	dir := t.TempDir()
	p := NewRecordingProcessor(&config.Config{StorageLocation: dir, CloudDriveLocation: filepath.Join(dir, "cloud")})
	queued := &RecordingProcessor{cfg: p.cfg, jobs: map[string]ProcessingJob{"queued": {ID: "queued", Stage: "queued"}}, queue: make(chan string, 1)}
	if err := queued.Cancel("queued"); err != nil {
		t.Fatal(err)
	}
	if queued.jobs["queued"].Stage != "cancelled" {
		t.Fatalf("unexpected stage: %s", queued.jobs["queued"].Stage)
	}
	cancelled := false
	running := &RecordingProcessor{cfg: p.cfg, jobs: map[string]ProcessingJob{"running": {ID: "running", Stage: "encoding"}}, cancels: map[string]context.CancelFunc{
		"running": func() { cancelled = true },
	}}
	if err := running.Cancel("running"); err != nil {
		t.Fatal(err)
	}
	if !cancelled {
		t.Fatal("expected cancel callback to be invoked")
	}
	if err := running.Cancel("missing"); err == nil {
		t.Fatal("expected error for missing job")
	}
}

func TestCloudNameNumberingPersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	cloud := filepath.Join(dir, "cloud")
	if err := os.MkdirAll(cloud, 0755); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{StorageLocation: dir, CloudDriveLocation: cloud}
	p := &RecordingProcessor{cfg: cfg}
	fixedTime := time.Date(2026, 4, 12, 10, 0, 0, 0, time.Local)
	name1, err := p.cloudNameFor("R_20260412-100000.wav", "R_20260412-100000-processed.mp3", fixedTime, true)
	if err != nil {
		t.Fatal(err)
	}
	if name1 != "2026-04-12_10-00-00.mp3" {
		t.Fatalf("unexpected first name: %s", name1)
	}
	name2, err := p.cloudNameFor("R_20260412-100000-other.wav", "R_20260412-100000-other-processed.mp3", fixedTime, true)
	if err != nil {
		t.Fatal(err)
	}
	if name2 != "2026-04-12_10-00-00-2.mp3" {
		t.Fatalf("unexpected second name: %s", name2)
	}

	restarted := &RecordingProcessor{cfg: cfg}
	repeated, err := restarted.cloudNameFor("R_20260412-100000-other.wav", "R_20260412-100000-other-processed.mp3", fixedTime, true)
	if err != nil {
		t.Fatal(err)
	}
	if repeated != "2026-04-12_10-00-00-2.mp3" {
		t.Fatalf("restarted processor must reuse preserved reservation: %s", repeated)
	}
	name3, err := restarted.cloudNameFor("R_20260412-100000-third.wav", "R_20260412-100000-third-processed.mp3", fixedTime, true)
	if err != nil {
		t.Fatal(err)
	}
	if name3 != "2026-04-12_10-00-00-3.mp3" {
		t.Fatalf("unexpected third name: %s", name3)
	}
}

func TestCloudNameSkipsExistingFileWithoutReservation(t *testing.T) {
	dir := t.TempDir()
	cloud := filepath.Join(dir, "cloud")
	if err := os.MkdirAll(cloud, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cloud, "2026-04-12_10-00-00.mp3"), []byte("manual upload"), 0644); err != nil {
		t.Fatal(err)
	}
	p := &RecordingProcessor{cfg: &config.Config{StorageLocation: dir, CloudDriveLocation: cloud}}
	fixedTime := time.Date(2026, 4, 12, 10, 0, 0, 0, time.Local)
	name, err := p.cloudNameFor("R_20260412-100000.wav", "R_20260412-100000-processed.mp3", fixedTime, true)
	if err != nil {
		t.Fatal(err)
	}
	if name != "2026-04-12_10-00-00-2.mp3" {
		t.Fatalf("expected next available suffix: %s", name)
	}
}

func TestCancelledExportReleasesUnusedCloudName(t *testing.T) {
	dir := t.TempDir()
	cloud := filepath.Join(dir, "cloud")
	p := &RecordingProcessor{cfg: &config.Config{StorageLocation: dir, CloudDriveLocation: cloud}}
	fixedTime := time.Date(2026, 4, 12, 10, 0, 0, 0, time.Local)
	target, err := p.cloudNameFor("R_20260412-100000.wav", "R_20260412-100000-processed.mp3", fixedTime, true)
	if err != nil {
		t.Fatal(err)
	}
	if target != "2026-04-12_10-00-00.mp3" {
		t.Fatalf("unexpected target: %s", target)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p.cleanupCancelledExport(ctx, "R_20260412-100000-processed.mp3", filepath.Join(dir, "R_20260412-100000-processed.mp3"))

	reused, err := p.cloudNameFor("R_20260412-100000-second.wav", "R_20260412-100000-second-processed.mp3", fixedTime, true)
	if err != nil {
		t.Fatal(err)
	}
	if reused != "2026-04-12_10-00-00.mp3" {
		t.Fatalf("expected cancelled name to be released for reuse: %s", reused)
	}
}

func TestConcurrentCloudNamesAreUnique(t *testing.T) {
	dir := t.TempDir()
	p := &RecordingProcessor{cfg: &config.Config{StorageLocation: dir, CloudDriveLocation: filepath.Join(dir, "cloud")}}
	fixedTime := time.Date(2026, 4, 12, 10, 0, 0, 0, time.Local)
	const count = 12
	results := make([]string, count)
	errs := make([]error, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			source := fmt.Sprintf("R_20260412-100000-%d.wav", idx)
			export := fmt.Sprintf("R_20260412-100000-%d-processed.mp3", idx)
			results[idx], errs[idx] = p.cloudNameFor(source, export, fixedTime, true)
		}(i)
	}
	wg.Wait()
	seen := make(map[string]bool, count)
	for i := 0; i < count; i++ {
		if errs[i] != nil {
			t.Fatalf("cloudNameFor failed: %v", errs[i])
		}
		if seen[results[i]] {
			t.Fatalf("duplicate target assigned: %s", results[i])
		}
		seen[results[i]] = true
	}
}

func TestReconcileCloudExportsPushesMissingMP3s(t *testing.T) {
	dir := t.TempDir()
	cloud := filepath.Join(dir, "cloud")
	p := &RecordingProcessor{cfg: &config.Config{StorageLocation: dir, CloudDriveLocation: cloud}, jobs: map[string]ProcessingJob{
		"failed": {ID: "failed", Output: "rec_1-processed.mp3", Stage: "failed"},
	}}
	const rate = 44100
	const frames = rate * 2
	f, err := os.Create(filepath.Join(dir, "rec_1.wav"))
	if err != nil {
		t.Fatal(err)
	}
	_ = audio_processing.WritePlaceholderWavHeader(f, 2, rate)
	_, _ = f.Write(make([]byte, frames*4))
	_ = audio_processing.FinalizeWavHeader(f, 2, int64(frames*4), rate)
	_ = f.Close()

	if err := os.WriteFile(filepath.Join(dir, "rec_1-processed.mp3"), []byte("mp3-payload"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := p.reconcileCloudExports(); err != nil {
		t.Fatalf("expected reconciliation to succeed: %v", err)
	}
	entries, err := p.Library()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || len(entries[0].Exports) != 1 || !entries[0].Exports[0].Pushed {
		t.Fatalf("expected reconciled export to be pushed: %+v", entries)
	}
	if _, err := os.Stat(entries[0].Exports[0].CloudPath); err != nil {
		t.Fatalf("reconciled cloud export missing: %v", err)
	}
}
