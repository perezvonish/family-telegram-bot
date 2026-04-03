package bot

import (
	"log"
	"strings"
)

type prefixEntry struct {
	prefix string
	module Module
}

// Router регистрирует модули и диспетчеризует входящие сообщения/callback-и.
type Router struct {
	modules      []Module
	commandIndex map[string]Module
	prefixIndex  []prefixEntry
	sessions     *SessionStore
	flags        FeatureFlagStore
}

func NewRouter(sessions *SessionStore, flags FeatureFlagStore) *Router {
	return &Router{
		commandIndex: make(map[string]Module),
		sessions:     sessions,
		flags:        flags,
	}
}

// Sessions возвращает хранилище сессий роутера (для внешнего доступа при переходном периоде).
func (r *Router) Sessions() *SessionStore { return r.sessions }

// GetSession возвращает активную сессию для chatID, если она есть.
func (r *Router) GetSession(chatID int64) *Session { return r.sessions.Get(chatID) }

// Register добавляет модуль в роутер.
func (r *Router) Register(m Module) {
	r.modules = append(r.modules, m)
	for _, cmd := range m.Commands() {
		r.commandIndex[cmd] = m
	}
	for _, pfx := range m.CallbackPrefixes() {
		r.prefixIndex = append(r.prefixIndex, prefixEntry{pfx, m})
	}
}

// HandleMessage обрабатывает входящее текстовое сообщение.
func (r *Router) HandleMessage(ctx BotContext, text string) {
	cmd, args := parseCommand(text)

	// 1. Есть активная сессия — передаём шаг в модуль
	if session := r.sessions.Get(ctx.ChatID); session != nil {
		m, ok := r.commandIndex[session.Module]
		if ok && r.flags.IsEnabled(m.Name(), ctx.UserID) {
			if err := m.HandleTextStep(ctx, session, text); err != nil {
				log.Printf("[router] HandleTextStep error (module=%s): %v", session.Module, err)
			}
			return
		}
	}

	// 2. Это команда — ищем модуль
	if cmd != "" {
		m, ok := r.commandIndex[cmd]
		if !ok {
			ctx.Send("Неизвестная команда. Используй /help")
			return
		}
		if !r.flags.IsEnabled(m.Name(), ctx.UserID) {
			ctx.Send("Эта функция недоступна.")
			return
		}
		if err := m.HandleCommand(ctx, cmd, args); err != nil {
			log.Printf("[router] HandleCommand error (module=%s, cmd=%s): %v", m.Name(), cmd, err)
		}
		return
	}

	ctx.Send("Используй /help чтобы увидеть доступные команды")
}

// HandleCallback обрабатывает нажатие inline-кнопки.
func (r *Router) HandleCallback(ctx BotContext, msgID int, data string) {
	for _, entry := range r.prefixIndex {
		if strings.HasPrefix(data, entry.prefix) {
			if r.flags.IsEnabled(entry.module.Name(), ctx.UserID) {
				if err := entry.module.HandleCallback(ctx, msgID, data); err != nil {
					log.Printf("[router] HandleCallback error (module=%s, data=%s): %v", entry.module.Name(), data, err)
				}
			}
			return
		}
	}
	log.Printf("[router] no module found for callback data: %s", data)
}

// parseCommand разбирает текст на команду (без слэша) и аргументы.
// "/stats 30" → ("stats", "30"), "hello" → ("", "")
func parseCommand(text string) (cmd, args string) {
	if !strings.HasPrefix(text, "/") {
		return "", ""
	}
	trimmed := strings.TrimPrefix(text, "/")
	// убираем @botname если есть
	if idx := strings.Index(trimmed, "@"); idx != -1 {
		trimmed = trimmed[:idx]
	}
	parts := strings.SplitN(trimmed, " ", 2)
	cmd = strings.ToLower(parts[0])
	if len(parts) == 2 {
		args = strings.TrimSpace(parts[1])
	}
	return cmd, args
}
