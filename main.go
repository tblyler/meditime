package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"git.0xdad.com/tblyler/meditime/config"
	"git.0xdad.com/tblyler/meditime/db"
	"github.com/google/uuid"
)

func errLog(messages ...interface{}) {
	fmt.Fprintln(os.Stderr, messages...)
}

func log(messages ...interface{}) {
	fmt.Println(messages...)
}

func help() {
}

func run(b *db.Badger) error {
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
			// FIXME do the stuff
			fmt.Println(user)
			fmt.Println(medication)
		}
	}

	return nil
}

func main() {
	lenArgs := len(os.Args)
	if lenArgs <= 1 {
		help()
		errLog("must supply at least one argument")
		os.Exit(1)
	}

	err := func() error {
		inputScanner := bufio.NewScanner(os.Stdin)
		config := config.Env{}

		badgerPath, err := config.BadgerPath()
		if err != nil {
			return err
		}

		b, err := db.NewBadger(badgerPath)
		if err != nil {
			return err
		}

		defer b.Close()

		switch os.Args[1] {
		case "run":
			return run(b)

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

				fmt.Print("interval pushover device token id: ")
				inputScanner.Scan()

				intervalPushoverDevice := string(bytes.TrimSpace(inputScanner.Bytes()))
				if _, ok := user.PushoverDeviceTokens[intervalPushoverDevice]; !ok {
					return fmt.Errorf("the '%s' pushover device token doesn't exist for user %s", intervalPushoverDevice, user.Name)
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
