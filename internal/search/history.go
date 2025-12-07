package search

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FileHistoryStatus string

const (
	FileStatusCreated  FileHistoryStatus = "created"
	FileStatusModified FileHistoryStatus = "modified"
	FileStatusDeleted  FileHistoryStatus = "deleted"
	FileStatusMoved    FileHistoryStatus = "moved"
)

type FileHistorySnapshot struct {
	Path         string
	Hash         string
	Content      string
	Status       FileHistoryStatus
	PreviousPath *string
}

type FileHistoryEntry struct {
	ID           int64             `json:"id"`
	Path         string            `json:"path"`
	Hash         string            `json:"hash"`
	Content      string            `json:"content"`
	Status       FileHistoryStatus `json:"status"`
	PreviousPath *string           `json:"previousPath,omitempty"`
	RecordedAt   time.Time         `json:"recordedAt"`
}

// CaptureFileHistory snapshots all Markdown files under dataDir.
// It records new rows when files are created, modified, removed or moved.
func (s *SQLiteIndex) CaptureFileHistory(dataDir string) error {
	if s.db == nil {
		return sql.ErrConnDone
	}

	currentFiles, err := scanMarkdownFiles(dataDir)
	if err != nil {
		return err
	}

	latest, err := s.latestFileSnapshots()
	if err != nil {
		return err
	}

	aliasPaths := map[string]bool{}
	for _, snap := range latest {
		if snap.PreviousPath != nil {
			aliasPaths[*snap.PreviousPath] = true
		}
	}

	missing := map[string]FileHistorySnapshot{}
	moveCandidates := map[string][]FileHistorySnapshot{}
	for path, snap := range latest {
		if aliasPaths[path] {
			continue
		}
		if _, ok := currentFiles[path]; !ok {
			missing[path] = snap
			if snap.Status != FileStatusDeleted {
				key := movementKey(snap.Hash, path)
				moveCandidates[key] = append(moveCandidates[key], snap)
			}
		}
	}

	for relPath, file := range currentFiles {
		hash := file.Hash
		content := file.Content
		if snap, ok := latest[relPath]; ok {
			if snap.Status == FileStatusDeleted {
				if err := s.insertHistoryEntry(relPath, hash, content, FileStatusCreated, nil); err != nil {
					return err
				}
				log.Printf("[history] recorded created for %s", relPath)
				continue
			}

			if snap.Hash != hash {
				if err := s.insertHistoryEntry(relPath, hash, content, FileStatusModified, nil); err != nil {
					return err
				}
				log.Printf("[history] recorded modified for %s", relPath)
			}
			continue
		}

		key := movementKey(hash, relPath)
		if snaps := moveCandidates[key]; len(snaps) > 0 {
			prev := snaps[0]
			moveCandidates[key] = snaps[1:]
			delete(missing, prev.Path)
			if err := s.insertHistoryEntry(relPath, hash, content, FileStatusMoved, &prev.Path); err != nil {
				return err
			}
			log.Printf("[history] recorded moved from %s to %s", prev.Path, relPath)
			continue
		}

		if err := s.insertHistoryEntry(relPath, hash, content, FileStatusCreated, nil); err != nil {
			return err
		}
		log.Printf("[history] recorded created for %s", relPath)
	}

	for _, snap := range missing {
		if snap.Status == FileStatusDeleted {
			continue
		}
		if err := s.insertHistoryEntry(snap.Path, snap.Hash, snap.Content, FileStatusDeleted, nil); err != nil {
			return err
		}
		log.Printf("[history] recorded deleted for %s", snap.Path)
	}

	return nil
}

func (s *SQLiteIndex) latestFileSnapshots() (map[string]FileHistorySnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		WITH latest AS (
			SELECT MAX(id) AS id, path FROM file_history GROUP BY path
		)
		SELECT fh.path, fh.hash, fh.content, fh.status, fh.previous_path
		FROM file_history fh
		JOIN latest l ON fh.id = l.id;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snapshots := make(map[string]FileHistorySnapshot)

	for rows.Next() {
		var snap FileHistorySnapshot
		var prev sql.NullString
		if err := rows.Scan(&snap.Path, &snap.Hash, &snap.Content, &snap.Status, &prev); err != nil {
			return nil, err
		}
		if prev.Valid {
			snap.PreviousPath = &prev.String
		}
		snapshots[snap.Path] = snap
	}

	return snapshots, rows.Err()
}

func (s *SQLiteIndex) insertHistoryEntry(path string, hash string, content string, status FileHistoryStatus, previousPath *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var prev interface{}
	if previousPath != nil {
		prev = *previousPath
	}

	_, err := s.db.Exec(`
		INSERT INTO file_history (path, hash, content, status, previous_path)
		VALUES (?, ?, ?, ?, ?);
	`, path, hash, content, status, prev)

	return err
}

// GetHistoryForPath returns history rows for the given path, following previous paths (moves).
func (s *SQLiteIndex) GetHistoryForPath(path string) ([]FileHistoryEntry, error) {
	if s.db == nil {
		return nil, sql.ErrConnDone
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	visited := map[string]bool{}
	queue := seedHistoryPaths(path)
	seenIDs := map[int64]bool{}
	var entries []FileHistoryEntry

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true

		rows, err := s.db.Query(`
			SELECT id, path, hash, content, status, previous_path, recorded_at
			FROM file_history
			WHERE path = ? OR previous_path = ?
			ORDER BY recorded_at DESC, id DESC;
		`, current, current)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var entry FileHistoryEntry
			var prev sql.NullString
			var recordedAt string
			if err := rows.Scan(&entry.ID, &entry.Path, &entry.Hash, &entry.Content, &entry.Status, &prev, &recordedAt); err != nil {
				rows.Close()
				return nil, err
			}
			if prev.Valid {
				entry.PreviousPath = &prev.String
				if !visited[prev.String] {
					queue = append(queue, prev.String)
				}
			}

			entry.RecordedAt = parseSQLiteTimestamp(recordedAt)

			if seenIDs[entry.ID] {
				continue
			}
			seenIDs[entry.ID] = true
			entries = append(entries, entry)
		}

		if err := rows.Close(); err != nil {
			return nil, err
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].RecordedAt.Equal(entries[j].RecordedAt) {
			return entries[i].ID > entries[j].ID
		}
		return entries[i].RecordedAt.After(entries[j].RecordedAt)
	})

	return entries, nil
}

type fileRecord struct {
	Hash    string
	Content string
}

func scanMarkdownFiles(dataDir string) (map[string]fileRecord, error) {
	current := make(map[string]fileRecord)

	err := filepath.WalkDir(dataDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("[history] walk error for %s: %v", p, err)
			return nil
		}

		if d.IsDir() || filepath.Ext(d.Name()) != ".md" {
			return nil
		}

		rel, relErr := filepath.Rel(dataDir, p)
		if relErr != nil {
			log.Printf("[history] rel path error for %s: %v", p, relErr)
			return nil
		}

		content, readErr := os.ReadFile(p)
		if readErr != nil {
			log.Printf("[history] read error for %s: %v", p, readErr)
			return nil
		}

		current[filepath.ToSlash(rel)] = fileRecord{
			Hash:    hashBytes(content),
			Content: string(content),
		}
		return nil
	})

	if os.IsNotExist(err) {
		return current, nil
	}

	return current, err
}

func movementKey(hash string, relPath string) string {
	return hash + "|" + filepath.Base(relPath)
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func seedHistoryPaths(p string) []string {
	paths := []string{}
	normalized := normalizeHistoryPath(p)
	if normalized == "" {
		return paths
	}
	paths = append(paths, normalized)
	if filepath.Ext(normalized) == "" {
		paths = append(paths,
			normalized+".md",
			filepath.ToSlash(filepath.Join(normalized, "index.md")),
		)
	}

	unique := make([]string, 0, len(paths))
	seen := map[string]bool{}
	for _, candidate := range paths {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		unique = append(unique, candidate)
	}

	return unique
}

// HashString returns a deterministic hash for the provided content string.
func HashString(content string) string {
	return hashBytes([]byte(content))
}

func parseSQLiteTimestamp(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, value); err == nil {
			return ts
		}
	}
	return time.Time{}
}

func normalizeHistoryPath(p string) string {
	normalized := strings.TrimSpace(p)
	normalized = strings.TrimLeft(normalized, "/")
	normalized = filepath.ToSlash(normalized)
	return normalized
}
