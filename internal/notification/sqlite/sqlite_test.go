package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/fedebram/showtime/internal/notification"
	"github.com/fedebram/showtime/internal/notification/repotest"
	"github.com/fedebram/showtime/internal/notification/sqlite"
)

// TestContract runs the repository contract against SQLite,
// each test on its own fresh database file.
func TestContract(t *testing.T) {
	repotest.Run(t, func(t *testing.T) notification.Repository {
		repo, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { repo.Close() })
		return repo
	})
}
