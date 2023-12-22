package models

import (
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

const (
	REDIS_ERROR_NOT_FOUND     = "redis: nil"
	DB_ERROR_NOT_FOUND        = "record not found"
	SCANNY_DB_ERROR_NOT_FOUND = "scany: no row was found"
	TOKEN_ERROR_AUTH          = "token not provided"
)

func IsErrNotFound(err error) bool {
	return err != nil &&
		(err.Error() == SCANNY_DB_ERROR_NOT_FOUND ||
			err.Error() == REDIS_ERROR_NOT_FOUND ||
			err.Error() == DB_ERROR_NOT_FOUND)
}

func AllowErrNotFound(err error) error {
	if IsErrNotFound(err) {
		return nil
	}
	return err
}

type DbClient interface {
	PingClient(ctx context.Context) error
	//Migrate(ctx context.Context, isViewMigrate bool) error
	Create(ctx context.Context, input interface{}) error
	Get(ctx context.Context, params FilterParams, out interface{}) error
	GetView(ctx context.Context, viewName string, params FilterParams, out interface{}) error
	Select(ctx context.Context, table string, params FilterParams, out interface{}) error
	Update(ctx context.Context, params FilterParams, input interface{}) error
	Upsert(ctx context.Context, params FilterParams, input interface{}) error
	Delete(ctx context.Context, params FilterParams, input interface{}) error
	CloseClient() error
}

type CacheClient interface {
	SetHash(ctx context.Context, key string, objectType interface{}, expTime time.Duration) error
	GetHash(ctx context.Context, key string, out interface{}) error
	DeleteHash(ctx context.Context, key string) error
	GetKeys(ctx context.Context, pattern string, out *[]string) error
	GetList(ctx context.Context, list string, out interface{}) error
	PushToList(ctx context.Context, key string, objectType interface{}) error
	SubScribe(ctx context.Context, topics ...string) error
	ReceiveMsg(ctx context.Context, out *redis.Message) error
	PublishMsg(ctx context.Context, topic string, msg interface{}) error
	CloseSub(ctx context.Context) error
	CloseClient() error
}

type FilterParams struct {
	Filter string
	Select string
	FeedParams
}

const (
	FEED_SIZE_DEFAULT   = 1
	FEED_SIZE_MAX       = 48
	FEED_SIZE_UNLIMITED = -1
)

type FeedParams struct {
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
	Orderings string `form:"orderings"`
}

func (this FeedParams) ValidLimit() int {
	if this.Limit == 0 {
		return FEED_SIZE_UNLIMITED
	}

	if this.Limit > FEED_SIZE_MAX {
		return FEED_SIZE_MAX
	}

	return this.Limit
}
