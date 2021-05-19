package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger"
)

// Badger db implementation
type Badger struct {
	db       *badger.DB
	cancelGC func()
	wg       sync.WaitGroup
}

// NewBadger creates a new badger instance for the given path
func NewBadger(dbPath string) (*Badger, error) {
	db, err := badger.Open(badger.DefaultOptions(dbPath).WithLogger(nil))
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db at path %s: %w", dbPath, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	b := &Badger{
		db:       db,
		cancelGC: cancel,
	}

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()

		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				for b.db.RunValueLogGC(0.5) == nil && ctx.Err() == nil {
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return b, nil
}

// Close the database
func (b *Badger) Close() error {
	b.cancelGC()
	b.wg.Wait()

	return b.db.Close()
}

// AddUser to the database
func (b *Badger) AddUser(user *User) error {
	return b.db.Update(func(tx *badger.Txn) error {
		data, err := json.Marshal(user)
		if err != nil {
			return fmt.Errorf("failed to JSON marshal user: %w", err)
		}

		key := user.badgerKey()
		if _, err = tx.Get(key); err == nil {
			return fmt.Errorf("user %s already exists", user.Name)
		}

		return tx.Set(key, data)
	})
}

// GetUser from the database
func (b *Badger) GetUser(username string) (user *User, err error) {
	err = b.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(badgerKeyForUsername(username))
		if err != nil {
			return fmt.Errorf("failed to get user value for username %s: %w", username, err)
		}

		user = &User{}

		return item.Value(func(val []byte) error {
			err = json.Unmarshal(val, user)
			if err != nil {
				return fmt.Errorf("failed to unmarshal user value for username %s: %w", username, err)
			}

			return nil
		})
	})

	return
}

// ListUsers from the database
func (b *Badger) ListUsers() (users []*User, err error) {
	err = b.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("user:")

		it := tx.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			err := item.Value(func(val []byte) error {
				user := &User{}
				err := json.Unmarshal(val, user)
				if err != nil {
					return fmt.Errorf("failed to unmarshal user value for user key %s: %w", string(item.Key()), err)
				}

				users = append(users, user)

				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	return
}

// AddMedication to the database
func (b *Badger) AddMedication(medication *Medication) error {
	return b.db.Update(func(tx *badger.Txn) error {
		data, err := json.Marshal(medication)
		if err != nil {
			return fmt.Errorf("failed to JSON marshal medication: %w", err)
		}

		return tx.Set(medication.badgerKey(), data)
	})
}

// RemoveMedication from the database
func (b *Badger) RemoveMedication(medication *Medication) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return tx.Delete(medication.badgerKey())
	})
}

// ListMedicationsForUser from the database
func (b *Badger) ListMedicationsForUser(user *User) (medications []*Medication, err error) {
	err = b.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = badgerPrefixKeyForMedicationUser(user)

		it := tx.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			err := item.Value(func(val []byte) error {
				medication := &Medication{}
				err := json.Unmarshal(val, medication)
				if err != nil {
					return fmt.Errorf("failed to unmarshal medication value for medication key %s: %w", string(item.Key()), err)
				}

				medications = append(medications, medication)

				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	return
}
