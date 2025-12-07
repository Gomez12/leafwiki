package search

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Gomez12/wiki/internal/core/tree"
	"github.com/fsnotify/fsnotify"
)

const historyScanInterval = 5 * time.Minute

type Watcher struct {
	DataDir     string
	TreeService *tree.TreeService
	Index       *SQLiteIndex
	Status      *IndexingStatus
	watcher     *fsnotify.Watcher
	historyTick *time.Ticker
	stopCh      chan struct{}
	historyReq  chan struct{}
}

func NewWatcher(dataDir string, treeService *tree.TreeService, index *SQLiteIndex, status *IndexingStatus) (*Watcher, error) {
	watcher := &Watcher{
		DataDir:     dataDir,
		TreeService: treeService,
		Index:       index,
		Status:      status,
		watcher:     nil,
	}

	return watcher, nil
}

func (w *Watcher) Start() error {
	var err error
	if w.watcher, err = fsnotify.NewWatcher(); err != nil {
		return err
	}

	w.stopCh = make(chan struct{})
	w.historyReq = make(chan struct{}, 1)
	w.historyTick = time.NewTicker(historyScanInterval)

	err = filepath.Walk(w.DataDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("[watcher] walk error: %v", err)
			return nil
		}
		if info.IsDir() {
			if err := w.watcher.Add(p); err != nil {
				log.Printf("[watcher] add error: %v", err)
			}
		}
		return nil
	})
	if err != nil {
		w.historyTick.Stop()
		close(w.stopCh)
		_ = w.watcher.Close()
		return err
	}

	go w.runHistoryRecorder()

	go func() {
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}

				// Normalize path
				eventPath := filepath.ToSlash(event.Name)

				info, statErr := os.Stat(eventPath)
				isDir := statErr == nil && info.IsDir()

				// New Directory or Moved
				if (event.Op&(fsnotify.Create|fsnotify.Rename) != 0) && isDir {
					// Watch recursive
					log.Printf("[watcher] watching new dir: %s", eventPath)
					if err := filepath.Walk(eventPath, func(p string, i os.FileInfo, walkErr error) error {
						if walkErr != nil {
							// Log and keep walking other files/dirs
							log.Printf("[watcher] walk error for %s: %v", p, walkErr)
							return nil
						}
						if i == nil {
							// Nothing to do for this node
							return nil
						}

						if i.IsDir() {
							if err := w.watcher.Add(p); err != nil {
								log.Printf("[watcher] add error: %v", err)
								return nil // continue walking
							}
						} else if filepath.Ext(p) == ".md" {
							reindexFile(p, w.DataDir, w.TreeService, w.Index, w.Status)
						}
						return nil
					}); err != nil {
						log.Printf("[watcher] walk error: %v", err)
					}
					continue
				}

				if filepath.Ext(eventPath) != ".md" {
					continue
				}

				switch {
				case event.Op&(fsnotify.Create|fsnotify.Write) != 0:
					reindexFile(eventPath, w.DataDir, w.TreeService, w.Index, w.Status)
					w.requestHistorySnapshot()

				case event.Op&fsnotify.Remove != 0:
					relPath, err := filepath.Rel(w.DataDir, eventPath)
					if err == nil {
						log.Printf("[watcher] file removed: %s", relPath)
						cnt, err := w.Index.RemovePageByFilePath(relPath)
						if err != nil {
							log.Printf("[watcher] remove error: %v", err)
						} else {
							log.Printf("[watcher] removed %d pages for: %s", cnt, relPath)
						}
					}
					w.requestHistorySnapshot()

				case event.Op&fsnotify.Rename != 0 && !isDir:
					relPath, err := filepath.Rel(w.DataDir, eventPath)
					if err == nil {
						log.Printf("[watcher] file renamed/removed: %s", relPath)
						cnt, err := w.Index.RemovePageByFilePath(relPath)
						if err != nil {
							log.Printf("[watcher] remove error: %v", err)
						} else {
							log.Printf("[watcher] removed %d pages for: %s", cnt, relPath)
						}
					}
					w.requestHistorySnapshot()
				}

			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				log.Printf("[watcher] error: %v", err)
			}
		}
	}()

	log.Println("[watcher] started watching:", w.DataDir)
	return nil
}

func (w *Watcher) runHistoryRecorder() {
	if w.Index == nil || w.historyTick == nil {
		return
	}

	// Run once immediately so we capture state at startup.
	if err := w.Index.CaptureFileHistory(w.DataDir); err != nil {
		log.Printf("[history] initial snapshot error: %v", err)
	}

	for {
		select {
		case <-w.historyTick.C:
			if err := w.Index.CaptureFileHistory(w.DataDir); err != nil {
				log.Printf("[history] snapshot error: %v", err)
			}
		case <-w.historyReq:
			if err := w.Index.CaptureFileHistory(w.DataDir); err != nil {
				log.Printf("[history] snapshot error: %v", err)
			}
		case <-w.stopCh:
			return
		}
	}
}

func (w *Watcher) Stop() error {
	if w.historyTick != nil {
		w.historyTick.Stop()
	}
	if w.stopCh != nil {
		close(w.stopCh)
	}
	if w.historyReq != nil {
		close(w.historyReq)
	}
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}

func (w *Watcher) requestHistorySnapshot() {
	if w.historyReq == nil {
		return
	}
	select {
	case w.historyReq <- struct{}{}:
	default:
	}
}

func reindexFile(fullPath, dataDir string, treeService *tree.TreeService, index *SQLiteIndex, status *IndexingStatus) {
	rel, err := filepath.Rel(dataDir, fullPath)
	if err != nil {
		log.Printf("[watcher] rel path error: %v", err)
		return
	}

	routePath := strings.TrimSuffix(rel, filepath.Ext(rel))
	routePath = filepath.ToSlash(strings.TrimSuffix(routePath, "/index"))

	content, err := os.ReadFile(fullPath)
	if err != nil {
		log.Printf("[watcher] read error: %v", err)
		return
	}

	page, err := treeService.FindPageByRoutePath(treeService.GetTree().Children, routePath)
	if err != nil {
		// File exists on disk but not in tree: auto-attach and continue indexing.
		node, ensureErr := ensureTreeNodeForFile(treeService, routePath, content)
		if ensureErr != nil {
			log.Printf("[watcher] auto-attach failed for %s: %v", rel, ensureErr)
			return
		}
		log.Printf("[watcher] auto-attached missing path: %s", rel)
		page = &tree.Page{PageNode: node, Content: string(content)}
	}

	err = index.IndexPage(page.CalculatePath(), rel, page.ID, page.Title, string(content))
	if err != nil {
		status.Fail()
		log.Printf("[watcher] index error: %v", err)
	} else {
		status.Success()
		log.Printf("[watcher] indexed: %s", rel)
	}
}
