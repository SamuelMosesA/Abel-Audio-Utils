package web

import (
	"abel/src/backend/lib/audioengine"
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
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
	if err := audioengine.WritePlaceholderHeader(f, 2, rate); err != nil {
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
	if err := audioengine.FinalizeWavHeader(f, 2, frames, rate); err != nil {
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
		t.Fatal(err)
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
	if _, err := p.enqueue("../"+name, 0, 0, true); err == nil {
		t.Error("accepted traversal")
	}
	if _, err := p.enqueue(name, 2, 1, false); err == nil {
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
	if _, err := p.enqueue(name, 0, 1, false); err == nil {
		t.Error("accepted trim before processed MP3 exists")
	}
}

func TestRecordingLibraryAndProcessingRequireAuthentication(t *testing.T) {
	dir := t.TempDir()
	name := testRecording(t, dir)
	cfg := &config.Config{StorageLocation: dir, CloudDriveLocation: filepath.Join(dir, "cloud")}
	router := setupTestRouter(state.NewAppState(dir, cfg.CloudDriveLocation), cfg)

	for _, path := range []string{"/api/recordings/library", "/api/recordings/process", "/api/recordings/process/cancel"} {
		method := http.MethodGet
		if path != "/api/recordings/library" {
			method = http.MethodPost
		}
		req := httptest.NewRequest(method, path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s without session returned %d", path, response.Code)
		}
	}

	body, _ := json.Marshal(map[string]interface{}{"source": name, "startSeconds": 4, "endSeconds": 2})
	req := httptest.NewRequest(http.MethodPost, "/api/recordings/process", bytes.NewReader(body))
	req.Header.Set("X-Test-Auth", "true")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid trim returned %d: %s", response.Code, response.Body.String())
	}
}

func TestUploadAudioStartsAutomaticProcessing(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	inputDir := t.TempDir()
	wav := filepath.Join(inputDir, testRecording(t, inputDir))
	mp3 := filepath.Join(inputDir, "guest-sermon.mp3")
	if out, err := exec.Command("ffmpeg", "-v", "error", "-i", wav, "-codec:a", "libmp3lame", mp3).CombinedOutput(); err != nil {
		t.Fatalf("prepare MP3: %v: %s", err, out)
	}
	storage := t.TempDir()
	cfg := &config.Config{StorageLocation: storage, CloudDriveLocation: filepath.Join(storage, "cloud")}
	processor := NewRecordingProcessor(cfg)
	router := setupTestRouterWithProcessor(state.NewAppState(storage, cfg.CloudDriveLocation), cfg, processor)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "guest-sermon.mp3")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(mp3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, f); err != nil {
		t.Fatal(err)
	}
	f.Close()
	writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/recordings/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Test-Auth", "true")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload returned %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		File  string `json:"file"`
		JobID string `json:"jobId"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || payload.JobID == "" {
		t.Fatalf("upload response: %s: %v", response.Body.String(), err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		processor.mu.RLock()
		job := processor.jobs[payload.JobID]
		processor.mu.RUnlock()
		if job.Stage == "completed" {
			sourceInfo, err := os.Stat(filepath.Join(storage, payload.File))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(cfg.CloudDriveLocation, cloudFilename(payload.File, job.Output, sourceInfo.ModTime()))); err != nil {
				t.Fatal(err)
			}
			break
		}
		if job.Stage == "failed" || time.Now().After(deadline) {
			t.Fatalf("import job ended in %s: %s", job.Stage, job.Error)
		}
		time.Sleep(20 * time.Millisecond)
	}
	library, err := processor.Library()
	if err != nil || len(library) != 1 || library[0].Display != "guest-sermon.mp3" {
		t.Fatalf("import library: %+v, %v", library, err)
	}
}

func TestUploadRejectsUnreadableAudioAndRequiresSession(t *testing.T) {
	storage := t.TempDir()
	cfg := &config.Config{StorageLocation: storage, CloudDriveLocation: filepath.Join(storage, "cloud")}
	router := setupTestRouter(state.NewAppState(storage, cfg.CloudDriveLocation), cfg)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "broken.wav")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("not audio"))
	writer.Close()
	request := func(auth bool) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/api/recordings/upload", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		if auth {
			req.Header.Set("X-Test-Auth", "true")
		}
		return req
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request(false))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated upload returned %d", response.Code)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request(true))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unreadable audio returned %d: %s", response.Code, response.Body.String())
	}
	library, err := os.ReadDir(storage)
	if err != nil || len(library) != 0 {
		t.Fatalf("invalid upload left files: %+v, %v", library, err)
	}
}

func TestCloudFilenameUsesReadableRecordingDateAndTrimSuffix(t *testing.T) {
	source := "import-123-R_20260830-112511.wav"
	processed := cloudFilename(source, "import-123-R_20260830-112511-processed.mp3", time.Now())
	trimmed := cloudFilename(source, "import-123-R_20260830-112511-trimmed-123.mp3", time.Now())
	if processed != "2026-08-30_11-25-11.mp3" {
		t.Fatalf("unexpected processed cloud name: %s", processed)
	}
	if trimmed != "2026-08-30_11-25-11-trimmed.mp3" {
		t.Fatalf("unexpected trimmed cloud name: %s", trimmed)
	}
	if processed != cloudFilename(source, "import-123-R_20260830-112511-processed.mp3", time.Now()) {
		t.Fatal("cloud name changed between calls")
	}
	if raw := cloudFilename("sermon-trimmed-notes.wav", "sermon-trimmed-notes.wav", time.Now()); !strings.HasSuffix(raw, "-original.wav") {
		t.Fatalf("raw recording got an MP3 name: %s", raw)
	}
}

func TestCancelQueuedAndRunningProcessing(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe unavailable")
	}
	dir := t.TempDir()
	source := testRecording(t, dir)
	p := NewRecordingProcessor(&config.Config{StorageLocation: dir, CloudDriveLocation: filepath.Join(dir, "cloud")})
	job, err := p.enqueue(source, 0, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		p.mu.RLock()
		current := p.jobs[job.ID]
		p.mu.RUnlock()
		if current.Stage == "analyzing" || current.Stage == "encoding" {
			break
		}
		if current.Stage == "completed" || current.Stage == "failed" || time.Now().After(deadline) {
			t.Fatalf("job did not become cancellable: %s", current.Stage)
		}
		time.Sleep(time.Millisecond)
	}
	if err := p.Cancel(job.ID); err != nil {
		t.Fatal(err)
	}
	if err := p.Cancel(job.ID); !errors.Is(err, errJobNotCancellable) {
		t.Fatalf("second cancel: %v", err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for {
		p.mu.RLock()
		current := p.jobs[job.ID]
		_, running := p.cancels[job.ID]
		p.mu.RUnlock()
		if !running {
			if current.Stage != "cancelled" {
				t.Fatalf("job stage after cancellation: %s", current.Stage)
			}
			if _, err := os.Stat(filepath.Join(dir, job.Output)); !os.IsNotExist(err) {
				t.Fatalf("cancelled job left output: %v", err)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("cancelled job did not stop")
		}
		time.Sleep(time.Millisecond)
	}
	queued := &RecordingProcessor{cfg: p.cfg, jobs: map[string]ProcessingJob{"queued": {ID: "queued", Stage: "queued"}}, queue: make(chan string, 1)}
	if err := queued.Cancel("queued"); err != nil {
		t.Fatal(err)
	}
	if queued.jobs["queued"].Stage != "cancelled" {
		t.Fatal("queued job was not cancelled")
	}
}

func TestCloudNameNumberingPersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	cloud := filepath.Join(dir, "cloud")
	cfg := &config.Config{StorageLocation: dir, CloudDriveLocation: cloud}
	p := &RecordingProcessor{cfg: cfg}
	date := time.Now()
	first, err := p.cloudNameFor("import-1-R_20260830-112511.wav", "import-1-R_20260830-112511-processed.mp3", date, true)
	if err != nil {
		t.Fatal(err)
	}
	second, err := p.cloudNameFor("import-2-R_20260830-112511.wav", "import-2-R_20260830-112511-processed.mp3", date, true)
	if err != nil {
		t.Fatal(err)
	}
	if first != "2026-08-30_11-25-11.mp3" || second != "2026-08-30_11-25-11-2.mp3" {
		t.Fatalf("unexpected numbered names: %s, %s", first, second)
	}
	restarted := &RecordingProcessor{cfg: cfg}
	same, err := restarted.cloudNameFor("import-2-R_20260830-112511.wav", "import-2-R_20260830-112511-processed.mp3", date, true)
	if err != nil || same != second {
		t.Fatalf("reserved name changed after restart: %s, %v", same, err)
	}
	third, err := restarted.cloudNameFor("import-3-R_20260830-112511.wav", "import-3-R_20260830-112511-processed.mp3", date, true)
	if err != nil || third != "2026-08-30_11-25-11-3.mp3" {
		t.Fatalf("third name: %s, %v", third, err)
	}
	trimmed, err := restarted.cloudNameFor("import-1-R_20260830-112511.wav", "import-1-R_20260830-112511-trimmed-123.mp3", date, true)
	if err != nil || trimmed != "2026-08-30_11-25-11-trimmed.mp3" {
		t.Fatalf("trimmed name: %s, %v", trimmed, err)
	}
	anotherTrim, err := restarted.cloudNameFor("import-1-R_20260830-112511.wav", "import-1-R_20260830-112511-trimmed-456.mp3", date, true)
	if err != nil || anotherTrim != "2026-08-30_11-25-11-trimmed-2.mp3" {
		t.Fatalf("second trimmed name: %s, %v", anotherTrim, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".abel-cloud-names.json"))
	if err != nil || !strings.Contains(string(data), "-trimmed-2.mp3") {
		t.Fatalf("name manifest missing: %v", err)
	}
}

func TestCloudNameSkipsExistingFileWithoutReservation(t *testing.T) {
	dir := t.TempDir()
	cloud := filepath.Join(dir, "cloud")
	if err := os.Mkdir(cloud, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cloud, "2026-08-30_11-25-11.mp3"), []byte("another recording"), 0644); err != nil {
		t.Fatal(err)
	}
	p := &RecordingProcessor{cfg: &config.Config{StorageLocation: dir, CloudDriveLocation: cloud}}
	target, err := p.cloudNameFor("R_20260830-112511.wav", "R_20260830-112511-processed.mp3", time.Now(), true)
	if err != nil || target != "2026-08-30_11-25-11-2.mp3" {
		t.Fatalf("collision target: %s, %v", target, err)
	}
}

func TestCancelledExportReleasesUnusedCloudName(t *testing.T) {
	dir := t.TempDir()
	cloud := filepath.Join(dir, "cloud")
	p := &RecordingProcessor{cfg: &config.Config{StorageLocation: dir, CloudDriveLocation: cloud}}
	source := "R_20260830-112511.wav"
	first := "R_20260830-112511-trimmed-first.mp3"
	second := "R_20260830-112511-trimmed-second.mp3"
	assigned, err := p.cloudNameFor(source, first, time.Now(), true)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.releaseUnusedCloudName(first); err != nil {
		t.Fatal(err)
	}
	reused, err := p.cloudNameFor(source, second, time.Now(), true)
	if err != nil || reused != assigned {
		t.Fatalf("unused cloud name was not released: %s, %v", reused, err)
	}
	if err := os.MkdirAll(cloud, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cloud, reused), []byte("cloud copy"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := p.releaseUnusedCloudName(second); err != nil {
		t.Fatal(err)
	}
	third, err := p.cloudNameFor(source, "R_20260830-112511-trimmed-third.mp3", time.Now(), true)
	if err != nil || third != "2026-08-30_11-25-11-trimmed-2.mp3" {
		t.Fatalf("existing cloud copy lost its reservation: %s, %v", third, err)
	}
}

func TestConcurrentCloudNamesAreUnique(t *testing.T) {
	dir := t.TempDir()
	p := &RecordingProcessor{cfg: &config.Config{StorageLocation: dir, CloudDriveLocation: filepath.Join(dir, "cloud")}}
	const count = 8
	names := make(chan string, count)
	errorsFound := make(chan error, count)
	var group sync.WaitGroup
	for i := 0; i < count; i++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			source := fmt.Sprintf("import-%d-R_20260830-112511.wav", index)
			output := strings.TrimSuffix(source, ".wav") + "-processed.mp3"
			name, err := p.cloudNameFor(source, output, time.Now(), true)
			if err != nil {
				errorsFound <- err
				return
			}
			names <- name
		}(i)
	}
	group.Wait()
	close(names)
	close(errorsFound)
	for err := range errorsFound {
		t.Error(err)
	}
	seen := make(map[string]bool)
	for name := range names {
		if seen[name] {
			t.Errorf("duplicate cloud name: %s", name)
		}
		seen[name] = true
	}
	if len(seen) != count {
		t.Errorf("got %d cloud names, want %d", len(seen), count)
	}
}

func TestReconcileCloudExportsPushesMissingMP3s(t *testing.T) {
	dir := t.TempDir()
	source := testRecording(t, dir)
	cloud := filepath.Join(dir, "cloud")
	output := "rec_test-processed.mp3"
	if err := os.WriteFile(filepath.Join(dir, output), []byte("existing MP3 export"), 0644); err != nil {
		t.Fatal(err)
	}
	p := &RecordingProcessor{cfg: &config.Config{StorageLocation: dir, CloudDriveLocation: cloud}, jobs: map[string]ProcessingJob{
		"1": {ID: "1", Source: source, Output: output, Stage: "cancelled", AutoPush: true},
	}}
	if err := p.reconcileCloudExports(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cloud); !os.IsNotExist(err) {
		t.Fatalf("cancelled export was pushed: %v", err)
	}
	p.jobs["1"] = ProcessingJob{ID: "1", Source: source, Output: output, Stage: "failed", AutoPush: true}
	if err := p.reconcileCloudExports(); err != nil {
		t.Fatal(err)
	}
	sourceInfo, err := os.Stat(filepath.Join(dir, source))
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(cloud, cloudFilename(source, output, sourceInfo.ModTime()))
	if data, err := os.ReadFile(target); err != nil || string(data) != "existing MP3 export" {
		t.Fatalf("cloud export: %q, %v", data, err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := p.reconcileCloudExports(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("retry changed the assigned name: %v", err)
	}
	if _, err := os.Stat(strings.TrimSuffix(target, ".mp3") + "-2.mp3"); !os.IsNotExist(err) {
		t.Fatalf("retry created a second name: %v", err)
	}
}
