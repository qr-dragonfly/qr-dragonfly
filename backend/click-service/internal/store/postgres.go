package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PostgresStore struct {
	db *gorm.DB
}

type clickDailyStatsRow struct {
	QrCodeID     string    `gorm:"primaryKey;not null"`
	Day          time.Time `gorm:"primaryKey;type:date;not null"`
	Total        int       `gorm:"not null;default:0"`
	RegionCounts []byte    `gorm:"column:region_counts;type:jsonb"`
	Hour00       int       `gorm:"column:hour00;not null;default:0"`
	Hour01       int       `gorm:"column:hour01;not null;default:0"`
	Hour02       int       `gorm:"column:hour02;not null;default:0"`
	Hour03       int       `gorm:"column:hour03;not null;default:0"`
	Hour04       int       `gorm:"column:hour04;not null;default:0"`
	Hour05       int       `gorm:"column:hour05;not null;default:0"`
	Hour06       int       `gorm:"column:hour06;not null;default:0"`
	Hour07       int       `gorm:"column:hour07;not null;default:0"`
	Hour08       int       `gorm:"column:hour08;not null;default:0"`
	Hour09       int       `gorm:"column:hour09;not null;default:0"`
	Hour10       int       `gorm:"column:hour10;not null;default:0"`
	Hour11       int       `gorm:"column:hour11;not null;default:0"`
	Hour12       int       `gorm:"column:hour12;not null;default:0"`
	Hour13       int       `gorm:"column:hour13;not null;default:0"`
	Hour14       int       `gorm:"column:hour14;not null;default:0"`
	Hour15       int       `gorm:"column:hour15;not null;default:0"`
	Hour16       int       `gorm:"column:hour16;not null;default:0"`
	Hour17       int       `gorm:"column:hour17;not null;default:0"`
	Hour18       int       `gorm:"column:hour18;not null;default:0"`
	Hour19       int       `gorm:"column:hour19;not null;default:0"`
	Hour20       int       `gorm:"column:hour20;not null;default:0"`
	Hour21       int       `gorm:"column:hour21;not null;default:0"`
	Hour22       int       `gorm:"column:hour22;not null;default:0"`
	Hour23       int       `gorm:"column:hour23;not null;default:0"`
	LastAt       time.Time `gorm:"not null"`
	LastCountry  string    `gorm:"not null;default:''"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (clickDailyStatsRow) TableName() string { return "click_daily_stats" }

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	s := &PostgresStore{db: db}
	if err := s.ensureSchema(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *PostgresStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *PostgresStore) ensureSchema(ctx context.Context) error {
	return s.db.WithContext(ctx).AutoMigrate(&clickDailyStatsRow{})
}

func (s *PostgresStore) RecordClick(event ClickEvent) error {
	t := event.At.UTC()
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	hour := t.Hour()
	if hour < 0 || hour > 23 {
		return errors.New("invalid hour")
	}

	hourCol := fmt.Sprintf("hour%02d", hour)

	// Atomic upsert: creates the per-day row on first click; increments the matching hour column per click.
	sql := fmt.Sprintf(
		`INSERT INTO click_daily_stats (qr_code_id, day, total, %s, last_at, last_country, region_counts, created_at, updated_at)
		 VALUES (?, ?, 1, 1, ?, ?, CASE WHEN ? <> '' THEN jsonb_build_object(?, 1) ELSE '{}'::jsonb END, now(), now())
		 ON CONFLICT (qr_code_id, day)
		 DO UPDATE SET
		   total = click_daily_stats.total + 1,
		   %s = click_daily_stats.%s + 1,
		   last_at = GREATEST(click_daily_stats.last_at, EXCLUDED.last_at),
		   last_country = CASE WHEN EXCLUDED.last_at >= click_daily_stats.last_at THEN EXCLUDED.last_country ELSE click_daily_stats.last_country END,
		   region_counts = CASE
		     WHEN EXCLUDED.last_country <> '' THEN
		       jsonb_set(
		         COALESCE(click_daily_stats.region_counts, '{}'::jsonb),
		         ARRAY[EXCLUDED.last_country]::text[],
		         to_jsonb(
		           COALESCE((COALESCE(click_daily_stats.region_counts, '{}'::jsonb)->>EXCLUDED.last_country)::int, 0) + 1
		         ),
		         true
		       )
		     ELSE click_daily_stats.region_counts
		   END,
		   updated_at = now()`,
		hourCol, hourCol, hourCol,
	)

	return s.db.Exec(sql, event.QrCodeID, day, t, event.Country, event.Country, event.Country).Error
}

func (s *PostgresStore) GetStats(qrCodeID string) (ClickStats, error) {
	type agg struct {
		Total int64
	}
	var a agg
	if err := s.db.Model(&clickDailyStatsRow{}).
		Select("COALESCE(SUM(total), 0) AS total").
		Where("qr_code_id = ?", qrCodeID).
		Scan(&a).Error; err != nil {
		return ClickStats{}, err
	}
	if a.Total == 0 {
		return ClickStats{}, ErrNotFound
	}

	var last clickDailyStatsRow
	if err := s.db.Where("qr_code_id = ?", qrCodeID).Order("last_at desc").Limit(1).Find(&last).Error; err != nil {
		return ClickStats{}, err
	}
	if last.QrCodeID == "" {
		return ClickStats{}, ErrNotFound
	}

	return ClickStats{QrCodeID: qrCodeID, Total: int(a.Total), LastAtIso: last.LastAt.UTC().Format(time.RFC3339), LastCountry: last.LastCountry}, nil
}

func (s *PostgresStore) GetDaily(qrCodeID string, day time.Time) (DailyClickStats, error) {
	day = day.UTC()
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)

	var row clickDailyStatsRow
	err := s.db.Where("qr_code_id = ? AND day = ?", qrCodeID, day).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DailyClickStats{}, ErrNotFound
		}
		return DailyClickStats{}, err
	}

	var regionCounts map[string]int
	if len(row.RegionCounts) > 0 {
		_ = json.Unmarshal(row.RegionCounts, &regionCounts)
		if len(regionCounts) == 0 {
			regionCounts = nil
		}
	}

	return DailyClickStats{
		QrCodeID:     qrCodeID,
		DayIso:       row.Day.UTC().Format("2006-01-02"),
		Total:        row.Total,
		RegionCounts: regionCounts,
		Hour00:       row.Hour00,
		Hour01:       row.Hour01,
		Hour02:       row.Hour02,
		Hour03:       row.Hour03,
		Hour04:       row.Hour04,
		Hour05:       row.Hour05,
		Hour06:       row.Hour06,
		Hour07:       row.Hour07,
		Hour08:       row.Hour08,
		Hour09:       row.Hour09,
		Hour10:       row.Hour10,
		Hour11:       row.Hour11,
		Hour12:       row.Hour12,
		Hour13:       row.Hour13,
		Hour14:       row.Hour14,
		Hour15:       row.Hour15,
		Hour16:       row.Hour16,
		Hour17:       row.Hour17,
		Hour18:       row.Hour18,
		Hour19:       row.Hour19,
		Hour20:       row.Hour20,
		Hour21:       row.Hour21,
		Hour22:       row.Hour22,
		Hour23:       row.Hour23,
	}, nil
}

func (s *PostgresStore) GetDailyBatch(qrCodeID string, days []time.Time) (map[string]DailyClickStats, error) {
	if len(days) == 0 {
		return map[string]DailyClickStats{}, nil
	}

	// Normalize days to UTC date boundaries
	normalizedDays := make([]time.Time, len(days))
	for i, day := range days {
		normalizedDays[i] = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	}

	var rows []clickDailyStatsRow
	err := s.db.Where("qr_code_id = ? AND day IN ?", qrCodeID, normalizedDays).Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]DailyClickStats)
	for _, row := range rows {
		var regionCounts map[string]int
		if len(row.RegionCounts) > 0 {
			_ = json.Unmarshal(row.RegionCounts, &regionCounts)
			if len(regionCounts) == 0 {
				regionCounts = nil
			}
		}

		dayIso := row.Day.UTC().Format("2006-01-02")
		result[dayIso] = DailyClickStats{
			QrCodeID:     qrCodeID,
			DayIso:       dayIso,
			Total:        row.Total,
			RegionCounts: regionCounts,
			Hour00:       row.Hour00,
			Hour01:       row.Hour01,
			Hour02:       row.Hour02,
			Hour03:       row.Hour03,
			Hour04:       row.Hour04,
			Hour05:       row.Hour05,
			Hour06:       row.Hour06,
			Hour07:       row.Hour07,
			Hour08:       row.Hour08,
			Hour09:       row.Hour09,
			Hour10:       row.Hour10,
			Hour11:       row.Hour11,
			Hour12:       row.Hour12,
			Hour13:       row.Hour13,
			Hour14:       row.Hour14,
			Hour15:       row.Hour15,
			Hour16:       row.Hour16,
			Hour17:       row.Hour17,
			Hour18:       row.Hour18,
			Hour19:       row.Hour19,
			Hour20:       row.Hour20,
			Hour21:       row.Hour21,
			Hour22:       row.Hour22,
			Hour23:       row.Hour23,
		}
	}

	return result, nil
}
