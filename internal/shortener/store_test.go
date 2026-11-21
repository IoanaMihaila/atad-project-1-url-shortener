package shortener

import (
	"errors"
	"sync"
	"testing"
)

func TestCreateAndValidate(t *testing.T) {
	s := NewStore()

	// Invalid URLs
	if _, err := s.Create("ftp://example.com", ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation for ftp scheme, got %v", err)
	}
	if _, err := s.Create("not-a-url", ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation for invalid url, got %v", err)
	}

	// Invalid custom codes
	if _, err := s.Create("https://example.com", "abc"); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation for short code, got %v", err)
	}
	if _, err := s.Create("https://example.com", "bad-code!"); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation for invalid chars, got %v", err)
	}

	// Custom code creation
	first, err := s.Create("https://go.dev/doc/effective_go", "gofmt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first.ID != 1 || first.Code != "gofmt" {
		t.Fatalf("unexpected link created: %+v", first)
	}

	// Duplicate custom code -> ErrConflict
	if _, err := s.Create("https://example.org", "gofmt"); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict for duplicate code, got %v", err)
	}

	// Generated code of length L=7
	second, err := s.Create("https://example.org", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if second.ID != 2 || len(second.Code) != GeneratedCodeLength {
		t.Fatalf("expected ID 2 and length %d, got %+v", GeneratedCodeLength, second)
	}
}

func TestVisitStatsAndReset(t *testing.T) {
	s := NewStore()
	_, err := s.Create("https://example.com", "test1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	statsBefore, err := s.Stats("test1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statsBefore.Visits != 0 || statsBefore.LastVisitAt != nil {
		t.Fatalf("expected 0 visits and nil timestamp, got %+v", statsBefore)
	}

	target, err := s.Visit("test1")
	if err != nil || target != "https://example.com" {
		t.Fatalf("expected redirect to https://example.com, got %q, err=%v", target, err)
	}

	// Reading stats multiple times must not increment visits
	for i := 0; i < 2; i++ {
		stats, err := s.Stats("test1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stats.Visits != 1 || stats.LastVisitAt == nil {
			t.Fatalf("expected 1 visit and non-nil timestamp, got %+v", stats)
		}
	}

	// Reset clears state and restarts IDs at 1
	s.Reset()
	if _, err := s.Stats("test1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after reset, got %v", err)
	}

	afterReset, err := s.Create("https://example.net", "fresh")
	if err != nil || afterReset.ID != 1 {
		t.Fatalf("expected ID 1 after reset, got %+v, err=%v", afterReset, err)
	}
}

func TestConcurrentSameCustomCode(t *testing.T) {
	s := NewStore()
	const goroutines = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	var createdCount, conflictCount int
	var mu sync.Mutex

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := s.Create("https://example.com", "race_code")
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				createdCount++
			} else if errors.Is(err, ErrConflict) {
				conflictCount++
			}
		}()
	}

	wg.Wait()

	if createdCount != 1 || conflictCount != goroutines-1 {
		t.Fatalf("expected 1 creation and %d conflicts, got created=%d conflict=%d",
			goroutines-1, createdCount, conflictCount)
	}
}
