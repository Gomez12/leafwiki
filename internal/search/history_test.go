package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureFileHistoryLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "root")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}

	index, err := NewSQLiteIndex(tmpDir)
	if err != nil {
		t.Fatalf("failed to create SQLiteIndex: %v", err)
	}
	defer index.Close()

	original := filepath.Join(dataDir, "note.md")
	writeFile(t, original, "# note")

	mustCapture(t, index, dataDir)
	entries := readHistoryEntries(t, index)
	assertHistory(t, entries[len(entries)-1], "note.md", FileStatusCreated, "")

	// Modify content
	writeFile(t, original, "# note\nupdated")
	mustCapture(t, index, dataDir)
	entries = readHistoryEntries(t, index)
	assertHistory(t, entries[len(entries)-1], "note.md", FileStatusModified, "")

	// Move to new folder without changing content
	moved := filepath.Join(dataDir, "docs", "note.md")
	if err := os.MkdirAll(filepath.Dir(moved), 0o755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}
	if err := os.Rename(original, moved); err != nil {
		t.Fatalf("failed to move file: %v", err)
	}

	mustCapture(t, index, dataDir)
	entries = readHistoryEntries(t, index)
	assertHistory(t, entries[len(entries)-1], "docs/note.md", FileStatusMoved, "note.md")

	// Delete the file
	if err := os.Remove(moved); err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}

	mustCapture(t, index, dataDir)
	entries = readHistoryEntries(t, index)
	assertHistory(t, entries[len(entries)-1], "docs/note.md", FileStatusDeleted, "")
}

type historyRow struct {
	path         string
	status       FileHistoryStatus
	previousPath string
}

func readHistoryEntries(t *testing.T, index *SQLiteIndex) []historyRow {
	t.Helper()

	rows, err := index.GetDB().Query(`SELECT path, status, COALESCE(previous_path, '') FROM file_history ORDER BY id;`)
	if err != nil {
		t.Fatalf("failed to read history: %v", err)
	}
	defer rows.Close()

	var entries []historyRow
	for rows.Next() {
		var row historyRow
		if err := rows.Scan(&row.path, &row.status, &row.previousPath); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}
		entries = append(entries, row)
	}

	return entries
}

func writeFile(t *testing.T, filename, content string) {
	t.Helper()
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

func mustCapture(t *testing.T, index *SQLiteIndex, dataDir string) {
	t.Helper()
	if err := index.CaptureFileHistory(dataDir); err != nil {
		t.Fatalf("capture failed: %v", err)
	}
}

func assertHistory(t *testing.T, row historyRow, expectedPath string, expectedStatus FileHistoryStatus, prev string) {
	t.Helper()
	if row.path != expectedPath {
		t.Fatalf("expected path %s, got %s", expectedPath, row.path)
	}
	if row.status != expectedStatus {
		t.Fatalf("expected status %s, got %s", expectedStatus, row.status)
	}
	if row.previousPath != prev {
		t.Fatalf("expected previous_path %s, got %s", prev, row.previousPath)
	}
}
