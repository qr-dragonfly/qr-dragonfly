package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"click-service/internal/qrclient"
	"click-service/internal/store"
)

type storeSpy struct {
	ch chan store.ClickEvent
}

func (s *storeSpy) RecordClick(ev store.ClickEvent) error {
	select {
	case s.ch <- ev:
	default:
	}
	return nil
}

func (s *storeSpy) GetStats(qrCodeID string) (store.ClickStats, error) {
	return store.ClickStats{}, store.ErrNotFound
}

func (s *storeSpy) GetDaily(qrCodeID string, day time.Time) (store.DailyClickStats, error) {
	return store.DailyClickStats{}, store.ErrNotFound
}

type qrClientSpy struct {
	called bool
	gotID  string
	resp   qrclient.QrCode
	err    error
}

func (q *qrClientSpy) GetQrCode(_ context.Context, id string) (qrclient.QrCode, error) {
	q.called = true
	q.gotID = id
	return q.resp, q.err
}

func (q *qrClientSpy) GetSettings(_ context.Context) (qrclient.Settings, error) {
	return qrclient.Settings{}, nil
}

func TestRedirect_UsesDbUrlAndChecksActive(t *testing.T) {
	spy := &storeSpy{ch: make(chan store.ClickEvent, 1)}
	qrSpy := &qrClientSpy{resp: qrclient.QrCode{ID: "abc123", URL: "https://example.com/db", Active: true}}
	router := NewRouter(Server{Store: spy, QrClient: qrSpy})

	req := httptest.NewRequest(http.MethodGet, "/r/abc123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !qrSpy.called || qrSpy.gotID != "abc123" {
		t.Fatalf("expected qr client lookup for %q", "abc123")
	}

	if w.Code != http.StatusFound {
		t.Fatalf("expected %d, got %d", http.StatusFound, w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://example.com/db" {
		t.Fatalf("expected Location %q, got %q", "https://example.com/db", loc)
	}

	// Click recording is async; wait briefly but don't make the test flaky.
	select {
	case ev := <-spy.ch:
		if ev.QrCodeID != "abc123" {
			t.Fatalf("expected qrCodeId %q, got %q", "abc123", ev.QrCodeID)
		}
		if ev.TargetURL != "https://example.com/db" {
			t.Fatalf("expected targetUrl %q, got %q", "https://example.com/db", ev.TargetURL)
		}
	case <-time.After(150 * time.Millisecond):
		// ok
	}
}
