package search

import (
	"bufio"
	"bytes"
	"log"
	"path/filepath"
	"strings"

	"github.com/Gomez12/wiki/internal/core/tree"
)

// BuildAndRunIndexer initializes the indexer with the given tree service and SQLite index,
func BuildAndRunIndexer(treeService *tree.TreeService, sqliteIndex *SQLiteIndex, dataDir string, workers int, status *IndexingStatus) error {
	status.Start()
	indexer := NewIndexer(dataDir, workers, func(file string, content []byte) error {
		rel, err := filepath.Rel(dataDir, file)
		if err != nil {
			status.Fail()
			return err
		}
		routePath := strings.TrimSuffix(rel, filepath.Ext(rel))
		routePath = filepath.ToSlash(routePath)

		// Remove "/index" suffix from the route path unconditionally
		routePath = strings.TrimSuffix(routePath, "/index")

		page, err := treeService.FindPageByRoutePath(treeService.GetTree().Children, routePath)
		if err != nil {
			// the page is on the filesystem but not in the tree, attach it automatically
			node, ensureErr := ensureTreeNodeForFile(treeService, routePath, content)
			if ensureErr != nil {
				log.Printf("[indexer] auto-attach failed for %s: %v", rel, ensureErr)
				status.Fail()
				return nil
			}
			page = &tree.Page{PageNode: node, Content: string(content)}
		}

		// Get path by PageID
		pagePath := page.CalculatePath()

		if err := sqliteIndex.IndexPage(pagePath, rel, page.ID, page.Title, string(content)); err != nil {
			log.Printf("[indexer] error indexing page %s: %v", rel, err)
			status.Fail()
			return err
		}

		status.Success()
		return nil
	})

	err := indexer.Start()
	status.Finish()
	return err
}

// ensureTreeNodeForFile attaches a missing file path to the tree and derives a title.
func ensureTreeNodeForFile(treeService *tree.TreeService, routePath string, content []byte) (*tree.PageNode, error) {
	slug := routePath
	if idx := strings.LastIndex(routePath, "/"); idx >= 0 && idx+1 < len(routePath) {
		slug = routePath[idx+1:]
	}
	title := titleFromContent(content, slug)
	return treeService.AttachExistingPath(routePath, title)
}

// titleFromContent extracts the first Markdown heading as title, falling back to the slug.
func titleFromContent(content []byte, fallbackSlug string) string {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			title := strings.TrimSpace(strings.TrimLeft(line, "#"))
			if title != "" {
				return title
			}
		}
	}
	return slugToTitle(fallbackSlug)
}

// slugToTitle converts a slug like "my-page" to "My Page".
func slugToTitle(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "Untitled"
	}

	parts := strings.FieldsFunc(slug, func(r rune) bool {
		return r == '-' || r == '_' || r == '/'
	})

	for i, p := range parts {
		if p == "" {
			continue
		}
		if len(p) == 1 {
			parts[i] = strings.ToUpper(p)
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}

	return strings.Join(parts, " ")
}
