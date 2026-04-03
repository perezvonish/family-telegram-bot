package repository

import (
	"context"
	"errors"
	"time"

	"perezvonish/health-tracker/internal/domain/daily_report"
	"perezvonish/health-tracker/internal/infrastructure/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (r *DailyReportRepository) FindByPeriod(ctx context.Context, userID string, from, to time.Time) ([]*daily_report.DailyReport, error) {
	if userID == "" {
		return nil, errors.New("userID is empty")
	}
	filter := bson.M{
		"user_id":     userID,
		"report_date": bson.M{"$gte": from, "$lte": to},
	}
	opts := options.Find().SetSort(bson.D{{Key: "report_date", Value: 1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var models []DailyReportModel
	if err = cursor.All(ctx, &models); err != nil {
		return nil, err
	}

	result := make([]*daily_report.DailyReport, 0, len(models))
	for _, m := range models {
		result = append(result, toDailyReportEntity(&m))
	}
	return result, nil
}

func (r *DailyReportRepository) FindLatest(ctx context.Context, userID string, limit int) ([]*daily_report.DailyReport, error) {
	if userID == "" {
		return nil, errors.New("userID is empty")
	}
	filter := bson.M{"user_id": userID}
	opts := options.Find().
		SetSort(bson.D{{Key: "report_date", Value: -1}}).
		SetLimit(int64(limit))
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var models []DailyReportModel
	if err = cursor.All(ctx, &models); err != nil {
		return nil, err
	}

	result := make([]*daily_report.DailyReport, 0, len(models))
	for _, m := range models {
		result = append(result, toDailyReportEntity(&m))
	}
	return result, nil
}

func (r *DailyReportRepository) FindByDate(ctx context.Context, userID string, date time.Time) (*daily_report.DailyReport, error) {
	if userID == "" {
		return nil, errors.New("userID is empty")
	}
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	filter := bson.M{
		"user_id":     userID,
		"report_date": bson.M{"$gte": dayStart, "$lt": dayEnd},
	}

	var m DailyReportModel
	err := r.collection.FindOne(ctx, filter).Decode(&m)
	if err != nil {
		return nil, err
	}

	return toDailyReportEntity(&m), nil
}
