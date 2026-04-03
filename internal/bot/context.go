package bot

import (
	"context"
	"perezvonish/health-tracker/internal/domain/user"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// BotContext передаётся в каждый хэндлер модуля вместо прямых зависимостей ChatBot.
// Модульные зависимости (репозитории) модули инжектят через замыкание при создании.
type BotContext struct {
	Ctx      context.Context
	ChatID   int64
	UserID   int64
	API      *tgbotapi.BotAPI
	Sessions *SessionStore
	Users    user.Repository
}

// Send отправляет текстовое сообщение без клавиатуры.
func (b BotContext) Send(text string) {
	msg := tgbotapi.NewMessage(b.ChatID, text)
	b.API.Send(msg) //nolint:errcheck
}

// SendWithKeyboard отправляет сообщение с reply-клавиатурой.
func (b BotContext) SendWithKeyboard(text string, keyboard tgbotapi.ReplyKeyboardMarkup) {
	msg := tgbotapi.NewMessage(b.ChatID, text)
	msg.ReplyMarkup = keyboard
	b.API.Send(msg) //nolint:errcheck
}

// SendWithInlineKeyboard отправляет сообщение с inline-клавиатурой.
func (b BotContext) SendWithInlineKeyboard(text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(b.ChatID, text)
	msg.ReplyMarkup = keyboard
	b.API.Send(msg) //nolint:errcheck
}

// SendRemovingKeyboard отправляет сообщение и убирает reply-клавиатуру.
func (b BotContext) SendRemovingKeyboard(text string) {
	msg := tgbotapi.NewMessage(b.ChatID, text)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	b.API.Send(msg) //nolint:errcheck
}

// EditMessageRemoveKeyboard убирает inline-клавиатуру у существующего сообщения.
func (b BotContext) EditMessageRemoveKeyboard(msgID int) {
	edit := tgbotapi.NewEditMessageReplyMarkup(
		b.ChatID, msgID,
		tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}},
	)
	b.API.Send(edit) //nolint:errcheck
}
