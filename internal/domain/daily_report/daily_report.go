package daily_report

import (
	"time"

	"github.com/google/uuid"
)

type DailyReport struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"userId"`
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
	MigraineDose float64  `json:"migraineDose,omitempty"`
	Libido       int      `json:"libido"`
}

func NewDailyReport(userID uuid.UUID) *DailyReport {
	return &DailyReport{
		ID:           uuid.New(),
		UserID:       userID,
		CreatedAt:    time.Now(),
		ReportDate:   time.Now(),
		MealsSkipped: []string{},
		MedsIssues:   []string{},
	}
}
