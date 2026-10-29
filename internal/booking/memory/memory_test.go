package memory_test

import (
	"testing"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/fedebram/showtime/internal/booking/memory"
	"github.com/fedebram/showtime/internal/booking/repotest"
)

// _test package is needed to avoid import cycle

// TestContract runs the repository contract against the memory repository.
func TestContract(t *testing.T) {
	repotest.Run(t, func(t *testing.T) booking.Repository {
		return memory.NewRepository()
	})
}
