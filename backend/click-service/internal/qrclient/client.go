package qrclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var ErrNotFound = errors.New("not found")

type QrCode struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Active bool   `json:"active"`
}

type Settings struct {
	DefaultRedirectURL string `json:"defaultRedirectUrl"`
}

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(baseURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &Client{
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) GetQrCode(ctx context.Context, id string) (QrCode, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return QrCode{}, ErrNotFound
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/qr-codes/%s", c.BaseURL, id), nil)
	if err != nil {
		return QrCode{}, err
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return QrCode{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return QrCode{}, ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return QrCode{}, fmt.Errorf("qr-service unexpected status: %d", resp.StatusCode)
	}

	var out QrCode
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return QrCode{}, err
	}
	return out, nil
}

func (c *Client) GetSettings(ctx context.Context) (Settings, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/settings", c.BaseURL), nil)
	if err != nil {
		return Settings{}, err
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Settings{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Settings{}, fmt.Errorf("qr-service unexpected status: %d", resp.StatusCode)
	}

	var out Settings
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Settings{}, err
	}
	return out, nil
}
