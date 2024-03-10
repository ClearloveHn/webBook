package repository

import (
	"context"
	"webBook/internal/repository/cache"
)

var ErrCodeVerifyTooMany = cache.ErrCodeVerifyTooMany // 导出错误，表明验证码验证请求太频繁。
var ErrCodeSendTooMany = cache.ErrCodeSendTooMany

type CodeRepository interface {
	Set(ctx context.Context, biz, phone, code string) error
	Verify(ctx context.Context, biz, phone, code string) (bool, error)
}

type CachedCodeRepository struct {
	cache cache.CodeCache // cache字段，类型为*cache.CodeCache，指定验证码缓存操作。
}

func NewCodeRepository(c cache.CodeCache) CodeRepository {
	return &CachedCodeRepository{
		cache: c,
	}
}

// Set 方法，将验证码存储到缓存中。
func (c *CachedCodeRepository) Set(ctx context.Context, biz, phone, code string) error {
	return c.cache.Set(ctx, biz, phone, code) // 调用缓存层的Set方法来存储验证码。
}

// Verify 方法，验证缓存中的验证码。
func (c *CachedCodeRepository) Verify(ctx context.Context, biz, phone, code string) (bool, error) {
	return c.cache.Verify(ctx, biz, phone, code) // 调用缓存层的Verify方法来验证验证码。
}
