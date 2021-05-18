package db

import (
	"time"

	"github.com/google/uuid"
)

// Medication information for a user
type Medication struct {
	IDUser                  uuid.UUID `json:"id_user"`
	ID                      uuid.UUID `json:"id"`
	Name                    string    `json:"name"`
	IntervalCrontab         string    `json:"interval_crontab"`
	IntervalQuantity        uint      `json:"interval_quantity"`
	IntervalPushoverDevices []string  `json:"interval_pushover_devices"`
	CreatedAt               time.Time `json:"created_at"`
}

func (m *Medication) badgerKey() []byte {
	return append(append([]byte("medication:"), m.IDUser[:]...), m.ID[:]...)
}

func badgerPrefixKeyForMedicationUser(user *User) []byte {
	return append([]byte("medication:"), user.ID[:]...)
}
