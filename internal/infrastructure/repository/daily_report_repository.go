package repository

import (
	"context"
	"perezvonish/health-tracker/internal/domain/daily_report"
	"perezvonish/health-tracker/internal/infrastructure/database"

	"go.mongodb.org/mongo-driver/mongo"
)

const dailyReportsCollection = "daily_reports"

type DailyReportRepository struct {
	collection *mongo.Collection
}

func NewDailyReportRepository(db *database.MongoDB) *DailyReportRepository {
	return &DailyReportRepository{
		collection: db.Collection(dailyReportsCollection),
	}
}

func (r *DailyReportRepository) Create(ctx context.Context, report *daily_report.DailyReport) error {
	model := ToDailyReportModel(report)

	_, err := r.collection.InsertOne(ctx, model)
	if err != nil {
		return err
	}

	return nil
}
