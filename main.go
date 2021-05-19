package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"git.0xdad.com/tblyler/meditime/config"
	"git.0xdad.com/tblyler/meditime/db"
	"github.com/google/uuid"
	"github.com/gregdel/pushover"
	"github.com/robfig/cron/v3"
)

func errLog(messages ...interface{}) {
	fmt.Fprintln(os.Stderr, messages...)
}

func log(messages ...interface{}) {
	fmt.Println(messages...)
}

func help() {
}

func run(ctx context.Context, b *db.Badger, pushoverClient *pushover.Pushover) error {
	cron := cron.New()

	users, err := b.ListUsers()
	if err != nil {
		return err
	}

	for _, user := range users {
		medications, err := b.ListMedicationsForUser(user)
		if err != nil {
			return err
		}

		for _, medication := range medications {
			_, err = cron.AddFunc(medication.IntervalCrontab, func() {
				for _, device := range medication.IntervalPushoverDevices {
					deviceToken, ok := user.PushoverDeviceTokens[device]
					if !ok {
						errLog(fmt.Sprintf(
							"invalid device name %s for id medication %s for id user %s",
							device,
							medication.ID.String(),
							user.ID.String(),
						))
						continue
					}

					_, err := pushoverClient.SendMessage(
						&pushover.Message{
							Message:  fmt.Sprintf("take %d dose(s) of %s", medication.IntervalQuantity, medication.Name),
							Priority: pushover.PriorityEmergency,
							Retry:    time.Minute * 5,
							Expire:   time.Hour * 24,
						},
						pushover.NewRecipient(deviceToken),
					)
					if err != nil {
						errLog(fmt.Sprintf(
							"failed to send message to id user's (%s) device (%s): %v",
							user.ID.String(),
							device,
							err,
						))
					}
				}
			})
			if err != nil {
				return fmt.Errorf("failed to add medication ID %s to cron: %w", medication.ID.String(), err)
			}
		}
	}

	cron.Start()
	<-ctx.Done()
	<-cron.Stop().Done()

	return ctx.Err()
}

func main() {
	lenArgs := len(os.Args)
	if lenArgs <= 1 {
		help()
		errLog("must supply at least one argument")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	err := func() error {
		inputScanner := bufio.NewScanner(os.Stdin)
		config := config.Env{}

		badgerPath, err := config.BadgerPath()
		if err != nil {
			return err
		}

		pushoverAPIToken, err := config.PushoverAPIToken()
		if err != nil {
			return err
		}

		b, err := db.NewBadger(badgerPath)
		if err != nil {
			return err
		}

		defer b.Close()

		pushoverClient := pushover.New(pushoverAPIToken)

		switch os.Args[1] {
		case "run":
			return run(ctx, b, pushoverClient)

		case "user":
			if lenArgs < 3 {
				return errors.New("must supply an argument to the user command")
			}

			switch os.Args[2] {
			case "add":
				fmt.Print("username: ")
				inputScanner.Scan()

				username := string(bytes.TrimSpace(inputScanner.Bytes()))
				if username == "" {
					return fmt.Errorf("failed to get username from STDIN prompt: %w", inputScanner.Err())
				}

				fmt.Print("pushover device token: ")
				inputScanner.Scan()

				deviceToken := string(bytes.TrimSpace(inputScanner.Bytes()))
				if deviceToken == "" {
					return fmt.Errorf("no pushover device token provided: %w", inputScanner.Err())
				}

				id := uuid.New()

				err = b.AddUser(&db.User{
					ID:   id,
					Name: username,
					PushoverDeviceTokens: map[string]string{
						"default": deviceToken,
					},
					CreatedAt: time.Now(),
				})
				if err != nil {
					return fmt.Errorf("failed to insert username %s: %w", username, err)
				}

				log("created user id", id)

			case "get":
				fmt.Print("username: ")
				inputScanner.Scan()

				username := string(bytes.TrimSpace(inputScanner.Bytes()))
				if username == "" {
					return fmt.Errorf("failed to get username from STDIN prompt: %w", inputScanner.Err())
				}

				user, err := b.GetUser(username)
				if err != nil {
					return err
				}

				if user == nil {
					return fmt.Errorf("username %s does not exist", username)
				}

				log(user)

			case "list":
				users, err := b.ListUsers()
				if err != nil {
					return err
				}

				for _, user := range users {
					fmt.Println(user)
				}
			}

		case "medication":
			if lenArgs < 3 {
				return errors.New("must supply an argument to the user command")
			}

			switch os.Args[2] {
			case "add":
				fmt.Print("username: ")
				inputScanner.Scan()

				username := string(bytes.TrimSpace(inputScanner.Bytes()))
				if username == "" {
					return fmt.Errorf("failed to get username from STDIN prompt: %w", inputScanner.Err())
				}

				user, err := b.GetUser(username)
				if err != nil {
					return fmt.Errorf("failed to lookup username %s: %w", username, err)
				}

				if user == nil {
					return fmt.Errorf("username %s doesn't exist", username)
				}

				fmt.Print("name: ")
				inputScanner.Scan()

				name := string(bytes.TrimSpace(inputScanner.Bytes()))
				if name == "" {
					return fmt.Errorf("failed to get medication name from STDIN prompt: %w", inputScanner.Err())
				}

				fmt.Print("cron schedule: ")
				inputScanner.Scan()

				crontab := string(bytes.TrimSpace(inputScanner.Bytes()))
				if crontab == "" {
					return fmt.Errorf("failed to get cron schedule from STDIN prompt: %w", inputScanner.Err())
				}

				fmt.Print("interval quantity: ")
				inputScanner.Scan()

				intervalQuantity, err := strconv.ParseUint(string(bytes.TrimSpace(inputScanner.Bytes())), 10, 64)
				if intervalQuantity == 0 || err != nil {
					return fmt.Errorf("failed to get interval quantity from STDIN prompt: %w", inputScanner.Err())
				}

				fmt.Print("interval pushover device token name: ")
				inputScanner.Scan()

				intervalPushoverDevice := string(bytes.TrimSpace(inputScanner.Bytes()))
				if _, ok := user.PushoverDeviceTokens[intervalPushoverDevice]; !ok {
					return fmt.Errorf("the '%s' pushover device token name doesn't exist for user %s", intervalPushoverDevice, user.Name)
				}

				medication := &db.Medication{
					IDUser:                  user.ID,
					ID:                      uuid.New(),
					Name:                    name,
					IntervalCrontab:         crontab,
					IntervalQuantity:        uint(intervalQuantity),
					IntervalPushoverDevices: []string{intervalPushoverDevice},
					CreatedAt:               time.Now(),
				}

				err = b.AddMedication(medication)
				if err != nil {
					return err
				}

				log(medication)

			case "remove":
				fmt.Print("username: ")
				inputScanner.Scan()

				username := string(bytes.TrimSpace(inputScanner.Bytes()))
				if username == "" {
					return fmt.Errorf("failed to get username from STDIN prompt: %w", inputScanner.Err())
				}

				user, err := b.GetUser(username)
				if err != nil {
					return fmt.Errorf("failed to lookup username %s: %w", username, err)
				}

				if user == nil {
					return fmt.Errorf("username %s doesn't exist", username)
				}

				fmt.Print("medication id: ")
				inputScanner.Scan()

				medicationID := uuid.MustParse(string(bytes.TrimSpace(inputScanner.Bytes())))
				err = b.RemoveMedication(&db.Medication{
					IDUser: user.ID,
					ID:     medicationID,
				})
				if err != nil {
					return err
				}

			case "list":
				fmt.Print("username: ")
				inputScanner.Scan()

				username := string(bytes.TrimSpace(inputScanner.Bytes()))
				if username == "" {
					return fmt.Errorf("failed to get username from STDIN prompt: %w", inputScanner.Err())
				}

				user, err := b.GetUser(username)
				if err != nil {
					return fmt.Errorf("failed to lookup username %s: %w", username, err)
				}

				if user == nil {
					return fmt.Errorf("username %s doesn't exist", username)
				}

				medications, err := b.ListMedicationsForUser(user)
				if err != nil {
					return err
				}

				for _, medication := range medications {
					log(medication)
				}
			}
		}

		return nil
	}()

	if err != nil {
		errLog(err.Error())
		os.Exit(1)
	}
}
