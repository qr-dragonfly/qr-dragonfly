package store

import (
	"errors"

	"qr-service/internal/model"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	List() []model.QrCode
	Get(id string) (model.QrCode, error)
	Create(input CreateInput) (model.QrCode, error)
	Update(id string, input UpdateInput) (model.QrCode, error)
	Delete(id string) error

	CountTotal() (int, error)
	CountActive() (int, error)

	// Settings
	GetSettings() (model.UserSettings, error)
	UpdateSettings(settings model.UserSettings) error
}

type CreateInput struct {
	Label  string
	URL    string
	Active *bool
}

type UpdateInput struct {
	Label  *string
	URL    *string
	Active *bool
}
