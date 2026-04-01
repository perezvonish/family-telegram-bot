package repository

import (
	"perezvonish/health-tracker/internal/domain/daily_report"
	"time"

	"github.com/google/uuid"
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
	MigraineSide string   `bson:"migraine_side,omitempty"`
	MigraineDose string   `bson:"migraine_dose,omitempty"`
	Libido       int      `bson:"libido"`

	Extras        []string `bson:"extras,omitempty"`
	Anxiety       int      `bson:"anxiety"`
	Energy        int      `bson:"energy"`
	SleepQuality  int      `bson:"sleep_quality"`
	MoodStability string   `bson:"mood_stability,omitempty"`
	Relationship  int      `bson:"relationship"`
	Closeness     int      `bson:"closeness"`
	DayComment    string   `bson:"day_comment,omitempty"`
}

func toDailyReportEntity(m *DailyReportModel) *daily_report.DailyReport {
	userID, _ := uuid.Parse(m.UserID)
	return &daily_report.DailyReport{
		UserID:        userID,
		CreatedAt:     m.CreatedAt,
		ReportDate:    m.ReportDate,
		SleepTime:     m.SleepTime,
		WakeTime:      m.WakeTime,
		WorkedToday:   m.WorkedToday,
		Menstruation:  m.Menstruation,
		Fasting:       m.Fasting,
		Activity:      m.Activity,
		MealsSkipped:  m.MealsSkipped,
		MedsIssues:    m.MedsIssues,
		Mood:          m.Mood,
		Migraine:      m.Migraine,
		MigraineSide:  m.MigraineSide,
		MigraineDose:  m.MigraineDose,
		Libido:        m.Libido,
		Extras:        m.Extras,
		Anxiety:       m.Anxiety,
		Energy:        m.Energy,
		SleepQuality:  m.SleepQuality,
		MoodStability: m.MoodStability,
		Relationship:  m.Relationship,
		Closeness:     m.Closeness,
		DayComment:    m.DayComment,
	}
}

func ToDailyReportModel(entity *daily_report.DailyReport) *DailyReportModel {
	return &DailyReportModel{
		UserID:        entity.UserID.String(),
		CreatedAt:     entity.CreatedAt,
		ReportDate:    entity.ReportDate,
		SleepTime:     entity.SleepTime,
		WakeTime:      entity.WakeTime,
		WorkedToday:   entity.WorkedToday,
		Menstruation:  entity.Menstruation,
		Fasting:       entity.Fasting,
		Activity:      entity.Activity,
		MealsSkipped:  entity.MealsSkipped,
		MedsIssues:    entity.MedsIssues,
		Mood:          entity.Mood,
		Migraine:      entity.Migraine,
		MigraineSide:  entity.MigraineSide,
		MigraineDose:  entity.MigraineDose,
		Libido:        entity.Libido,
		Extras:        entity.Extras,
		Anxiety:       entity.Anxiety,
		Energy:        entity.Energy,
		SleepQuality:  entity.SleepQuality,
		MoodStability: entity.MoodStability,
		Relationship:  entity.Relationship,
		Closeness:     entity.Closeness,
		DayComment:    entity.DayComment,
	}
}
