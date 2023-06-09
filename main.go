package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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
	Username        string
	Password        string
	ChatID          int64
	State           UserState
	AddContactState ContactAddState
}

type Contact struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

// Struct for the function Add contact.
type ContactAddToUser struct {
	Username    string  `json:"username"`
	ContactInfo Contact `json:"contact_info"`
}

// struct for adding new contact status
type ContactAddState int

const (
	InitialAdd ContactAddState = iota
	WaitingForUsernameAdd
	WaitingForPhoneAdd
	WaitingForEmailAdd
	ConfirmationAdd
	PositiveFullAdd
	NegaiveFullAdd
	Done
)

type Server_alert struct {
	Username string `json:"username"`
}
type Vidoe_alert struct {
	Data []byte `json:"data"`
}

// contact instance to save the data from user bot.
var global_contact Contact

var users = make(map[int64]*User)
var usernames_to_user_map = make(map[string]*User)

var bot *tgbotapi.BotAPI

var server_url = "http://localhost:5000"

func main() {
	middleContactCreation := false
	fmt.Println("Getting the APi token from os.env")
	API_TOKEN := os.Getenv("API_TOKEN")
	if API_TOKEN == "" {
		log.Panic("No API_TOKEN found")
	}

	fmt.Println("Setting up the http server")
	http.HandleFunc("/fall_telegram", fall_handler)
	go http.ListenAndServe(":8090", nil)

	var err error
	bot, err = tgbotapi.NewBotAPI(API_TOKEN)
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
		fmt.Println(user.State)

		// Checking the user start, the user state flow should be initial -> waitingForUsername -> waitingForPassword -> Connected.
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
				msg := tgbotapi.NewMessage(userID, "Welcome "+user.Username+"\n")

				// Add the register user to the map
				usernames_to_user_map[user.Username] = user
				user.ChatID = userID
				_, err := bot.Send(msg)
				if err != nil {
					log.Println("Error sending message:", err)
				}
				menu_string, error := get_menu(user.Username)
				if error != nil {
					log.Println("Error getting menu message")
					menu_string = "Error"
				}
				user.AddContactState = Done
				msg = tgbotapi.NewMessage(userID, menu_string)
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

		//user is connected now. functionality is available
		case Connected:
			fmt.Println(update.Message.Text)
			if update.Message.Text == "/getAllContacts" {
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
			} else if update.Message.Text == "/menu" {
				msg_value, err := get_menu(user.Username)
				if err != nil {
					log.Println(err)
					continue
				}
				msg := tgbotapi.NewMessage(userID, msg_value)
				_, err = bot.Send(msg)
				if err != nil {
					log.Println("Error sending message:", err)
				}
				//add contact flow is: initalAdd -> WaitingForUsernameAdd -> WaitingForPhoneAdd -> ConfirmationAdd -> Done
			} else if update.Message.Text == "/addcontact" {
				middleContactCreation = true
				user.AddContactState = InitialAdd
				msg := tgbotapi.NewMessage(userID, "Please enter contact data:")
				_, err := bot.Send(msg)
				if err != nil {
					log.Println("Error sending message:", err)
				}
			} else {
				if middleContactCreation == false {
					msg := tgbotapi.NewMessage(userID, "Please select valid option.\nClick /menu to show options.")
					_, err := bot.Send(msg)
					if err != nil {
						log.Println("Error sending message:", err)
					}
				}

			}
			// if add contact state is not done, then the user is in process to add contact
			if user.AddContactState != Done {
				switch user.AddContactState {
				case InitialAdd:
					user.AddContactState = WaitingForUsernameAdd
					msg := tgbotapi.NewMessage(userID, "Contact's name:")
					_, err := bot.Send(msg)
					if err != nil {
						log.Println("Error sending message:", err)
					}
				case WaitingForUsernameAdd:
					global_contact.Name = update.Message.Text
					user.AddContactState = WaitingForPhoneAdd
					msg := tgbotapi.NewMessage(userID, "Contact's phone:")
					_, err := bot.Send(msg)
					if err != nil {
						log.Println("Error sending message:", err)
					}
				case WaitingForPhoneAdd:
					global_contact.Phone = update.Message.Text
					user.AddContactState = WaitingForEmailAdd
					msg := tgbotapi.NewMessage(userID, "Contact's email:")
					_, err := bot.Send(msg)
					if err != nil {
						log.Println("Error sending message:", err)
					}
				case WaitingForEmailAdd:
					global_contact.Email = update.Message.Text
					user.AddContactState = ConfirmationAdd
					msg := tgbotapi.NewMessage(userID, "Are you sure to add the new contact? please type yes")
					_, err := bot.Send(msg)
					if err != nil {
						log.Println("Error sending message:", err)
					}
				case ConfirmationAdd:
					user.AddContactState = Done
					if update.Message.Text == "YES" || update.Message.Text == "yes" || update.Message.Text == "Yes" {
						fmt.Println("In positive add")
						status_code, err := addContact(user.Username, global_contact.Name, global_contact.Phone, global_contact.Email)
						if err != nil {
							log.Println("Error sending message: ", err)
						} else {
							if status_code == 200 {
								msg := tgbotapi.NewMessage(userID, "Contact Added.")
								_, err := bot.Send(msg)
								if err != nil {
									log.Println("Error sending message:", err)
								}
							} else {
								msg := tgbotapi.NewMessage(userID, "Error from server.")
								_, err := bot.Send(msg)
								if err != nil {
									log.Println("Error sending message:", err)
								}
							}
						}
					} else {
						msg := tgbotapi.NewMessage(userID, "Aborted.")
						_, err := bot.Send(msg)
						if err != nil {
							log.Println("Error sending message:", err)
						}
						user.AddContactState = Done
					}
					middleContactCreation = false

				}
			}

		}
	}
}

func checkLogin(username string, password string) (int, error) {
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

func addContact(username string, contactname string, contactphone string, contactmail string) (int, error) {
	log.Println("addcontact")
	payload := ContactAddToUser{
		Username: username,
		ContactInfo: Contact{
			Name:  contactname,
			Phone: contactphone,
			Email: contactmail,
		},
	}
	payload_bytes, err := json.Marshal(payload)
	if err != nil {
		return -1, err
	}
	req, err := http.NewRequest("PUT", server_url+"/add_contact", bytes.NewBuffer(payload_bytes))
	if err != nil {
		return -1, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func fall_handler(w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Println("Error from getting server updates")
	}
	var payload Server_alert
	err = json.Unmarshal(body, &payload)
	if err != nil {
		log.Println("Erro from decoding the data")
	}
	w.WriteHeader(http.StatusOK)
	falling_user := usernames_to_user_map[payload.Username]
	currentDateTime := time.Now().Format("2006-01-02 15:04:05")
	msg_text := "🚨 Alert: A fall detected! 🚨\n\n" +
		"Date and Time: " + currentDateTime + "\n" +
		"Review attached video footage for action.\n" +
		"Stay vigilant and act accordingly."
	msg := tgbotapi.NewMessage(falling_user.ChatID, msg_text)
	_, err = bot.Send(msg)
	if err != nil {
		log.Println("Error from sending falling alert in telegram", err)
	}
	if falling_user != nil {
		var video_url string
		video_url, err = getVideoFromServer(payload.Username)
		if err != nil {
			fmt.Println("Got error from getting the video from server")
			return
		}
		// send video url
		msg = tgbotapi.NewMessage(falling_user.ChatID, video_url)
		_, err := bot.Send(msg)
		if err != nil {
			log.Println("Error sending video")
		}
	}

}

func getVideoFromServer(username string) (string, error) {
	postBody, _ := json.Marshal(map[string]string{
		"username": username,
	})
	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post(server_url+"/get_latest_video", "application/json", responseBody)
	if err != nil {
		log.Println("Error sending latest video request")
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error getting latest video request")
		return "", err
	}
	return string(body), nil
}

func get_menu(username string) (string, error) {
	message := "Following features are: \n"
	message += "/menu -> Prints this menu.\n"
	message += "/getAllContacts -> Prints contacts.\n"
	message += "/addcontact -> Add contact to notification list.\n"

	message += "If your elder will fall, we will send you a message via this channel and provide a video.\n"
	message += "Thanks. ELS team.\n"
	return message, nil
}
