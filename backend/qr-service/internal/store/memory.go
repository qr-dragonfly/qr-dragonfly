package store

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"qr-service/internal/model"
)

type MemoryStore struct {
	mu       sync.RWMutex
	byID     map[string]model.QrCode
	settings model.UserSettings
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byID: make(map[string]model.QrCode)}
}

func (s *MemoryStore) List() []model.QrCode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.QrCode, 0, len(s.byID))
	for _, v := range s.byID {
		items = append(items, v)
	}

	// newest first
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	return items
}

func (s *MemoryStore) Get(id string) (model.QrCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.byID[id]
	if !ok {
		return model.QrCode{}, ErrNotFound
	}
	return v, nil
}

func (s *MemoryStore) Create(input CreateInput) (model.QrCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.NewString()
	q := model.QrCode{
		ID:        id,
		Label:     input.Label,
		URL:       input.URL,
		Active:    true,
		CreatedAt: time.Now().UTC(),
	}
	if input.Active != nil {
		q.Active = *input.Active
	}
	if q.Label == "" {
		q.Label = "Untitled"
	}

	s.byID[id] = q
	return q, nil
}

func (s *MemoryStore) Update(id string, input UpdateInput) (model.QrCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	q, ok := s.byID[id]
	if !ok {
		return model.QrCode{}, ErrNotFound
	}

	if input.Label != nil {
		q.Label = *input.Label
	}
	if input.URL != nil {
		q.URL = *input.URL
	}
	if input.Active != nil {
		q.Active = *input.Active
	}
	if q.Label == "" {
		q.Label = "Untitled"
	}

	s.byID[id] = q
	return q, nil
}

func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byID[id]; !ok {
		return ErrNotFound
	}
	delete(s.byID, id)
	return nil
}

func (s *MemoryStore) CountTotal() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byID), nil
}

func (s *MemoryStore) CountActive() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	active := 0
	for _, v := range s.byID {
		if v.Active {
			active++
		}
	}
	return active, nil
}

func (s *MemoryStore) GetSettings() (model.UserSettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings, nil
}

func (s *MemoryStore) UpdateSettings(settings model.UserSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = settings
	return nil
}
