package pills

import (
	"fmt"

	"perezvonish/health-tracker/internal/domain/pill_tracker"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func countKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("30"),
			tgbotapi.NewKeyboardButton("60"),
			tgbotapi.NewKeyboardButton("90"),
			tgbotapi.NewKeyboardButton("100"),
		),
	)
}

func doseKeyboard(currentDose string) tgbotapi.ReplyKeyboardMarkup {
	rows := [][]tgbotapi.KeyboardButton{
		{
			tgbotapi.NewKeyboardButton("0.5"),
			tgbotapi.NewKeyboardButton("1"),
			tgbotapi.NewKeyboardButton("2"),
			tgbotapi.NewKeyboardButton("3"),
		},
	}
	if currentDose != "" {
		rows = append(rows, []tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton("Оставить " + currentDose + " в день"),
		})
	}
	return tgbotapi.NewReplyKeyboard(rows...)
}

func listKeyboard(trackers []*pill_tracker.PillTracker) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, t := range trackers {
		label := fmt.Sprintf("✏️ %s (осталось %.0f, до %s)",
			t.Name, t.Remaining(), t.EmptyDate().Format("2 Jan"))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "pills:edit:"+t.ID.String()),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("＋ Добавить препарат", "pills:add"),
	))
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func emptyKeyboard(trackerID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, купила", "pills:restock:"+trackerID),
			tgbotapi.NewInlineKeyboardButtonData("⏰ Напомни через 2 дня", "pills:snooze:"+trackerID),
		),
	)
}
