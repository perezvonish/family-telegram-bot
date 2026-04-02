package telegram_bot

import "sync"

type SceneType string

const (
	SceneDiary SceneType = "diary"
	ScenePills SceneType = "pills"
)

type PillSetupAnswers struct {
	EditingID string
	Name      string
	Total     int
	DailyDose float64
}

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
	MigraineSide string   `json:"migraineSide,omitempty"`
	MigraineDose string   `json:"migraineDose,omitempty"`
	Libido       int      `json:"libido,omitempty"`

	Extras        []string `json:"extras,omitempty"`
	Anxiety       int      `json:"anxiety,omitempty"`
	Energy        int      `json:"energy,omitempty"`
	SleepQuality  int      `json:"sleepQuality,omitempty"`
	MoodStability string   `json:"moodStability,omitempty"`
	Relationship  int      `json:"relationship,omitempty"`
	Closeness     int      `json:"closeness,omitempty"`
	DayComment    string   `json:"dayComment,omitempty"`
}

type Session struct {
	Scene      SceneType
	Step       int
	Answers    HealthAnswers
	PillsSetup PillSetupAnswers
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
		Scene: SceneDiary,
		Step:  -1,
		Answers: HealthAnswers{
			MealsSkipped: []string{},
			MedsIssues:   []string{},
			Extras:       []string{},
		},
	}
	s.sessions[chatID] = session
	return session
}

func (s *SessionStore) Reset(chatID int64) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := &Session{
		Scene: SceneDiary,
		Step:  0,
		Answers: HealthAnswers{
			MealsSkipped: []string{},
			MedsIssues:   []string{},
			Extras:       []string{},
		},
	}
	s.sessions[chatID] = session
	return session
}

func (s *SessionStore) ResetPills(chatID int64) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := &Session{
		Scene: ScenePills,
		Step:  0,
	}
	s.sessions[chatID] = session
	return session
}

func (s *SessionStore) Delete(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, chatID)
}
