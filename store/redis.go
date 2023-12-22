package store

import (
	"chatgpt/config"
	"chatgpt/models"
	"context"
	"encoding/json"
	"errors"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisClientReal struct {
	Client         *redis.Client
	PubSub         *redis.PubSub
	IsEnablePubSub bool
}

func NewRedisConn(config *config.Config) *RedisClientReal {
	return &RedisClientReal{
		Client: redis.NewClient(&redis.Options{
			Addr:     config.CacheHost,
			Password: config.CachePass,
			DB:       0,
		}),
		IsEnablePubSub: true,
	}
}

func NewCacheClient(ctx context.Context, config *config.Config, out *models.CacheClient) (err error) {
	cacheClient := NewRedisConn(config)

	_, err = cacheClient.Client.Ping(ctx).Result()
	if err != nil {
		return err
	}

	if out != nil && cacheClient != nil {
		*out = cacheClient
	}
	return nil
}

func (this RedisClientReal) SetHash(ctx context.Context, key string, objectType interface{}, expTime time.Duration) error {
	value, err := json.Marshal(objectType)
	if err != nil {
		return err
	}
	return this.Client.Set(ctx, key, value, expTime).Err()
}

func (this RedisClientReal) GetHash(ctx context.Context, key string, out interface{}) error {
	result, err := this.Client.Get(ctx, key).Bytes()

	if err != nil {
		return err
	}
	return json.Unmarshal(result, &out)
}

func (this RedisClientReal) GetKeys(ctx context.Context, pattern string, out *[]string) error {
	keys, err := this.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}

	*out = append(*out, keys...)
	return nil
}

const (
	ListStart = 0
	ListEnd   = -1
)

func UnMarshalStruct(in interface{}, out interface{}) error {
	bytes, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, out)
}

func (this RedisClientReal) GetList(ctx context.Context, list string, out interface{}) error {
	elements, err := this.Client.LRange(ctx, list, ListStart, ListEnd).Result()
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}

	var results []interface{}
	for _, elem := range elements {
		var result interface{}
		err = json.Unmarshal([]byte(elem), &result)
		if err != nil {
			return err
		}

		results = append(results, result)
	}

	return UnMarshalStruct(results, out)
}

func (this RedisClientReal) PushToList(ctx context.Context, key string, objectType interface{}) error {
	value, err := json.Marshal(objectType)
	if err != nil {
		return err
	}
	return this.Client.RPush(ctx, key, value).Err()
}

func (this RedisClientReal) DeleteHash(ctx context.Context, key string) error {
	return this.Client.Del(ctx, key).Err()
}

func (this RedisClientReal) PublishMsg(ctx context.Context, topic string, msg interface{}) error {
	if !this.IsEnablePubSub {
		return nil
	}
	return this.Client.Publish(ctx, topic, msg).Err()
}

func (this *RedisClientReal) SubScribe(ctx context.Context, topics ...string) error {
	if !this.IsEnablePubSub {
		return nil
	}
	this.PubSub = this.Client.Subscribe(ctx, topics...)
	_, err := this.PubSub.Receive(ctx)
	return err
}

func (this *RedisClientReal) ReceiveMsg(ctx context.Context, out *redis.Message) error {
	if this.PubSub == nil {
		return errors.New("no subscriber is provided")
	}
	msg, err := this.PubSub.ReceiveMessage(ctx)
	if err != nil {
		return err
	}
	if out != nil && msg != nil {
		*out = *msg
	}
	return nil
}

func (this RedisClientReal) CloseSub(ctx context.Context) error {
	if !this.IsEnablePubSub {
		return nil
	}
	return this.PubSub.Unsubscribe(ctx)
}

func (this RedisClientReal) CloseClient() error {
	return this.Client.Close()
}
