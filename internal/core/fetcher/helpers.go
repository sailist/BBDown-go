package fetcher

import "github.com/sailist/BBDown-go/internal/core/entity"

// containsPage reports whether pages contains p using Page.Equal.
func containsPage(pages []entity.Page, p entity.Page) bool {
	for _, existing := range pages {
		if existing.Equal(p) {
			return true
		}
	}
	return false
}
