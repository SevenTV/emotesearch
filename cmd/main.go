package main

import (
	"context"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"

	"github.com/seventv/emotesearch/config"
)

var (
	cfg   *config.Config
	db    *mongo.Database
	index *meilisearch.Index
)

func main() {
	cfg = config.New()

	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   cfg.Meilisearch.Host,
		APIKey: cfg.Meilisearch.Key,
	})

	index = client.Index(cfg.Meilisearch.Index)

	err := connectMongo()
	if err != nil {
		panic(err)
	}

	// sync before starting ticker, so we don't have to wait for the first tick
	err = sync()
	if err != nil {
		panic(err)
	}

	for range time.Tick(cfg.SyncInterval) {
		err := sync()
		if err != nil {
			panic(err)
		}
	}
}

func connectMongo() error {
	opt := options.Client().ApplyURI(cfg.Mongo.URI)
	opt.SetAuth(options.Credential{
		Username: cfg.Mongo.Username,
		Password: cfg.Mongo.Password,
	})

	mongoClient, err := mongo.Connect(context.Background(), opt)
	if err != nil {
		return err
	}

	err = mongoClient.Ping(context.Background(), readpref.Primary())
	if err != nil {
		return err
	}

	db = mongoClient.Database(cfg.Mongo.Database)
	return nil
}

func sync() error {
	cur, err := db.Collection(cfg.Mongo.Collection).Find(context.Background(), bson.D{})
	if err != nil {
		return err
	}

	for cur.Next(context.Background()) {
		var emote struct {
			ID       primitive.ObjectID `bson:"_id"`
			Name     string             `bson:"name"`
			Tags     []string           `bson:"tags"`
			Versions []struct {
				State struct {
					ChannelCount int64 `bson:"channel_count"`
				} `bson:"state"`
				CreatedAt time.Time `bson:"created_at"`
			} `bson:"versions"`
			Flags string `bson:"flags"`
		}
		err := cur.Decode(&emote)
		if err != nil {
			zap.S().Errorw("failed to decode emote", "error", err)
			continue
		}

		channelCount := 0
		createdAt := time.Now()

		for _, version := range emote.Versions {
			channelCount += int(version.State.ChannelCount)

			if createdAt.Sub(version.CreatedAt) > 0 {
				createdAt = version.CreatedAt
			}
		}

		doc := []map[string]interface{}{
			{
				"id":            emote.ID.Hex(),
				"name":          emote.Name,
				"tags":          emote.Tags,
				"channel_count": channelCount,
				"created_at":    createdAt,
				"flags":         channelCount,
			},
		}

		_, err = index.UpdateDocuments(doc)
		if err != nil {
			return err
		}
	}

	return nil
}
