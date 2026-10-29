package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/fedebram/showtime/internal/booking/repotest"
	"github.com/fedebram/showtime/internal/booking/sqlite"
)

// TestContract runs the repository contract against SQLite,
// each test on its own fresh database file.
func TestContract(t *testing.T) {
	repotest.Run(t, func(t *testing.T) booking.Repository {
		repo, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { repo.Close() })
		return repo
	})
}
