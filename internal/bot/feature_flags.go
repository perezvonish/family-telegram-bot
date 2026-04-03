package bot

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// FeatureFlagStore позволяет проверить, включён ли модуль для конкретного пользователя.
type FeatureFlagStore interface {
	IsEnabled(feature string, userID int64) bool
	Reload(ctx context.Context) error
}

// --- EnvFeatureFlags — реализация через переменные окружения ---
// Формат: FEATURE_PILLS=true, FEATURE_ANALYTICS=false

type EnvFeatureFlags struct {
	flags map[string]bool
}

func NewEnvFeatureFlags() *EnvFeatureFlags {
	flags := map[string]bool{}
	for _, pair := range os.Environ() {
		if !strings.HasPrefix(pair, "FEATURE_") {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.ToLower(strings.TrimPrefix(parts[0], "FEATURE_"))
		flags[name] = parts[1] == "true" || parts[1] == "1"
	}
	return &EnvFeatureFlags{flags: flags}
}

func (e *EnvFeatureFlags) IsEnabled(feature string, _ int64) bool {
	enabled, ok := e.flags[feature]
	if !ok {
		return true // неизвестная фича = включена (безопасный дефолт для env-режима)
	}
	return enabled
}

func (e *EnvFeatureFlags) Reload(_ context.Context) error {
	return nil // env-переменные статичны
}

// --- MongoFeatureFlags — реализация через коллекцию MongoDB ---
// Коллекция: feature_flags
// { "name": "pills", "enabled": true, "user_ids": [] }           // для всех
// { "name": "analytics", "enabled": true, "user_ids": [123] }    // только для userID=123

type featureFlagDoc struct {
	Name    string  `bson:"name"`
	Enabled bool    `bson:"enabled"`
	UserIDs []int64 `bson:"user_ids"`
}

type MongoFeatureFlags struct {
	collection *mongo.Collection
	cache      map[string]featureFlagDoc
	mu         sync.RWMutex
}

func NewMongoFeatureFlags(ctx context.Context, collection *mongo.Collection) (*MongoFeatureFlags, error) {
	m := &MongoFeatureFlags{
		collection: collection,
		cache:      make(map[string]featureFlagDoc),
	}
	if err := m.Reload(ctx); err != nil {
		return nil, err
	}
	go m.startAutoReload(ctx, 5*time.Minute)
	return m, nil
}

func (m *MongoFeatureFlags) IsEnabled(feature string, userID int64) bool {
	m.mu.RLock()
	flag, ok := m.cache[feature]
	m.mu.RUnlock()

	if !ok {
		return true // неизвестная фича = включена
	}
	if !flag.Enabled {
		return false
	}
	if len(flag.UserIDs) == 0 {
		return true // включена для всех
	}
	for _, id := range flag.UserIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func (m *MongoFeatureFlags) Reload(ctx context.Context) error {
	cursor, err := m.collection.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	cache := make(map[string]featureFlagDoc)
	for cursor.Next(ctx) {
		var doc featureFlagDoc
		if err := cursor.Decode(&doc); err != nil {
			log.Printf("[feature_flags] decode error: %v", err)
			continue
		}
		cache[doc.Name] = doc
	}

	m.mu.Lock()
	m.cache = cache
	m.mu.Unlock()

	log.Printf("[feature_flags] reloaded %d flags from MongoDB", len(cache))
	return cursor.Err()
}

func (m *MongoFeatureFlags) startAutoReload(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.Reload(ctx); err != nil {
				log.Printf("[feature_flags] auto-reload error: %v", err)
			}
		}
	}
}
