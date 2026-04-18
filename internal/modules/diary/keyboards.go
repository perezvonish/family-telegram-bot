package diary

import (
	"fmt"
	"slices"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func yesNoKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("да"),
			tgbotapi.NewKeyboardButton("нет"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("← Назад"),
		),
	)
}

func sleepTimeKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("раньше 22:00"),
			tgbotapi.NewKeyboardButton("22:00"),
			tgbotapi.NewKeyboardButton("23:00"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("00:00"),
			tgbotapi.NewKeyboardButton("01:00"),
			tgbotapi.NewKeyboardButton("02:00"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("позже 2:00"),
		),
	)
}

func wakeTimeKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("раньше 9:00"),
			tgbotapi.NewKeyboardButton("9:00"),
			tgbotapi.NewKeyboardButton("10:00"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("11:00"),
			tgbotapi.NewKeyboardButton("12:00"),
			tgbotapi.NewKeyboardButton("позже 12:00"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("← Назад"),
		),
	)
}

func fastingKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("нет")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("около часа")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("2–3 часа")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("больше 3 часов")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("← Назад")),
	)
}

func activityKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Не было")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Мало (дорога/быт)")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Средне (5к+ шагов/спорт)")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Сверх нормы")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("← Назад")),
	)
}

func skipKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Пропустить")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("← Назад")),
	)
}

func removeKeyboard() tgbotapi.ReplyKeyboardRemove {
	return tgbotapi.NewRemoveKeyboard(true)
}

func multiSelectKeyboard(options []string, selected []string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, opt := range options {
		checked := slices.Contains(selected, opt)
		icon := "✅"
		if checked {
			icon = "❌"
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %s", icon, opt), "m:"+opt),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", "back:"),
		tgbotapi.NewInlineKeyboardButtonData("Готово", "m:done"),
	))
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func scaleRangeKeyboard(prefix string, min, max int) tgbotapi.InlineKeyboardMarkup {
	var buttons []tgbotapi.InlineKeyboardButton
	for i := min; i <= max; i++ {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%d", i),
			fmt.Sprintf("%s:%d", prefix, i),
		))
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(buttons); i += 6 {
		end := i + 6
		if end > len(buttons) {
			end = len(buttons)
		}
		rows = append(rows, buttons[i:end])
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", "back:"),
	))
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func labeledScaleKeyboard(prefix string, labels []string) tgbotapi.InlineKeyboardMarkup {
	var buttons []tgbotapi.InlineKeyboardButton
	for i := range labels {
		value := i + 1
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%d", value),
			fmt.Sprintf("%s:%d", prefix, value),
		))
	}
	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			buttons,
			{tgbotapi.NewInlineKeyboardButtonData("← Назад", "back:")},
		},
	}
}

func migraineSideKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Двусторонняя", "mside:bilateral"),
			tgbotapi.NewInlineKeyboardButtonData("Сильнее слева", "mside:left"),
			tgbotapi.NewInlineKeyboardButtonData("Сильнее справа", "mside:right"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Назад", "back:"),
		),
	)
}

func stabilityKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Ровное", "stability:ровное"),
			tgbotapi.NewInlineKeyboardButtonData("Были качели", "stability:качели"),
			tgbotapi.NewInlineKeyboardButtonData("Резкие перепады", "stability:перепады"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Назад", "back:"),
		),
	)
}
