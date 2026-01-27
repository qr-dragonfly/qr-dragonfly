package model

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name,omitempty"`
	UserType     string    `json:"userType,omitempty"`
	Entitlements string    `json:"entitlements,omitempty"`
	CreatedAt    time.Time `json:"-"`
	CreatedAtIso string    `json:"createdAtIso,omitempty"`
}

func (u User) NormalizeForResponse() User {
	if !u.CreatedAt.IsZero() {
		u.CreatedAtIso = u.CreatedAt.UTC().Format(time.RFC3339)
	}
	return u
}
