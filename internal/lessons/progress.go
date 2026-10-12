package lessons

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Progress records which lessons the learner has completed.
type Progress struct {
	Completed map[string]bool `json:"completed"`
}

func progressPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gitwise", "progress.json"), nil
}

// LoadProgress reads saved progress; a missing file yields empty progress.
func LoadProgress() *Progress {
	p := &Progress{Completed: map[string]bool{}}
	path, err := progressPath()
	if err != nil {
		return p
	}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, p)
	}
	if p.Completed == nil {
		p.Completed = map[string]bool{}
	}
	return p
}

// MarkDone records a completed lesson and saves progress to disk.
func (p *Progress) MarkDone(id string) error {
	p.Completed[id] = true
	path, err := progressPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
