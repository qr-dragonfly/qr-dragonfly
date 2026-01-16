package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"qr-service/internal/store"
)

type qrResp struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	URL   string `json:"url"`
}

func TestURLValidation_Create_RequiresHTTPS(t *testing.T) {
	s := store.NewMemoryStore()
	r := NewRouter(Server{Store: s})

	body, _ := json.Marshal(map[string]any{"label": "x", "url": "http://example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/qr-codes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, w.Code)
	}
	var resp errResp
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != "url_invalid" {
		t.Fatalf("expected url_invalid, got %q", resp.Error)
	}
}

func TestURLValidation_Update_RequiresHTTPS(t *testing.T) {
	s := store.NewMemoryStore()
	r := NewRouter(Server{Store: s})

	createBody, _ := json.Marshal(map[string]any{"label": "x", "url": "https://example.com"})
	createReq := httptest.NewRequest(http.MethodPost, "/api/qr-codes", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, createW.Code)
	}
	var created qrResp
	_ = json.NewDecoder(createW.Body).Decode(&created)
	if created.ID == "" {
		t.Fatalf("expected created id")
	}

	patchBody, _ := json.Marshal(map[string]any{"url": "http://example.com"})
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/qr-codes/"+created.ID, bytes.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchW := httptest.NewRecorder()
	r.ServeHTTP(patchW, patchReq)
	if patchW.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, patchW.Code)
	}
	var resp errResp
	_ = json.NewDecoder(patchW.Body).Decode(&resp)
	if resp.Error != "url_invalid" {
		t.Fatalf("expected url_invalid, got %q", resp.Error)
	}
}
