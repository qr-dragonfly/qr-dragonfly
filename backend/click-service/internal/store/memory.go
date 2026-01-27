package store

import (
	"sync"
	"time"
)

type MemoryStore struct {
	mu    sync.RWMutex
	stats map[string]ClickStats
	daily map[string]map[string]*DailyClickStats
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{stats: map[string]ClickStats{}, daily: map[string]map[string]*DailyClickStats{}}
}

func (s *MemoryStore) RecordClick(event ClickEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := event.At.UTC()
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	dayIso := day.Format("2006-01-02")
	hour := t.Hour()

	byDay, ok := s.daily[event.QrCodeID]
	if !ok {
		byDay = map[string]*DailyClickStats{}
		s.daily[event.QrCodeID] = byDay
	}

	ds, ok := byDay[dayIso]
	if !ok {
		ds = &DailyClickStats{QrCodeID: event.QrCodeID, DayIso: dayIso}
		byDay[dayIso] = ds
	}

	ds.Total++
	incrementHour(ds, hour)
	if region := event.Country; region != "" {
		if ds.RegionCounts == nil {
			ds.RegionCounts = map[string]int{}
		}
		ds.RegionCounts[region]++
	}

	st := s.stats[event.QrCodeID]
	if st.QrCodeID == "" {
		st.QrCodeID = event.QrCodeID
	}
	st.Total++
	st.LastAtIso = event.At.UTC().Format(time.RFC3339)
	st.LastCountry = event.Country
	s.stats[event.QrCodeID] = st
	return nil
}

func (s *MemoryStore) GetStats(qrCodeID string) (ClickStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st, ok := s.stats[qrCodeID]
	if !ok {
		return ClickStats{}, ErrNotFound
	}
	return st, nil
}

func (s *MemoryStore) GetDaily(qrCodeID string, day time.Time) (DailyClickStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	byDay, ok := s.daily[qrCodeID]
	if !ok {
		return DailyClickStats{}, ErrNotFound
	}

	day = day.UTC()
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	key := day.Format("2006-01-02")

	ds, ok := byDay[key]
	if !ok {
		return DailyClickStats{}, ErrNotFound
	}
	return *ds, nil
}

func (s *MemoryStore) GetDailyBatch(qrCodeID string, days []time.Time) (map[string]DailyClickStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	byDay, ok := s.daily[qrCodeID]
	if !ok {
		return map[string]DailyClickStats{}, nil
	}

	result := make(map[string]DailyClickStats)
	for _, day := range days {
		day = day.UTC()
		day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
		key := day.Format("2006-01-02")

		if ds, ok := byDay[key]; ok {
			result[key] = *ds
		}
	}

	return result, nil
}

func incrementHour(ds *DailyClickStats, hour int) {
	switch hour {
	case 0:
		ds.Hour00++
	case 1:
		ds.Hour01++
	case 2:
		ds.Hour02++
	case 3:
		ds.Hour03++
	case 4:
		ds.Hour04++
	case 5:
		ds.Hour05++
	case 6:
		ds.Hour06++
	case 7:
		ds.Hour07++
	case 8:
		ds.Hour08++
	case 9:
		ds.Hour09++
	case 10:
		ds.Hour10++
	case 11:
		ds.Hour11++
	case 12:
		ds.Hour12++
	case 13:
		ds.Hour13++
	case 14:
		ds.Hour14++
	case 15:
		ds.Hour15++
	case 16:
		ds.Hour16++
	case 17:
		ds.Hour17++
	case 18:
		ds.Hour18++
	case 19:
		ds.Hour19++
	case 20:
		ds.Hour20++
	case 21:
		ds.Hour21++
	case 22:
		ds.Hour22++
	case 23:
		ds.Hour23++
	default:
		// ignore
	}
}
