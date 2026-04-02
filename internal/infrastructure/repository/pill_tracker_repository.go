package repository

import (
	"context"

	"perezvonish/health-tracker/internal/domain/pill_tracker"
	"perezvonish/health-tracker/internal/infrastructure/database"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const pillTrackersCollection = "pill_trackers"

type PillTrackerRepository struct {
	collection *mongo.Collection
}

func NewPillTrackerRepository(db *database.MongoDB) *PillTrackerRepository {
	return &PillTrackerRepository{
		collection: db.Collection(pillTrackersCollection),
	}
}

func (r *PillTrackerRepository) Create(ctx context.Context, tracker *pill_tracker.PillTracker) error {
	model := toPillTrackerModel(tracker)
	_, err := r.collection.InsertOne(ctx, model)
	return err
}

func (r *PillTrackerRepository) Update(ctx context.Context, tracker *pill_tracker.PillTracker) error {
	model := toPillTrackerModel(tracker)
	_, err := r.collection.ReplaceOne(ctx, bson.M{"uuid": tracker.ID.String()}, model)
	return err
}

func (r *PillTrackerRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]*pill_tracker.PillTracker, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID.String()})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var models []PillTrackerModel
	if err = cursor.All(ctx, &models); err != nil {
		return nil, err
	}

	result := make([]*pill_tracker.PillTracker, 0, len(models))
	for i := range models {
		result = append(result, toPillTrackerEntity(&models[i]))
	}
	return result, nil
}

func (r *PillTrackerRepository) FindByID(ctx context.Context, id uuid.UUID) (*pill_tracker.PillTracker, error) {
	var m PillTrackerModel
	err := r.collection.FindOne(ctx, bson.M{"uuid": id.String()}).Decode(&m)
	if err != nil {
		return nil, err
	}
	return toPillTrackerEntity(&m), nil
}

func (r *PillTrackerRepository) FindAllActive(ctx context.Context) ([]*pill_tracker.PillTracker, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var models []PillTrackerModel
	if err = cursor.All(ctx, &models); err != nil {
		return nil, err
	}

	result := make([]*pill_tracker.PillTracker, 0, len(models))
	for i := range models {
		e := toPillTrackerEntity(&models[i])
		if !e.IsEmpty() {
			result = append(result, e)
		}
	}
	return result, nil
}
