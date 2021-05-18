package db

import (
	"time"

	"github.com/google/uuid"
)

// User information
type User struct {
	ID                   uuid.UUID         `json:"id"`
	Name                 string            `json:"name"`
	PushoverDeviceTokens map[string]string `json:"pushover_device_tokens"`
	CreatedAt            time.Time         `json:"created_at"`
}

func (u *User) badgerKey() []byte {
	return badgerKeyForUsername(u.Name)
}

func badgerKeyForUsername(username string) []byte {
	return append([]byte("user:"), []byte(username)...)
}
