package services

import (
	"dds-backend/common"
	"dds-backend/database"
	"dds-backend/models"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
	"time"
)

const (
	RegistrationTokenExpirationDuration = time.Minute * 10
	BotAlias                            = "dds_project_f19_bot"
)

// Generate bot registration link for user
func GetChatRegistrationLink(username string) (string, error) {
	chat := models.TelegramChat{
		Username: username,
	}
	res := database.DB.Model(&models.TelegramChat{}).Where(&chat).First(&chat)
	if res.RecordNotFound() {
		// need to be registered
		chat.RegistrationToken = common.GenerateNewToken()
		chat.TokenExpiration = time.Now().UTC().Add(RegistrationTokenExpirationDuration)
		res = database.DB.Model(&models.TelegramChat{}).Create(&chat)
		if res.Error != nil {
			return "", res.Error
		}
	} else if res.Error != nil {
		return "", res.Error
	} else {
		chat.RegistrationToken = common.GenerateNewToken()
		chat.TokenExpiration = time.Now().UTC().Add(RegistrationTokenExpirationDuration)
		res = database.DB.Model(&models.TelegramChat{}).Save(&chat)
		if res.Error != nil {
			return "", res.Error
		}
	}
	return fmt.Sprintf("https://t.me/%s?start=%s", BotAlias, chat.RegistrationToken), nil
}

// Find username of user that corresponds to given chat id
func GetUsernameByChat(chatID int64) (string, error) {
	chat := models.TelegramChat{ChatID: chatID}
	res := database.DB.Model(&models.TelegramChat{}).Where(&chat).First(&chat)
	if res.Error != nil {
		return "", errors.New("could not get this chat")
	}
	return chat.Username, nil
}

// Find chat id of user that corresponds to given username
func GetChatIDByUsername(username string) (int64, error) {
	chat := models.TelegramChat{Username: username}
	res := database.DB.Model(&models.TelegramChat{}).Where(&chat).First(&chat)
	if res.Error != nil {
		return 0, errors.New("could not get this chat")
	}
	return chat.ChatID, nil
}

// Requested when user logs in via Telegram
func ValidateChat(registrationToken string, chatID int64) error {
	chat := models.TelegramChat{
		RegistrationToken: registrationToken,
	}
	res := database.DB.Model(&models.TelegramChat{}).Where(&chat).First(&chat)
	if res.RecordNotFound() {
		// registration token does not exist
		return errors.New("can't validate this token")
	} else if res.Error != nil {
		// unexpected error
		return errors.New("something went wrong")
	} else {
		// ok
		if chat.TokenExpiration.Before(time.Now().UTC()) {
			return errors.New("token has expired")
		}
		chat.ChatID = chatID
		chat.RegistrationToken = common.GenerateNewToken(chat.Username) // to erase previous token
		res = database.DB.Model(&models.TelegramChat{}).Save(&chat)
		if res.Error != nil {
			return errors.New("registration failed")
		}
	}
	return nil
}

var commandKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("/schedule"),
	),
)

var BotInstance *tgbotapi.BotAPI

func SendNotification(username string, text string) error {
	chatID, err := GetChatIDByUsername(username)
	if err != nil {
		return err
	}
	if BotInstance != nil {
		msg := tgbotapi.NewMessage(chatID, text)
		_, err := BotInstance.Send(msg)
		if err != nil {
			log.Panic(err.Error())
		}
	} else {
		return errors.New("bot instance is nil")
	}
	return nil
}

func LaunchBot() {
	var err error
	BotInstance, err = tgbotapi.NewBotAPI(os.Getenv("DDS_TELEGRAM_BOT_APIKEY"))
	if err != nil {
		log.Panic(err)
	}
	BotInstance.Debug = false
	log.Printf("Authorized on account %s", BotInstance.Self.UserName)

	var updateConf tgbotapi.UpdateConfig = tgbotapi.NewUpdate(0)
	updateConf.Timeout = 60
	updatesChan, err := BotInstance.GetUpdatesChan(updateConf)

	// TODO handle /schedule command
	// TODO send notifications
	for {
		select {
		case update := <-updatesChan:
			var msg tgbotapi.MessageConfig
			if update.Message == nil {
				log.Println("Message is nil")
				return
			}
			// handle login of new user
			if update.Message.IsCommand() && update.Message.Command() == "start" {
				var cmd string
				var key string
				readNum, _ := fmt.Sscanf(update.Message.Text, "%s %s", &cmd, &key)
				msg.ChatID = update.Message.Chat.ID
				if readNum != 2 {
					msg.Text = "Sorry, you don't have access to this bot."
				} else {
					// validate key, register chatid for this user
					msg.Text = fmt.Sprintf("You are registering with key: %s", key)
					err := ValidateChat(key, msg.ChatID)
					if err != nil {
						msg.Text = err.Error()
					} else {
						msg.Text = "Welcome to DDS Schedule Bot!\n You are successfully registered. See /help for available commands."
					}
				}
				// handle commands of existing user
			} else if update.Message.IsCommand() {
				// check if not registered and refuse further communication in that case
				username, err := GetUsernameByChat(update.Message.Chat.ID)
				if err != nil {
					msg.Text = ""
				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "")
					switch update.Message.Command() {
					case "help":
						msg.Text = "type /schedule to know your schedule"
					case "schedule":
						tp1, tp2, wks, err := GetSchedule(username)
						if err != nil {
							if _, ok := err.(*ScheduleNotFoundError); ok {
								msg.Text = "You don't have schedule right now."
							} else {
								msg.Text = "Something went wrong, contact your manager."
							}
						} else {
							msg.Text = fmt.Sprintf(PrettySchedule(tp1, tp2, wks))
						}
					default:
						msg.Text = "I don't know that command"
					}
				}
			} else {
				// TODO fill this `not understood` branch
			}

			msg.ReplyMarkup = commandKeyboard
			_, err := BotInstance.Send(msg)
			if err != nil {
				log.Println(err.Error())
			}
		}
	}
}
