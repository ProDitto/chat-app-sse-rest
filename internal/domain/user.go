package domain

import "time"

type User struct {
	UserID               string    `json:"userId"`
	Username             string    `json:"username"`
	LastActive           time.Time `json:"-"`
	IsOnline             bool      `json:"isOnline"`
	LastDeliveredEventID string    `json:"-"`
}

