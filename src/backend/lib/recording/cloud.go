package recording

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var datedRecording = regexp.MustCompile(`^R_([0-9]{8})-([0-9]{6})(?:\.|$)`)
var unixRecording = regexp.MustCompile(`^rec_([0-9]+)(?:\.|$)`)

func recordingDate(source string, fallback time.Time) time.Time {
	original := source
	if match := importedName.FindStringSubmatch(source); match != nil {
		original = match[1]
	}
	if match := datedRecording.FindStringSubmatch(original); match != nil {
		if date, err := time.Parse("20060102150405", match[1]+match[2]); err == nil {
			return date
		}
	}
	if match := unixRecording.FindStringSubmatch(original); match != nil {
		if seconds, err := strconv.ParseInt(match[1], 10, 64); err == nil {
			return time.Unix(seconds, 0).Local()
		}
	}
	return fallback.Local()
}

func CloudFilename(source, name string, modTime time.Time) string {
	base := recordingDate(source, modTime).Format("2006-01-02_15-04-05")
	if name == source {
		return base + "-original" + filepath.Ext(name)
	}
	if strings.Contains(name, "-trimmed-") {
		return base + "-trimmed.mp3"
	}
	return base + ".mp3"
}

func cloudFilename(source, name string, modTime time.Time) string {
	return CloudFilename(source, name, modTime)
}

func numberedCloudFilename(base string, number int) string {
	if number == 1 {
		return base
	}
	extension := filepath.Ext(base)
	return strings.TrimSuffix(base, extension) + "-" + strconv.Itoa(number) + extension
}

func (p *RecordingProcessor) loadCloudNamesLocked() error {
	if p.cloudNamesLoaded {
		return nil
	}
	path := filepath.Join(p.cfg.StorageLocation, ".abel-cloud-names.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		p.cloudNames = make(map[string]string)
		p.cloudNamesLoaded = true
		return nil
	}
	if err != nil {
		return fmt.Errorf("read cloud names: %w", err)
	}
	var names map[string]string
	if err := json.Unmarshal(data, &names); err != nil {
		return fmt.Errorf("parse cloud names: %w", err)
	}
	for local, target := range names {
		if !ValidAudioName(local) || !ValidAudioName(target) {
			return errors.New("cloud name manifest contains an invalid filename")
		}
	}
	p.cloudNames = names
	p.cloudNamesLoaded = true
	return nil
}

func (p *RecordingProcessor) saveCloudNamesLocked(names map[string]string) error {
	path := filepath.Join(p.cfg.StorageLocation, ".abel-cloud-names.json")
	data, err := json.MarshalIndent(names, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(p.cfg.StorageLocation, ".abel-cloud-names-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
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
	if err := os.Rename(tmp.Name(), path); err != nil {
		return err
	}
	return nil
}

func (p *RecordingProcessor) findNextCloudFilename(base string, used map[string]bool, reserve bool) (string, error) {
	for number := 1; ; number++ {
		candidate := numberedCloudFilename(base, number)
		if used[candidate] {
			continue
		}
		_, err := os.Lstat(filepath.Join(p.cfg.CloudDriveLocation, candidate))
		if err == nil {
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			if reserve {
				return "", fmt.Errorf("check cloud filename: %w", err)
			}
			return candidate, nil
		}
		return candidate, nil
	}
}

// cloudNameFor reserves a destination before pushing. Reservations survive a
// restart, keeping repeated pushes and library status tied to the same file.
func (p *RecordingProcessor) cloudNameFor(source, name string, modTime time.Time, reserve bool) (string, error) {
	p.cloudMu.Lock()
	defer p.cloudMu.Unlock()
	if err := p.loadCloudNamesLocked(); err != nil {
		return "", err
	}
	if target, ok := p.cloudNames[name]; ok {
		return target, nil
	}
	used := make(map[string]bool, len(p.cloudNames))
	for _, target := range p.cloudNames {
		used[target] = true
	}
	base := cloudFilename(source, name, modTime)
	target, err := p.findNextCloudFilename(base, used, reserve)
	if err != nil {
		return "", err
	}
	if reserve {
		next := make(map[string]string, len(p.cloudNames)+1)
		for local, existing := range p.cloudNames {
			next[local] = existing
		}
		next[name] = target
		if err := p.saveCloudNamesLocked(next); err != nil {
			return "", fmt.Errorf("save cloud filename: %w", err)
		}
		p.cloudNames = next
	}
	return target, nil
}

func (p *RecordingProcessor) releaseUnusedCloudName(name string) error {
	p.cloudMu.Lock()
	defer p.cloudMu.Unlock()
	if err := p.loadCloudNamesLocked(); err != nil {
		return err
	}
	target, ok := p.cloudNames[name]
	if !ok {
		return nil
	}
	if _, err := os.Lstat(filepath.Join(p.cfg.StorageLocation, name)); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check cancelled export: %w", err)
	}
	if _, err := os.Lstat(filepath.Join(p.cfg.CloudDriveLocation, target)); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check cancelled cloud copy: %w", err)
	}
	next := make(map[string]string, len(p.cloudNames)-1)
	for local, existing := range p.cloudNames {
		if local != name {
			next[local] = existing
		}
	}
	if err := p.saveCloudNamesLocked(next); err != nil {
		return err
	}
	p.cloudNames = next
	return nil
}

func cloudCopyPath(cloudDir, target, legacy string, sourceSize int64, sourceModTime time.Time) (string, bool) {
	preferred := filepath.Join(cloudDir, target)
	if info, err := regularFile(preferred); err == nil && info.Size() == sourceSize && !sourceModTime.After(info.ModTime()) {
		return preferred, true
	}
	old := filepath.Join(cloudDir, legacy)
	if info, err := regularFile(old); err == nil && info.Size() == sourceSize && !sourceModTime.After(info.ModTime()) {
		return old, true
	}
	return preferred, false
}
