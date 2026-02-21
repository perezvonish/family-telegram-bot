package repository

import (
	"perezvonish/health-tracker/internal/domain/daily_report"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DailyReportModel struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	UserID     string             `bson:"user_id"`
	CreatedAt  time.Time          `bson:"created_at"`
	ReportDate time.Time          `bson:"report_date"`

	SleepTime    string   `bson:"sleep_time"`
	WakeTime     string   `bson:"wake_time"`
	WorkedToday  string   `bson:"worked_today"`
	Menstruation string   `bson:"menstruation"`
	Fasting      string   `bson:"fasting"`
	Activity     string   `bson:"activity"`
	MealsSkipped []string `bson:"meals_skipped"`
	MedsIssues   []string `bson:"meds_issues"`
	Mood         int      `bson:"mood"`
	Migraine     int      `bson:"migraine"`
	MigraineDose string   `bson:"migraine_dose,omitempty"`
	Libido       int      `bson:"libido"`
}

func ToDailyReportModel(entity *daily_report.DailyReport) *DailyReportModel {
	return &DailyReportModel{
		UserID:       entity.UserID.String(),
		CreatedAt:    entity.CreatedAt,
		ReportDate:   entity.ReportDate,
		SleepTime:    entity.SleepTime,
		WakeTime:     entity.WakeTime,
		WorkedToday:  entity.WorkedToday,
		Menstruation: entity.Menstruation,
		Fasting:      entity.Fasting,
		Activity:     entity.Activity,
		MealsSkipped: entity.MealsSkipped,
		MedsIssues:   entity.MedsIssues,
		Mood:         entity.Mood,
		Migraine:     entity.Migraine,
		MigraineDose: entity.MigraineDose,
		Libido:       entity.Libido,
	}
}
