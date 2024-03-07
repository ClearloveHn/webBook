package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
	"webBook/internal/domain"
)

// ErrKeyNotExist 定义ErrKeyNotExist为redis.Nil，表示当Redis中不存在该键时返回的错误
var ErrKeyNotExist = redis.Nil

type UserCache interface {
	Get(ctx context.Context, uid int64) (domain.User, error)
	Set(ctx context.Context, du domain.User) error
}

type RedisUserCache struct {
	cmd        redis.Cmdable
	expiration time.Duration // expiration字段，设置缓存过期时间。
}

func NewUserCache(cmd redis.Cmdable) UserCache {
	return &RedisUserCache{
		cmd:        cmd,
		expiration: time.Minute * 15,
	}
}

// Get 方法，从缓存中获取用户信息。
func (c *RedisUserCache) Get(ctx context.Context, uid int64) (domain.User, error) {
	key := c.key(uid)
	data, err := c.cmd.Get(ctx, key).Result() // 从Redis获取数据。
	if err != nil {
		return domain.User{}, err // 如果出现错误，返回空的User结构体和错误信息。
	}
	var u domain.User
	err = json.Unmarshal([]byte(data), &u)
	return u, err
}

// Set 方法，将用户信息设置到缓存中。
func (c *RedisUserCache) Set(ctx context.Context, du domain.User) error {
	key := c.key(du.Id)
	data, err := json.Marshal(du)
	if err != nil {
		return err // 如果序列化失败，返回错误。
	}
	return c.cmd.Set(ctx, key, data, c.expiration).Err() // 将数据写入Redis，并设置过期时间，返回可能的错误。
}

// key 方法，生成Redis键名。
func (c *RedisUserCache) key(uid int64) string {
	return fmt.Sprintf("user:info:%d", uid) // 格式化生成特定格式的键名。
}
