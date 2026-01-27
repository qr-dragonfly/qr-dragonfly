package store

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type ClickEvent struct {
	At         time.Time `json:"-"`
	AtIso      string    `json:"atIso"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"userAgent"`
	Referer    string    `json:"referer"`
	Country    string    `json:"country"`
	RequestID  string    `json:"requestId"`
	QrCodeID   string    `json:"qrCodeId"`
	TargetURL  string    `json:"targetUrl"`
	UserType   string    `json:"userType,omitempty"`
	AcceptLang string    `json:"acceptLanguage,omitempty"`
}

type ClickStats struct {
	QrCodeID    string `json:"qrCodeId"`
	Total       int    `json:"total"`
	LastAtIso   string `json:"lastAtIso,omitempty"`
	LastCountry string `json:"lastCountry,omitempty"`
}

type DailyClickStats struct {
	QrCodeID     string         `json:"qrCodeId"`
	DayIso       string         `json:"dayIso"`
	Total        int            `json:"total"`
	RegionCounts map[string]int `json:"regionCounts,omitempty"`
	Hour00       int            `json:"hour00"`
	Hour01       int            `json:"hour01"`
	Hour02       int            `json:"hour02"`
	Hour03       int            `json:"hour03"`
	Hour04       int            `json:"hour04"`
	Hour05       int            `json:"hour05"`
	Hour06       int            `json:"hour06"`
	Hour07       int            `json:"hour07"`
	Hour08       int            `json:"hour08"`
	Hour09       int            `json:"hour09"`
	Hour10       int            `json:"hour10"`
	Hour11       int            `json:"hour11"`
	Hour12       int            `json:"hour12"`
	Hour13       int            `json:"hour13"`
	Hour14       int            `json:"hour14"`
	Hour15       int            `json:"hour15"`
	Hour16       int            `json:"hour16"`
	Hour17       int            `json:"hour17"`
	Hour18       int            `json:"hour18"`
	Hour19       int            `json:"hour19"`
	Hour20       int            `json:"hour20"`
	Hour21       int            `json:"hour21"`
	Hour22       int            `json:"hour22"`
	Hour23       int            `json:"hour23"`
}

type Store interface {
	RecordClick(event ClickEvent) error
	GetStats(qrCodeID string) (ClickStats, error)
	GetDaily(qrCodeID string, day time.Time) (DailyClickStats, error)
	GetDailyBatch(qrCodeID string, days []time.Time) (map[string]DailyClickStats, error)
}
