/*
 * IstkharMusic - Telegram Music Bot
 *  https://github.com/TEAM-ISTKHAR/IstkharMusic
 *
 *  Based on TgMusicBot, Copyright (c) 2025-2026 Ashok Shau
 *  Licensed under GNU GPL v3
 */

package db

import (
	"IstkharMusic/internal/cache"
	"IstkharMusic/internal/config"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Database encapsulates the MongoDB connection, database, collections, and caches.
type Database struct {
	client      *mongo.Client
	DB          *mongo.Database
	chatDB      *mongo.Collection
	userDB      *mongo.Collection
	playlistDB  *mongo.Collection
	assistantDB *mongo.Collection
	authDB      *mongo.Collection
	langDB      *mongo.Collection
	cacheDB     *mongo.Collection

	chatCache      *cache.Cache[*Chats]
	userCache      *cache.Cache[*Users]
	assistantCache *cache.Cache[int]
	authCache      *cache.Cache[[]int64]
	langCache      *cache.Cache[string]
	loggerCache    *cache.Cache[bool]
	blChatsCache   *cache.Cache[[]int64]
	blUsersCache   *cache.Cache[[]int64]
}

// Instance is the global singleton for the database.
var Instance *Database

// InitDatabase initializes the database connection and sets up the global instance.
func InitDatabase() error {
	uri := strings.TrimSpace(config.MongoUri)
	if uri == "" {
		return errors.New("MONGO_URI is empty")
	}

	dbName := strings.TrimSpace(config.DbName)
	if dbName == "" {
		dbName = "Anon"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := options.Client().ApplyURI(uri).
		SetMinPoolSize(2).
		SetMaxConnecting(10).
		SetMaxConnIdleTime(10 * time.Minute).
		SetConnectTimeout(25 * time.Second).
		SetServerSelectionTimeout(25 * time.Second)

	client, err := mongo.Connect(opts)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	db := client.Database(dbName)
	Instance = &Database{
		client:      client,
		DB:          db,
		chatDB:      db.Collection("chats"),
		userDB:      db.Collection("users"),
		playlistDB:  db.Collection("playlists"),
		assistantDB: db.Collection("assistant"),
		authDB:      db.Collection("auth"),
		langDB:      db.Collection("lang"),
		cacheDB:     db.Collection("cache"),

		chatCache:      cache.NewCache[*Chats](20 * time.Minute),
		userCache:      cache.NewCache[*Users](20 * time.Minute),
		assistantCache: cache.NewCache[int](20 * time.Minute),
		authCache:      cache.NewCache[[]int64](20 * time.Minute),
		langCache:      cache.NewCache[string](20 * time.Minute),
		loggerCache:    cache.NewCache[bool](20 * time.Minute),
		blChatsCache:   cache.NewCache[[]int64](20 * time.Minute),
		blUsersCache:   cache.NewCache[[]int64](20 * time.Minute),
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	Instance.EnsureEventIndex()
	slog.Info("[DB] The database connection has been successfully established.", "db", dbName)
	return nil
}

func toKey(id int64) string {
	return strconv.FormatInt(id, 10)
}

// Close gracefully closes the database connection.
func (db *Database) Close() error {
	if db == nil || db.client == nil {
		return nil
	}
	ctx, cancel := db.ctx()
	defer cancel()

	slog.Info("[DB] Closing the database connection...")
	return db.client.Disconnect(ctx)
}

func (db *Database) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
