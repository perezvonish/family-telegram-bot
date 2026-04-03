package daily_report

import (
	"time"
)

type DailyReport struct {
	UserID     string    `json:"userId"`
	CreatedAt  time.Time `json:"createdAt"`
	ReportDate time.Time `json:"reportDate"`

	SleepTime    string   `json:"sleepTime"`
	WakeTime     string   `json:"wakeTime"`
	WorkedToday  string   `json:"workedToday"`
	Menstruation string   `json:"menstruation"`
	Fasting      string   `json:"fasting"`
	Activity     string   `json:"activity"`
	MealsSkipped []string `json:"mealsSkipped"`
	MedsIssues   []string `json:"medsIssues"`
	Mood         int      `json:"mood"`
	Migraine     int      `json:"migraine"`
	MigraineSide string   `json:"migraineSide,omitempty"`
	MigraineDose string   `json:"migraineDose,omitempty"`
	Libido       int      `json:"libido"`

	Extras        []string `json:"extras,omitempty"`
	Anxiety       int      `json:"anxiety"`
	Energy        int      `json:"energy"`
	SleepQuality  int      `json:"sleepQuality"`
	MoodStability string   `json:"moodStability,omitempty"`
	Relationship  int      `json:"relationship"`
	Closeness     int      `json:"closeness"`
	DayComment    string   `json:"dayComment,omitempty"`
}

func NewDailyReport(userID string) *DailyReport {
	now := time.Now().UTC()
	reportDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	return &DailyReport{
		UserID:       userID,
		CreatedAt:    now,
		ReportDate:   reportDate,
		MealsSkipped: []string{},
		MedsIssues:   []string{},
		Extras:       []string{},
	}
}
