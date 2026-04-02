package telegram_bot

import (
	"fmt"
	"slices"

	"perezvonish/health-tracker/internal/domain/pill_tracker"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func yesNoKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("да"),
			tgbotapi.NewKeyboardButton("нет"),
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
	)
}

func fastingKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("нет"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("около часа"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("2–3 часа"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("больше 3 часов"),
		),
	)
}

func activityKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Не было"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Мало (дорога/быт)"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Средне (5к+ шагов/спорт)"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Сверх нормы"),
		),
	)
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
		tgbotapi.NewInlineKeyboardButtonData("Готово", "m:done"),
	))

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func scaleRangeKeyboard(prefix string, min, max int, isPositive bool) tgbotapi.InlineKeyboardMarkup {
	count := max - min + 1
	base := []string{"😣", "😕", "😐", "🙂", "😊", "😌", "💪", "🔥", "🚀", "🤯", "🌟"}

	emojis := make([]string, len(base))
	if isPositive {
		copy(emojis, base)
	} else {
		for i := range base {
			emojis[i] = base[len(base)-1-i]
		}
	}

	var buttons []tgbotapi.InlineKeyboardButton
	for i := 0; i < count; i++ {
		value := min + i
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%d %s", value, emojis[i]),
			fmt.Sprintf("%s:%d", prefix, value),
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

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func scaleKeyboard(prefix string, isPositive bool) tgbotapi.InlineKeyboardMarkup {
	return scaleRangeKeyboard(prefix, 0, 10, isPositive)
}

func labeledScaleKeyboard(prefix string, labels []string) tgbotapi.InlineKeyboardMarkup {
	var buttons []tgbotapi.InlineKeyboardButton
	for i, label := range labels {
		value := i + 1
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%d %s", value, label),
			fmt.Sprintf("%s:%d", prefix, value),
		))
	}
	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{buttons},
	}
}

func migraineSideKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Двусторонняя", "mside:bilateral"),
			tgbotapi.NewInlineKeyboardButtonData("Сильнее слева", "mside:left"),
			tgbotapi.NewInlineKeyboardButtonData("Сильнее справа", "mside:right"),
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
	)
}

func skipKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Пропустить"),
		),
	)
}

func removeKeyboard() tgbotapi.ReplyKeyboardRemove {
	return tgbotapi.NewRemoveKeyboard(true)
}

func pillsCountKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("30"),
			tgbotapi.NewKeyboardButton("60"),
			tgbotapi.NewKeyboardButton("90"),
			tgbotapi.NewKeyboardButton("100"),
		),
	)
}

func pillsDoseKeyboard(currentDose string) tgbotapi.ReplyKeyboardMarkup {
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

func pillsListKeyboard(trackers []*pill_tracker.PillTracker) tgbotapi.InlineKeyboardMarkup {
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

func pillsEmptyKeyboard(trackerID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, купила", "pills:restock:"+trackerID),
			tgbotapi.NewInlineKeyboardButtonData("⏰ Напомни через 2 дня", "pills:snooze:"+trackerID),
		),
	)
}
