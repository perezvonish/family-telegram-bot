package bot

// Module — интерфейс каждого функционального модуля бота.
// Каждый модуль регистрирует свои команды и callback-префиксы в роутере.
type Module interface {
	// Name — уникальный идентификатор модуля ("diary", "pills", "analytics")
	Name() string

	// Commands — список команд без слэша: ["diary", "start"]
	Commands() []string

	// CallbackPrefixes — префиксы callback_data: ["pills:", "diary:"]
	CallbackPrefixes() []string

	// HandleCommand вызывается при получении текстовой команды (/pills, /diary)
	HandleCommand(ctx BotContext, cmd string, args string) error

	// HandleCallback вызывается при нажатии inline-кнопки
	HandleCallback(ctx BotContext, msgID int, data string) error

	// HandleTextStep вызывается при получении текста в рамках активной сессии модуля
	HandleTextStep(ctx BotContext, session *Session, text string) error
}
