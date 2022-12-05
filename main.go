package main

import (
	"log"
	"net/http"
	"net/url"

	"github.com/NicoNex/echotron/v3"
)

type stateFn func(*echotron.Update) stateFn

type bot struct {
	chatID int64
	state  stateFn
	name   string
	echotron.API
}

const token = ""

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
	kb := [][]echotron.KeyboardButton{
		{
			{Text: "Мои профили"},
		},
		{
			{Text: "Получить профиль"},
		},
		{
			{Text: "Удалить профиль"},
		},
	}
	b.SendMessage("", b.chatID, &echotron.MessageOptions{
		ReplyMarkup: echotron.ReplyKeyboardMarkup{
			Keyboard:       kb,
			ResizeKeyboard: true,
		},
	})

	if update.Message.Text == "Получить профиль" {
		b.SendMessage("Название профиля?", b.chatID, nil)

		return b.handleType
	}

	return b.handleMessage
}
func (b *bot) handleType(update *echotron.Update) stateFn {
	data := url.Values{
		"name":        {update.Message.From.Username},
		"config_type": {update.Message.Text},
	}

	_, err := http.PostForm("http://127.0.0.1/create-config", data)

	if err != nil {
		log.Fatal(err)
	}

	b.SendMessage("Профиль создан!", b.chatID, nil)

	return b.handleMessage
}

func main() {
	dsp := echotron.NewDispatcher(token, newBot)
	log.Println(dsp.Poll())
}
