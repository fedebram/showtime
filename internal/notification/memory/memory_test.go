package memory_test

import (
	"testing"

	"github.com/fedebram/showtime/internal/notification"
	"github.com/fedebram/showtime/internal/notification/memory"
	"github.com/fedebram/showtime/internal/notification/repotest"
)

// TestContract runs the repository contract against the memory repository.
func TestContract(t *testing.T) {
	repotest.Run(t, func(t *testing.T) notification.Repository {
		return memory.NewRepository()
	})
}
