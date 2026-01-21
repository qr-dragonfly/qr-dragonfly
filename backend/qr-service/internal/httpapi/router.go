package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"qr-service/internal/middleware"
	"qr-service/internal/model"
	"qr-service/internal/store"
)

type Server struct {
	Store store.Store
}

type quota struct {
	maxActive int
	maxTotal  int
}

func userTypeFromRequest(r *http.Request) string {
	v := strings.TrimSpace(strings.ToLower(r.Header.Get("X-User-Type")))
	if v == "" {
		return "free"
	}
	switch v {
	case "free", "basic", "enterprise", "admin":
		return v
	default:
		return "free"
	}
}

func quotaForUserType(userType string) quota {
	switch userType {
	case "basic":
		return quota{maxActive: 50, maxTotal: 200}
	case "enterprise":
		return quota{maxActive: 2000, maxTotal: 10000}
	case "admin":
		// Treat admin as effectively unlimited for now.
		return quota{maxActive: 1_000_000_000, maxTotal: 1_000_000_000}
	case "free":
		fallthrough
	default:
		return quota{maxActive: 5, maxTotal: 20}
	}
}

type createQrCodeRequest struct {
	Label  string `json:"label"`
	URL    string `json:"url"`
	Active *bool  `json:"active,omitempty"`
}

type updateQrCodeRequest struct {
	Label  *string `json:"label"`
	URL    *string `json:"url"`
	Active *bool   `json:"active,omitempty"`
}

func NewRouter(srv Server) http.Handler {
	mux := http.NewServeMux()

	healthHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	collectionHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items := srv.Store.List()
			for i := range items {
				items[i] = items[i].NormalizeForResponse()
			}
			writeJSON(w, http.StatusOK, items)
			return

		case http.MethodPost:
			qt := quotaForUserType(userTypeFromRequest(r))
			var req createQrCodeRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
				return
			}
			req.URL = strings.TrimSpace(req.URL)
			req.Label = strings.TrimSpace(req.Label)
			if req.URL == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url_required"})
				return
			}
			if !isValidHTTPURL(req.URL) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url_invalid"})
				return
			}

			requestedActive := true
			if req.Active != nil {
				requestedActive = *req.Active
			}

			total, err := srv.Store.CountTotal()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "quota_check_failed"})
				return
			}
			if total >= qt.maxTotal {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "quota_total_exceeded"})
				return
			}
			if requestedActive {
				active, err := srv.Store.CountActive()
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "quota_check_failed"})
					return
				}
				if active >= qt.maxActive {
					writeJSON(w, http.StatusForbidden, map[string]string{"error": "quota_active_exceeded"})
					return
				}
			}
			created, err := srv.Store.Create(store.CreateInput{Label: req.Label, URL: req.URL, Active: req.Active})
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create_failed"})
				return
			}
			writeJSON(w, http.StatusCreated, created.NormalizeForResponse())
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})

	itemHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/qr-codes/")
		id = strings.Trim(id, "/")
		if id == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			item, err := srv.Store.Get(id)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
					return
				}
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_failed"})
				return
			}
			writeJSON(w, http.StatusOK, item.NormalizeForResponse())
			return
		case http.MethodPatch:
			qt := quotaForUserType(userTypeFromRequest(r))
			var req updateQrCodeRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
				return
			}
			if req.URL != nil {
				v := strings.TrimSpace(*req.URL)
				req.URL = &v
				if v == "" {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url_required"})
					return
				}
				if !isValidHTTPURL(v) {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url_invalid"})
					return
				}
			}
			if req.Label != nil {
				v := strings.TrimSpace(*req.Label)
				req.Label = &v
			}

			if req.Active != nil && *req.Active {
				current, err := srv.Store.Get(id)
				if err != nil {
					if errors.Is(err, store.ErrNotFound) {
						writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
						return
					}
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_failed"})
					return
				}

				// Only enforce if we're transitioning false -> true.
				if !current.Active {
					active, err := srv.Store.CountActive()
					if err != nil {
						writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "quota_check_failed"})
						return
					}
					if active >= qt.maxActive {
						writeJSON(w, http.StatusForbidden, map[string]string{"error": "quota_active_exceeded"})
						return
					}
				}
			}
			updated, err := srv.Store.Update(id, store.UpdateInput{Label: req.Label, URL: req.URL, Active: req.Active})
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
					return
				}
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_failed"})
				return
			}
			writeJSON(w, http.StatusOK, updated.NormalizeForResponse())
			return
		case http.MethodDelete:
			err := srv.Store.Delete(id)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
					return
				}
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete_failed"})
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})

	settingsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			settings, err := srv.Store.GetSettings()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_get_settings"})
				return
			}
			writeJSON(w, http.StatusOK, settings)
			return
		case http.MethodPut:
			var req model.UserSettings
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
				return
			}
			if err := srv.Store.UpdateSettings(req); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_update_settings"})
				return
			}
			writeJSON(w, http.StatusOK, req)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})

	wrap := func(h http.Handler) http.Handler {
		return middleware.Recoverer(middleware.RequestID(middleware.ExposeResponseHeaders(middleware.EnforceJSONHandler(h))))
	}

	mux.Handle("/healthz", wrap(healthHandler))
	mux.Handle("/api/qr-codes", wrap(collectionHandler))
	mux.Handle("/api/qr-codes/", wrap(itemHandler))
	mux.Handle("/api/settings", wrap(settingsHandler))

	return mux
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func isValidHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "https" {
		return false
	}
	if u.Host == "" {
		return false
	}
	return true
}
