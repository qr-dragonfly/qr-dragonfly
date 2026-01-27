package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"click-service/internal/middleware"
	"click-service/internal/qrclient"
	"click-service/internal/store"
)

type Server struct {
	Store    store.Store
	QrClient interface {
		GetQrCode(ctx context.Context, id string) (qrclient.QrCode, error)
		GetSettings(ctx context.Context) (qrclient.Settings, error)
	}
}

func NewRouter(srv Server) http.Handler {
	mux := http.NewServeMux()

	wrapAPI := func(h http.Handler) http.Handler {
		return middleware.Recoverer(middleware.RequestID(middleware.ExposeResponseHeaders(middleware.EnforceJSONHandler(h))))
	}
	wrapAny := func(h http.Handler) http.Handler {
		return middleware.Recoverer(middleware.RequestID(h))
	}

	healthHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		id := strings.TrimPrefix(r.URL.Path, "/r/")
		id = strings.Trim(id, "/")
		if id == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		ctx := r.Context()
		if srv.QrClient == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		qr, err := srv.QrClient.GetQrCode(ctx, id)
		if err != nil {
			if errors.Is(err, qrclient.ErrNotFound) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusBadGateway)
			return
		}

		// If inactive, check for global default redirect URL
		if !qr.Active {
			settings, err := srv.QrClient.GetSettings(ctx)
			if err == nil && strings.TrimSpace(settings.DefaultRedirectURL) != "" {
				// Redirect to global default URL without recording click
				w.Header().Set("Cache-Control", "no-store")
				http.Redirect(w, r, strings.TrimSpace(settings.DefaultRedirectURL), http.StatusFound)
				return
			}
			// No default URL, return 404
			w.WriteHeader(http.StatusNotFound)
			return
		}

		targetURL := strings.TrimSpace(qr.URL)
		if targetURL == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Build the click event now, but record it asynchronously so the redirect is as fast as possible.
		event := store.ClickEvent{
			At:         time.Now().UTC(),
			QrCodeID:   id,
			TargetURL:  targetURL,
			IP:         clientIP(r),
			UserAgent:  strings.TrimSpace(r.UserAgent()),
			Referer:    strings.TrimSpace(r.Referer()),
			Country:    countryFromHeaders(r),
			RequestID:  strings.TrimSpace(w.Header().Get("X-Request-Id")),
			AcceptLang: strings.TrimSpace(r.Header.Get("Accept-Language")),
		}

		w.Header().Set("Cache-Control", "no-store")
		http.Redirect(w, r, targetURL, http.StatusFound)

		go func(ev store.ClickEvent) {
			defer func() { _ = recover() }()
			_ = srv.Store.RecordClick(ev)
		}(event)
	})

	clicksHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		rest := strings.TrimPrefix(r.URL.Path, "/api/clicks/")
		rest = strings.Trim(rest, "/")

		// Check for query-based endpoints first
		if rest == "stats" {
			// /api/clicks/stats?qrId=xxx
			qrID := strings.TrimSpace(r.URL.Query().Get("qrId"))
			if qrID == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "qrId_required"})
				return
			}
			st, err := srv.Store.GetStats(qrID)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
					return
				}
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stats_failed"})
				return
			}
			writeJSON(w, http.StatusOK, st)
			return
		}

		if rest == "daily" {
			// /api/clicks/daily?qrId=xxx&day=2026-01-02
			qrID := strings.TrimSpace(r.URL.Query().Get("qrId"))
			if qrID == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "qrId_required"})
				return
			}

			day := time.Now().UTC()
			if raw := strings.TrimSpace(r.URL.Query().Get("day")); raw == "" {
				raw = strings.TrimSpace(r.URL.Query().Get("date"))
				if raw == "" {
					// default: today (UTC)
					day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
				} else {
					parsed, err := time.Parse("2006-01-02", raw)
					if err != nil {
						writeJSON(w, http.StatusBadRequest, map[string]string{"error": "day_invalid"})
						return
					}
					day = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
				}
			} else {
				parsed, err := time.Parse("2006-01-02", raw)
				if err != nil {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "day_invalid"})
					return
				}
				day = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
			}

			ds, err := srv.Store.GetDaily(qrID, day)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
					return
				}
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "daily_failed"})
				return
			}
			writeJSON(w, http.StatusOK, ds)
			return
		}

		if rest == "daily-batch" {
			// /api/clicks/daily-batch?qrId=xxx&days=2026-01-19,2026-01-20,2026-01-21
			qrID := strings.TrimSpace(r.URL.Query().Get("qrId"))
			if qrID == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "qrId_required"})
				return
			}

			daysParam := strings.TrimSpace(r.URL.Query().Get("days"))
			if daysParam == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "days_required"})
				return
			}

			dayStrings := strings.Split(daysParam, ",")
			days := make([]time.Time, 0, len(dayStrings))
			for _, ds := range dayStrings {
				ds = strings.TrimSpace(ds)
				if ds == "" {
					continue
				}
				parsed, err := time.Parse("2006-01-02", ds)
				if err != nil {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_day_format"})
					return
				}
				days = append(days, parsed)
			}

			if len(days) == 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no_valid_days"})
				return
			}

			result, err := srv.Store.GetDailyBatch(qrID, days)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "batch_failed"})
				return
			}
			writeJSON(w, http.StatusOK, result)
			return
		}

		// Legacy path-based endpoints for backward compatibility
		if rest == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		parts := strings.Split(rest, "/")
		if len(parts) == 1 {
			// /api/clicks/{qrId}
			qrID := parts[0]
			st, err := srv.Store.GetStats(qrID)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
					return
				}
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stats_failed"})
				return
			}
			writeJSON(w, http.StatusOK, st)
			return
		}

		if len(parts) == 2 && parts[1] == "series" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if len(parts) == 2 && parts[1] == "daily" {
			// /api/clicks/{qrId}/daily?day=2026-01-02
			qrID := parts[0]
			day := time.Now().UTC()
			if raw := strings.TrimSpace(r.URL.Query().Get("day")); raw == "" {
				raw = strings.TrimSpace(r.URL.Query().Get("date"))
				if raw == "" {
					// default: today (UTC)
					day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
				} else {
					parsed, err := time.Parse("2006-01-02", raw)
					if err != nil {
						writeJSON(w, http.StatusBadRequest, map[string]string{"error": "day_invalid"})
						return
					}
					day = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
				}
			} else {
				parsed, err := time.Parse("2006-01-02", raw)
				if err != nil {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "day_invalid"})
					return
				}
				day = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
			}

			ds, err := srv.Store.GetDaily(qrID, day)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
					return
				}
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "daily_failed"})
				return
			}
			writeJSON(w, http.StatusOK, ds)
			return
		}

		if len(parts) == 2 && parts[1] == "daily-batch" {
			// /api/clicks/{qrId}/daily-batch?days=2026-01-19,2026-01-20,2026-01-21
			qrID := parts[0]
			daysParam := strings.TrimSpace(r.URL.Query().Get("days"))
			if daysParam == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "days_required"})
				return
			}

			dayStrings := strings.Split(daysParam, ",")
			days := make([]time.Time, 0, len(dayStrings))
			for _, ds := range dayStrings {
				ds = strings.TrimSpace(ds)
				if ds == "" {
					continue
				}
				parsed, err := time.Parse("2006-01-02", ds)
				if err != nil {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_day_format"})
					return
				}
				days = append(days, parsed)
			}

			if len(days) == 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no_valid_days"})
				return
			}

			result, err := srv.Store.GetDailyBatch(qrID, days)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "batch_failed"})
				return
			}
			writeJSON(w, http.StatusOK, result)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	mux.Handle("/healthz", wrapAPI(healthHandler))
	mux.Handle("/r/", wrapAny(redirectHandler))
	mux.Handle("/api/clicks/", wrapAPI(clicksHandler))

	return mux
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func clientIP(r *http.Request) string {
	// Prefer proxy headers if present.
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff != "" {
		// First IP in the list is the client.
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-Ip")); xrip != "" {
		return xrip
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func countryFromHeaders(r *http.Request) string {
	// Cloudflare
	if v := strings.TrimSpace(r.Header.Get("CF-IPCountry")); v != "" {
		return v
	}
	// Generic options
	if v := strings.TrimSpace(r.Header.Get("X-Geo-Country")); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.Header.Get("X-Country")); v != "" {
		return v
	}
	return ""
}
