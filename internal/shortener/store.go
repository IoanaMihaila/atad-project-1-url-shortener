package shortener

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"sort"
	"sync"
	"time"
)

const (
	GeneratedCodeLength = 7
	MinCodeLength       = 4
	MaxCodeLength       = 10
	codeAlphabet        = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_"
)

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
)

// Link represents a shortened URL and its visit statistics.
type Link struct {
	ID          int        `json:"id"`
	Code        string     `json:"code"`
	URL         string     `json:"url"`
	CreatedAt   time.Time  `json:"created_at"`
	Visits      int        `json:"visits"`
	LastVisitAt *time.Time `json:"last_visit_at"`
}

// LinkStats is returned by GET /links/{code}/stats.
type LinkStats struct {
	Code        string     `json:"code"`
	Visits      int        `json:"visits"`
	LastVisitAt *time.Time `json:"last_visit_at"`
}

// Store holds links safely for concurrent HTTP requests.
type Store struct {
	mu     sync.RWMutex
	links  map[string]*Link
	nextID int
}

// NewStore creates an initialized empty Store.
func NewStore() *Store {
	return &Store{
		links:  make(map[string]*Link),
		nextID: 1,
	}
}

// Create validates the URL and optional custom code, stores the link, and returns a copy.
func (s *Store) Create(rawURL, customCode string) (Link, error) {
	if err := validateURL(rawURL); err != nil {
		return Link{}, err
	}

	if customCode != "" {
		if err := validateCode(customCode); err != nil {
			return Link{}, err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	code := customCode
	if code != "" {
		if _, exists := s.links[code]; exists {
			return Link{}, fmt.Errorf("%w: code %q is already taken", ErrConflict, code)
		}
	} else {
		var err error
		code, err = s.generateUniqueCodeLocked()
		if err != nil {
			return Link{}, err
		}
	}

	link := &Link{
		ID:        s.nextID,
		Code:      code,
		URL:       rawURL,
		CreatedAt: time.Now().UTC(),
		Visits:    0,
	}
	s.nextID++
	s.links[code] = link

	return *link, nil
}

// Visit records a redirect hit and returns the target URL.
func (s *Store) Visit(code string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	link, exists := s.links[code]
	if !exists {
		return "", fmt.Errorf("%w: unknown code %q", ErrNotFound, code)
	}

	now := time.Now().UTC()
	link.Visits++
	link.LastVisitAt = &now

	return link.URL, nil
}

// Stats returns visit statistics without incrementing the visit counter.
func (s *Store) Stats(code string) (LinkStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	link, exists := s.links[code]
	if !exists {
		return LinkStats{}, fmt.Errorf("%w: unknown code %q", ErrNotFound, code)
	}

	return LinkStats{
		Code:        link.Code,
		Visits:      link.Visits,
		LastVisitAt: link.LastVisitAt,
	}, nil
}

// List returns all links ordered newest first (descending ID).
func (s *Store) List() []Link {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Link, 0, len(s.links))
	for _, link := range s.links {
		result = append(result, *link)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID > result[j].ID
	})

	return result
}

// Delete removes a link by code.
func (s *Store) Delete(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.links[code]; !exists {
		return fmt.Errorf("%w: unknown code %q", ErrNotFound, code)
	}

	delete(s.links, code)
	return nil
}

// Reset clears all stored links and resets ID numbering back to 1.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.links = make(map[string]*Link)
	s.nextID = 1
}

func (s *Store) generateUniqueCodeLocked() (string, error) {
	for attempt := 0; attempt < 20; attempt++ {
		candidate, err := randomCode(GeneratedCodeLength)
		if err != nil {
			return "", err
		}
		if _, exists := s.links[candidate]; !exists {
			return candidate, nil
		}
	}
	return "", errors.New("failed to generate unique code after retries")
}

func randomCode(length int) (string, error) {
	buf := make([]byte, length)
	max := big.NewInt(int64(len(codeAlphabet)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		buf[i] = codeAlphabet[n.Int64()]
	}
	return string(buf), nil
}

func validateURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("%w: url is required", ErrValidation)
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("%w: invalid url", ErrValidation)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%w: url must use http or https scheme", ErrValidation)
	}
	return nil
}

func validateCode(code string) error {
	if len(code) < MinCodeLength || len(code) > MaxCodeLength {
		return fmt.Errorf("%w: code must be between %d and %d characters", ErrValidation, MinCodeLength, MaxCodeLength)
	}
	for i := 0; i < len(code); i++ {
		c := code[i]
		isValid := (c >= 'A' && c <= 'Z') ||
			(c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') ||
			c == '_'
		if !isValid {
			return fmt.Errorf("%w: code contains invalid character %q", ErrValidation, c)
		}
	}
	return nil
}