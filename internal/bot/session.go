package bot

import "sync"

// Session — модуль-независимая сессия пользователя.
// Данные модуля хранятся в Data["ключ"] — каждый модуль сам управляет своими ключами.
type Session struct {
	Module string // "diary" | "pills" | ...
	Step   int
	Data   map[string]any
}

func NewSession(module string) *Session {
	return &Session{
		Module: module,
		Step:   0,
		Data:   make(map[string]any),
	}
}

func (s *Session) Get(key string) (any, bool) {
	v, ok := s.Data[key]
	return v, ok
}

func (s *Session) Set(key string, value any) {
	s.Data[key] = value
}

func (s *Session) GetString(key string) string {
	v, ok := s.Data[key]
	if !ok {
		return ""
	}
	str, _ := v.(string)
	return str
}

func (s *Session) GetInt(key string) int {
	v, ok := s.Data[key]
	if !ok {
		return 0
	}
	n, _ := v.(int)
	return n
}

func (s *Session) GetStringSlice(key string) []string {
	v, ok := s.Data[key]
	if !ok {
		return nil
	}
	sl, _ := v.([]string)
	return sl
}

// SessionStore — потокобезопасное хранилище сессий.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[int64]*Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[int64]*Session),
	}
}

func (s *SessionStore) Get(chatID int64) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[chatID]
}

func (s *SessionStore) Set(chatID int64, session *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[chatID] = session
}

func (s *SessionStore) Delete(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, chatID)
}
