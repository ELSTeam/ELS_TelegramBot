package main

import (
	"fmt"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserState int

const (
	Initial UserState = iota
	WaitingForUsername
	WaitingForPassword
	Connected
)

type User struct {
	Username string
	Password string
	State    UserState
}

var users = make(map[int64]*User)

// server_url := "http://localhost:5000"

func main() {
	fmt.Println("Getting the APi token from os.env")
	API_TOKEN := os.Getenv("API_TOKEN")
	if API_TOKEN == "" {
		log.Panic("No API_TOKEN found")
	}
	bot, err := tgbotapi.NewBotAPI(API_TOKEN)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true
	fmt.Println("Bot is up and running")

	//setting up channel updates (open a channel and get updates)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		userID := update.Message.Chat.ID

		// check if the user is already in the bot cache, if not add him in initiliaze state
		if users[userID] == nil {
			users[userID] = &User{State: Initial}
		}
		user := users[userID]
		switch user.State {
		case Initial:
			if update.Message.Text == "/signin" {
				user.State = WaitingForUsername
				msg := tgbotapi.NewMessage(userID, "Please enter your username:")
				_, err := bot.Send(msg)
				if err != nil {
					log.Println("Error sending message:", err)
				}
			} else {
				msg := tgbotapi.NewMessage(userID, "Please sign in first by typing /signin")
				_, err := bot.Send(msg)
				if err != nil {
					log.Println("Error sending message:", err)
				}
			}
		case WaitingForUsername:
			user.State = WaitingForPassword
			msg := tgbotapi.NewMessage(userID, "Please enter your password:")
			_, err := bot.Send(msg)
			if err != nil {
				log.Println("Error sending message:", err)
			}
		case WaitingForPassword:
			// connect to the server and check for credential.
			connected := true
			if connected == true {
				user.State = Connected
				msg := tgbotapi.NewMessage(userID, "Connected.")
				_, err := bot.Send(msg)
				if err != nil {
					log.Println("Error sending message:", err)
				}
			} else {
				user.State = Initial
				msg := tgbotapi.NewMessage(userID, "Wrong creds..")
				_, err := bot.Send(msg)
				if err != nil {
					log.Println("Error sending message:", err)
				}
			}
		}
	}
}
