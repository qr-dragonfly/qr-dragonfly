package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"qr-service/internal/model"
)

type PostgresStore struct {
	db *gorm.DB
}

type qrCodeRow struct {
	ID        uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Label     string    `gorm:"not null"`
	URL       string    `gorm:"not null"`
	Active    bool      `gorm:"not null;default:true;index:qr_codes_active_idx"`
	CreatedAt time.Time `gorm:"not null;index:qr_codes_created_at_idx,sort:desc"`
}

func (qrCodeRow) TableName() string { return "qr_codes" }

type settingsRow struct {
	ID                 int    `gorm:"primaryKey;autoIncrement"`
	DefaultRedirectURL string `gorm:"default:''"`
}

func (settingsRow) TableName() string { return "user_settings" }

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	gdb, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	// Heroku Postgres can be bursty; keep defaults conservative.
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	s := &PostgresStore{db: gdb}
	if err := s.ensureSchema(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return s, nil
}

func (s *PostgresStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *PostgresStore) ensureSchema(ctx context.Context) error {
	db := s.db.WithContext(ctx)

	// Needed for gen_random_uuid().
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto;`).Error; err != nil {
		return err
	}

	// If an earlier version created id as TEXT, convert it to UUID.
	if db.Migrator().HasTable(&qrCodeRow{}) {
		if err := db.Exec(`ALTER TABLE qr_codes ALTER COLUMN id TYPE uuid USING id::uuid;`).Error; err != nil {
			return err
		}
		if err := db.Exec(`ALTER TABLE qr_codes ALTER COLUMN id SET DEFAULT gen_random_uuid();`).Error; err != nil {
			return err
		}

		// Add active flag (default true) for existing rows.
		if err := db.Exec(`ALTER TABLE qr_codes ADD COLUMN IF NOT EXISTS active boolean NOT NULL DEFAULT true;`).Error; err != nil {
			return err
		}
	}

	if err := db.AutoMigrate(&qrCodeRow{}); err != nil {
		return err
	}
	return db.AutoMigrate(&settingsRow{})
}

func (s *PostgresStore) List() []model.QrCode {
	rows := make([]qrCodeRow, 0, 32)
	if err := s.db.Order("created_at desc").Find(&rows).Error; err != nil {
		return []model.QrCode{}
	}

	items := make([]model.QrCode, 0, len(rows))
	for _, r := range rows {
		items = append(items, model.QrCode{ID: r.ID.String(), Label: r.Label, URL: r.URL, Active: r.Active, CreatedAt: r.CreatedAt})
	}
	return items
}

func (s *PostgresStore) Get(id string) (model.QrCode, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.QrCode{}, ErrNotFound
	}

	var r qrCodeRow
	err = s.db.First(&r, "id = ?", uid).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.QrCode{}, ErrNotFound
		}
		return model.QrCode{}, err
	}
	return model.QrCode{ID: r.ID.String(), Label: r.Label, URL: r.URL, Active: r.Active, CreatedAt: r.CreatedAt}, nil
}

func (s *PostgresStore) Create(input CreateInput) (model.QrCode, error) {
	id := uuid.New()
	active := true
	if input.Active != nil {
		active = *input.Active
	}

	q := model.QrCode{
		ID:        id.String(),
		Label:     input.Label,
		URL:       input.URL,
		Active:    active,
		CreatedAt: time.Now().UTC(),
	}
	if q.Label == "" {
		q.Label = "Untitled"
	}

	r := qrCodeRow{ID: id, Label: q.Label, URL: q.URL, Active: q.Active, CreatedAt: q.CreatedAt}
	if err := s.db.Create(&r).Error; err != nil {
		return model.QrCode{}, err
	}
	return q, nil
}

func (s *PostgresStore) Update(id string, input UpdateInput) (model.QrCode, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.QrCode{}, ErrNotFound
	}

	// Load first so we can map not-found cleanly.
	current, err := s.Get(id)
	if err != nil {
		return model.QrCode{}, err
	}

	if input.Label != nil {
		current.Label = *input.Label
	}
	if input.URL != nil {
		current.URL = *input.URL
	}
	if input.Active != nil {
		current.Active = *input.Active
	}
	if current.Label == "" {
		current.Label = "Untitled"
	}

	updates := map[string]any{"label": current.Label, "url": current.URL, "active": current.Active}
	if err := s.db.Model(&qrCodeRow{}).Where("id = ?", uid).Updates(updates).Error; err != nil {
		return model.QrCode{}, err
	}
	return current, nil
}

func (s *PostgresStore) Delete(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return ErrNotFound
	}

	res := s.db.Delete(&qrCodeRow{}, "id = ?", uid)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) CountTotal() (int, error) {
	var n int64
	if err := s.db.Model(&qrCodeRow{}).Count(&n).Error; err != nil {
		return 0, err
	}
	return int(n), nil
}

func (s *PostgresStore) CountActive() (int, error) {
	var n int64
	if err := s.db.Model(&qrCodeRow{}).Where("active = ?", true).Count(&n).Error; err != nil {
		return 0, err
	}
	return int(n), nil
}

func (s *PostgresStore) GetSettings() (model.UserSettings, error) {
	var row settingsRow
	err := s.db.FirstOrCreate(&row, settingsRow{ID: 1}).Error
	if err != nil {
		return model.UserSettings{}, err
	}
	return model.UserSettings{DefaultRedirectURL: row.DefaultRedirectURL}, nil
}

func (s *PostgresStore) UpdateSettings(settings model.UserSettings) error {
	return s.db.Model(&settingsRow{}).Where("id = ?", 1).Update("default_redirect_url", settings.DefaultRedirectURL).Error
}
