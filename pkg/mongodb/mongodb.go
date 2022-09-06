package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// New mongodb connection with context given
func NewClientWithContext(ctx context.Context, url string) (*mongo.Client, error) {
	mCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	opts := options.Client()
	opts.ApplyURI(url)

	client, err := mongo.NewClient(opts)
	if err != nil {
		return nil, fmt.Errorf("could not create new mongodb client: %v", err)
	}

	if err := client.Connect(mCtx); err != nil {
		return nil, fmt.Errorf("could not connect to mongodb client: %v", err)
	}

	if err := client.Ping(mCtx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("could not ping mongodb client: %v", err)
	}

	return client, nil
}

// New mongodb connection
func NewClient(url string) (*mongo.Client, error) {
	return NewClientWithContext(context.Background(), url)
}
