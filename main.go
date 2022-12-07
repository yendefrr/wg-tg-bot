package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/NicoNex/echotron/v3"
	_ "github.com/go-sql-driver/mysql"
)

type stateFn func(*echotron.Update) stateFn

type bot struct {
	chatID int64
	state  stateFn
	name   string
	echotron.API
}

type Profile struct {
	ID                   uint8
	Type, Config, QRCode string
}

const token = "5891720043:AAEzUjPVqCCr3lahdOEHnUZ0k_EHN81o5i0"

var (
	mainKeyboard = [][]echotron.KeyboardButton{
		{
			{Text: "Мои профили"},
		},
		{
			{Text: "Получить профиль"},
		},
	}

	typeKeyboard = [][]echotron.KeyboardButton{
		{
			{Text: "Отмена"},
		},
	}
)

func newBot(chatID int64) echotron.Bot {
	bot := &bot{
		chatID: chatID,
		API:    echotron.NewAPI(token),
	}

	bot.state = bot.handleMessage
	return bot
}

func (b *bot) Update(update *echotron.Update) {
	b.state = b.state(update)
}

func (b *bot) handleMessage(update *echotron.Update) stateFn {
	switch text := extractText(update); {
	case text == "/start":
		return b.StartMsg(update)
	case text == "/menu":
		return b.Menu(update)
	case text == "Получить профиль":
		return b.RequestProfile(update)
	case text == "Мои профили":
		return b.GetProfiles(update)
	}

	if update.CallbackQuery != nil {
		b.handleCallback(update)
	}

	return b.handleMessage
}

func (b *bot) StartMsg(update *echotron.Update) stateFn {
	b.SendMessage("[Скачать](https://www.wireguard.com/install/) WireGuard", b.chatID, &echotron.MessageOptions{
		ParseMode: echotron.Markdown,
		ReplyMarkup: echotron.ReplyKeyboardMarkup{
			Keyboard:       mainKeyboard,
			ResizeKeyboard: true,
		},
	})

	return b.handleMessage
}

func (b *bot) Menu(update *echotron.Update) stateFn {
	b.SendMessage("Что хочешь?", b.chatID, &echotron.MessageOptions{
		ReplyMarkup: echotron.ReplyKeyboardMarkup{
			Keyboard:       mainKeyboard,
			ResizeKeyboard: true,
		},
	})

	return b.handleMessage
}

func (b *bot) RequestProfile(update *echotron.Update) stateFn {
	b.SendMessage("Название профиля?", b.chatID, &echotron.MessageOptions{
		ReplyMarkup: echotron.ReplyKeyboardMarkup{
			Keyboard:       typeKeyboard,
			ResizeKeyboard: true,
		},
	})

	return b.handleType
}

func (b *bot) GetProfiles(update *echotron.Update) stateFn {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/wgadmin")
	if err != nil {
		log.Println(err)
	}
	defer db.Close()

	res, err := db.Query(fmt.Sprintf("SELECT id, type, config, qrcode FROM profiles WHERE username = '%s'", update.Message.From.Username))
	if err != nil {
		log.Println(err)
	}

	i := 0
	for res.Next() {
		i++
		var profile Profile

		if err := res.Scan(&profile.ID, &profile.Type, &profile.Config, &profile.QRCode); err != nil {
			log.Println(err)
		}

		config, _ := base64.StdEncoding.DecodeString(profile.Config)
		qrcode, _ := base64.StdEncoding.DecodeString(profile.QRCode)

		b.SendPhoto(echotron.NewInputFileBytes(profile.Type+".png", qrcode), b.chatID, &echotron.PhotoOptions{
			Caption: profile.Type,
			ReplyMarkup: echotron.InlineKeyboardMarkup{
				InlineKeyboard: [][]echotron.InlineKeyboardButton{
					{
						{
							Text:         "Удалить",
							CallbackData: fmt.Sprintf("http://127.0.0.1/remove-config?id=%d", profile.ID),
						},
					},
				},
			},
		})

		b.SendDocument(echotron.NewInputFileBytes(profile.Type+".conf", config), b.chatID, &echotron.DocumentOptions{
			ReplyMarkup: echotron.ReplyKeyboardMarkup{
				Keyboard:       mainKeyboard,
				ResizeKeyboard: true,
			},
		})
	}

	if i == 0 {
		b.SendMessage("У тебя еще нет профилей!", b.chatID, &echotron.MessageOptions{
			ReplyMarkup: echotron.ReplyKeyboardMarkup{
				Keyboard:       mainKeyboard,
				ResizeKeyboard: true,
			},
		})
	}

	return b.handleMessage
}

func (b *bot) handleType(update *echotron.Update) stateFn {
	if update.Message.Text == "Отмена" || update.Message.Text == "/menu" || update.Message.Text == "/start" {
		b.SendMessage("Что хочешь?", b.chatID, &echotron.MessageOptions{
			ReplyMarkup: echotron.ReplyKeyboardMarkup{
				Keyboard:       mainKeyboard,
				ResizeKeyboard: true,
			},
		})
		return b.handleMessage
	}

	if len(update.Message.Text) < 3 {
		b.SendMessage("Название должно содержать не менее двух символов!", b.chatID, &echotron.MessageOptions{
			ReplyMarkup: echotron.ReplyKeyboardMarkup{
				Keyboard:       typeKeyboard,
				ResizeKeyboard: true,
			},
		})

		return b.handleType
	}

	data := url.Values{
		"name":        {update.Message.From.Username},
		"config_type": {update.Message.Text},
	}

	_, err := http.PostForm("http://127.0.0.1/create-config", data)

	if err != nil {
		log.Println(err)
	}

	b.SendMessage("Профиль создан!", b.chatID, &echotron.MessageOptions{
		ReplyMarkup: echotron.ReplyKeyboardMarkup{
			Keyboard:       mainKeyboard,
			ResizeKeyboard: true,
		},
	})

	return b.handleMessage
}

func (b *bot) handleCallback(update *echotron.Update) stateFn {
	url := update.CallbackQuery.Data
	b.DeleteMessage(b.chatID, update.CallbackQuery.Message.ID)
	b.DeleteMessage(b.chatID, update.CallbackQuery.Message.ID+1)

	_, err := http.Get(url)
	if err != nil {
		log.Println(err)
	}

	b.SendMessage("Профиль удален!", b.chatID, &echotron.MessageOptions{
		ReplyMarkup: echotron.ReplyKeyboardMarkup{
			Keyboard:       mainKeyboard,
			ResizeKeyboard: true,
		},
	})

	return b.handleMessage
}

func extractText(update *echotron.Update) string {
	if update.Message != nil {
		return update.Message.Text
	}
	return ""
}

func main() {
	dsp := echotron.NewDispatcher(token, newBot)
	log.Println(dsp.Poll())
}
