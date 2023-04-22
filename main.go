package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

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

type Contact struct {
	Name  string `json:"name"`
	Phone string `json:phone`
	Email string `json:email`
}

var users = make(map[int64]*User)

var server_url = "http://localhost:5000"

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
	// bot.Debug = true
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
			if update.Message.Text == "/start" {
				user.State = WaitingForUsername
				msg := tgbotapi.NewMessage(userID, "Please enter your username:")
				_, err := bot.Send(msg)
				if err != nil {
					log.Println("Error sending message:", err)
				}
			} else {
				msg := tgbotapi.NewMessage(userID, "Please sign in first by typing /start")
				_, err := bot.Send(msg)
				if err != nil {
					log.Println("Error sending message:", err)
				}
			}
		case WaitingForUsername:
			user.Username = update.Message.Text
			user.State = WaitingForPassword
			msg := tgbotapi.NewMessage(userID, "Please enter your password:")
			_, err := bot.Send(msg)
			if err != nil {
				log.Println("Error sending message:", err)
			}
		case WaitingForPassword:
			user.Password = update.Message.Text
			// connect to the server and check for credential.
			connected, err := checkLogin(user.Username, user.Password)
			if err != nil {
				log.Panic(err)
			}
			if connected == 200 {
				user.State = Connected
				msg := tgbotapi.NewMessage(userID, "Welcome "+user.Username)
				_, err := bot.Send(msg)
				if err != nil {
					log.Println("Error sending message:", err)
				}
				msg = tgbotapi.NewMessage(userID, "Menu")
				_, err = bot.Send(msg)
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
		case Connected:
			if update.Message.Text == "getAllContacts" {
				contacts, err := getAllContacts(user.Username)
				if err != nil {
					log.Println("Error getting contacts:", err)
				}
				var message_to_user string
				for index, contact := range contacts {
					message_to_user += strconv.Itoa(index+1) + ")"
					message_to_user += "\n"
					message_to_user += "Name: " + contact.Name + "\n"
					message_to_user += "Phone: " + contact.Phone + "\n"
					message_to_user += "Email: " + contact.Email + "\n"
				}
				msg := tgbotapi.NewMessage(userID, message_to_user)
				_, err = bot.Send(msg)
				if err != nil {
					log.Println("Error sending message:", err)
				}
			}

		}
	}
}

func checkLogin(username string, password string) (int, error) {
	log.Println("checklogin")
	log.Println(username)
	log.Println(password)
	// Create a struct representing the JSON payload
	payload := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: username,
		Password: password,
	}

	// Encode the payload into a JSON string
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	// Create a new HTTP POST request with the payload as the request body
	req, err := http.NewRequest("POST", server_url+"/sign_in", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return 0, err
	}

	// Set the request header to specify that the payload is in JSON format
	req.Header.Set("Content-Type", "application/json")

	// Send the HTTP request using the default HTTP client
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}
	defer resp.Body.Close()

	// Return the status code of the HTTP response
	return resp.StatusCode, nil
}

func getAllContacts(username string) ([]Contact, error) {
	log.Println("check_all_contacts")
	log.Println(username)

	payload := struct {
		Username string `json:"username"`
	}{
		Username: username,
	}
	payload_bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", server_url+"/all_contacts", bytes.NewBuffer(payload_bytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	var contactList struct {
		Contacts []Contact `json:"contacts"`
	}

	// Decode the return into the contactList struct
	err = json.NewDecoder(resp.Body).Decode(&contactList)
	if err != nil {
		return nil, err
	}
	return contactList.Contacts, nil
}