package search

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io/fs"
	"log"
	"os"
	"path/filepath"
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
	Status       FileHistoryStatus
	PreviousPath *string
}

// CaptureFileHistory snapshots all Markdown files under dataDir.
// It records new rows when files are created, modified, removed or moved.
func (s *SQLiteIndex) CaptureFileHistory(dataDir string) error {
	if s.db == nil {
		return sql.ErrConnDone
	}

	currentFiles, err := scanMarkdownHashes(dataDir)
	if err != nil {
		return err
	}

	latest, err := s.latestFileSnapshots()
	if err != nil {
		return err
	}

	missing := map[string]FileHistorySnapshot{}
	moveCandidates := map[string][]FileHistorySnapshot{}
	for path, snap := range latest {
		if _, ok := currentFiles[path]; !ok {
			missing[path] = snap
			if snap.Status != FileStatusDeleted {
				key := movementKey(snap.Hash, path)
				moveCandidates[key] = append(moveCandidates[key], snap)
			}
		}
	}

	for relPath, hash := range currentFiles {
		if snap, ok := latest[relPath]; ok {
			if snap.Status == FileStatusDeleted {
				if err := s.insertHistoryEntry(relPath, hash, FileStatusCreated, nil); err != nil {
					return err
				}
				log.Printf("[history] recorded created for %s", relPath)
				continue
			}

			if snap.Hash != hash {
				if err := s.insertHistoryEntry(relPath, hash, FileStatusModified, nil); err != nil {
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
			if err := s.insertHistoryEntry(relPath, hash, FileStatusMoved, &prev.Path); err != nil {
				return err
			}
			log.Printf("[history] recorded moved from %s to %s", prev.Path, relPath)
			continue
		}

		if err := s.insertHistoryEntry(relPath, hash, FileStatusCreated, nil); err != nil {
			return err
		}
		log.Printf("[history] recorded created for %s", relPath)
	}

	for _, snap := range missing {
		if snap.Status == FileStatusDeleted {
			continue
		}
		if err := s.insertHistoryEntry(snap.Path, snap.Hash, FileStatusDeleted, nil); err != nil {
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
		SELECT fh.path, fh.hash, fh.status, fh.previous_path
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
		if err := rows.Scan(&snap.Path, &snap.Hash, &snap.Status, &prev); err != nil {
			return nil, err
		}
		if prev.Valid {
			snap.PreviousPath = &prev.String
		}
		snapshots[snap.Path] = snap
	}

	return snapshots, rows.Err()
}

func (s *SQLiteIndex) insertHistoryEntry(path string, hash string, status FileHistoryStatus, previousPath *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var prev interface{}
	if previousPath != nil {
		prev = *previousPath
	}

	_, err := s.db.Exec(`
		INSERT INTO file_history (path, hash, status, previous_path)
		VALUES (?, ?, ?, ?);
	`, path, hash, status, prev)

	return err
}

func scanMarkdownHashes(dataDir string) (map[string]string, error) {
	current := make(map[string]string)

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

		hash, hashErr := hashFile(p)
		if hashErr != nil {
			log.Printf("[history] hash error for %s: %v", p, hashErr)
			return nil
		}

		current[filepath.ToSlash(rel)] = hash
		return nil
	})

	if os.IsNotExist(err) {
		return current, nil
	}

	return current, err
}

func hashFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func movementKey(hash string, relPath string) string {
	return hash + "|" + filepath.Base(relPath)
}
