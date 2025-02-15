package main

import (
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	lastReplyTimeMap    map[int64]time.Time
	lastReminderTimeMap map[int64]time.Time
	lastUpdateTime      time.Time
	updateInterval      = 2 * time.Hour
	checkInterval       = 5 * time.Minute
	reminderInterval    = 24 * time.Hour
	reminderMessage     = "Ну чо, посоны, вы как? Живы?"
	reminderChatID      int64 = -1002039497735
)

func shouldSendReply(chatID int64) bool {
	currentTime := time.Now()
	diff := currentTime.Sub(lastReplyTimeMap[chatID])
	return diff.Minutes() >= 25
}

func shouldSendReminder() bool {
	currentTime := time.Now()
	if currentTime.Hour() >= 10 && currentTime.Hour() <= 20 {
		diff := currentTime.Sub(lastUpdateTime)
		if diff >= updateInterval {
			lastCheckTime, ok := lastReminderTimeMap[reminderChatID]
			if !ok || currentTime.Sub(lastCheckTime) >= reminderInterval {
				return true
			}
		}
	}
	return false
}

func sendReminder(bot *tgbotapi.BotAPI) {
	reply := tgbotapi.NewMessage(reminderChatID, reminderMessage)
	_, err := bot.Send(reply)
	if err != nil {
		log.Println(err)
	}
	lastReminderTimeMap[reminderChatID] = time.Now()
}

func main() {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Println(err)
	}
	time.Local = loc

	lastReplyTimeMap = make(map[int64]time.Time)
	lastReminderTimeMap = make(map[int64]time.Time)
	lastReminderTimeMap[reminderChatID] = time.Now()

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_APITOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"channel_post", "message"}

	updates := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	// Updates loop
	go func() {
		for update := range updates {
			lastUpdateTime = time.Now()
			if update.ChannelPost != nil {
				channelMsg := update.ChannelPost
				if bot.Debug {
					log.Printf("Channel: [%s] %s", channelMsg.Chat.UserName, channelMsg.Text)
				}
			} else if update.Message != nil {
				groupMsg := update.Message
				if bot.Debug {
					log.Printf("Group: [%s] %s", groupMsg.Chat.UserName, groupMsg.Text)
				}

				chatID := update.Message.Chat.ID

				if shouldSendReply(chatID) {
					text := strings.ToLower(update.Message.Text)

					// Используйте регулярное выражение для поиска "да" и "нет" в пределах границ предложения
					yesRegex := regexp.MustCompile(`\bда\b`)
					noRegex := regexp.MustCompile(`\bнет\b`)

					if yesRegex.MatchString(text) {
						reply := tgbotapi.NewMessage(chatID, "Пизда")
						_, err := bot.Send(reply)
						if err != nil {
							log.Println(err)
						}
						lastReplyTimeMap[chatID] = time.Now()
					} else if noRegex.MatchString(text) {
						reply := tgbotapi.NewMessage(chatID, "Пидора ответ")
						_, err := bot.Send(reply)
						if err != nil {
							log.Println(err)
						}
						lastReplyTimeMap[chatID] = time.Now()
					} else if text == "/get_id" {
						chatIDStr := strconv.FormatInt(chatID, 10)
						reply := tgbotapi.NewMessage(chatID, chatIDStr)
						_, err := bot.Send(reply)
						if err != nil {
							log.Println(err)
						}
					}
				}
			}
		}
	}()

	// Reminder loop
	go func() {
		for {
			if shouldSendReminder() {
				sendReminder(bot)
			}
			time.Sleep(checkInterval)
		}
	}()

	// Keep main goroutine alive
	select {}
}
