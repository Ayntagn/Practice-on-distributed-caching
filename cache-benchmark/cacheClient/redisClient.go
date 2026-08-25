package cacheClient

import (
	"context"

	"caches-core/ports"
	"github.com/redis/go-redis/v9"
)

type redisClient struct {
	*redis.Client
}

func (r *redisClient) get(key string) (string, error) {
	res, e := r.Get(context.Background(), key).Result()
	if e == redis.Nil {
		return "", nil
	}
	return res, e
}

func (r *redisClient) set(key, value string) error {
	return r.Set(context.Background(), key, value, 0).Err()
}

func (r *redisClient) del(key string) error {
	return r.Del(context.Background(), key).Err()
}

func (r *redisClient) Run(c *Cmd) {
	if c.Name == "get" {
		c.Value, c.Error = r.get(c.Key)
		return
	}
	if c.Name == "set" {
		c.Error = r.set(c.Key, c.Value)
		return
	}
	if c.Name == "del" {
		c.Error = r.del(c.Key)
		return
	}
	panic("unknown cmd name " + c.Name)
}

func (r *redisClient) PipelinedRun(cmds []*Cmd) {
	if len(cmds) == 0 {
		return
	}
	ctx := context.Background()
	cmders := make([]redis.Cmder, len(cmds))
	_, e := r.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for i, c := range cmds {
			switch c.Name {
			case "get":
				cmders[i] = pipe.Get(ctx, c.Key)
			case "set":
				cmders[i] = pipe.Set(ctx, c.Key, c.Value, 0)
			case "del":
				cmders[i] = pipe.Del(ctx, c.Key)
			default:
				panic("unknown cmd name " + c.Name)
			}
		}
		return nil
	})
	if e != nil && e != redis.Nil {
		panic(e)
	}
	for i, c := range cmds {
		if c.Name == "get" {
			value, e := cmders[i].(*redis.StringCmd).Result()
			if e == redis.Nil {
				value, e = "", nil
			}
			c.Value, c.Error = value, e
		} else {
			c.Error = cmders[i].Err()
		}
	}
}

func newRedisClient(server string) *redisClient {
	return &redisClient{redis.NewClient(&redis.Options{Addr: server + ":" + ports.Redis, ReadTimeout: -1})}
}
