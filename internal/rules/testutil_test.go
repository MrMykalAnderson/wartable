package rules

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTempFile writes content to a temp file matching pattern and returns
// its path. The file is removed when the test completes.
func writeTempFile(t *testing.T, pattern, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), pattern)
	if err != nil {
		t.Fatalf("writeTempFile: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writeTempFile: %v", err)
	}
	return filepath.Join(f.Name())
}
