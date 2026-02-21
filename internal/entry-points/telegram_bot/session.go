package telegram_bot

import "sync"

type HealthAnswers struct {
	SleepTime    string   `json:"sleepTime,omitempty"`
	WakeTime     string   `json:"wakeTime,omitempty"`
	WorkedToday  string   `json:"workedToday,omitempty"`
	Menstruation string   `json:"menstruation,omitempty"`
	Fasting      string   `json:"fasting,omitempty"`
	Activity     string   `json:"activity,omitempty"`
	MealsSkipped []string `json:"mealsSkipped,omitempty"`
	MedsIssues   []string `json:"medsIssues,omitempty"`
	Mood         int      `json:"mood,omitempty"`
	Migraine     int      `json:"migraine,omitempty"`
	MigraineDose float64  `json:"migraineDose,omitempty"`
	Libido       int      `json:"libido,omitempty"`
}

type Session struct {
	Step    int
	Answers HealthAnswers
}

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

func (s *SessionStore) GetOrCreate(chatID int64) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, ok := s.sessions[chatID]; ok {
		return session
	}

	session := &Session{
		Step: -1,
		Answers: HealthAnswers{
			MealsSkipped: []string{},
			MedsIssues:   []string{},
		},
	}
	s.sessions[chatID] = session
	return session
}

func (s *SessionStore) Reset(chatID int64) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := &Session{
		Step: 0,
		Answers: HealthAnswers{
			MealsSkipped: []string{},
			MedsIssues:   []string{},
		},
	}
	s.sessions[chatID] = session
	return session
}

func (s *SessionStore) Delete(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, chatID)
}
