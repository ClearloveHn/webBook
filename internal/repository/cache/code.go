package cache

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
)

var (
	//go:embed lua/set_code.lua
	luaSetCode string // 嵌入一个Lua脚本文件，这个脚本用于设置验证码。
	//go:embed lua/verify_code.lua
	luaVerifyCode string // 嵌入另一个Lua脚本文件，这个脚本用于验证验证码。

	ErrCodeSendTooMany   = errors.New("发送太频繁") // 定义一个错误，表示验证码发送太频繁。
	ErrCodeVerifyTooMany = errors.New("发送太频繁") // 定义一个错误，表示验证码验证请求太频繁。
)

// CodeCache 结构体定义，持有redis命令接口。
type CodeCache struct {
	cmd redis.Cmdable // cmd是一个redis命令接口，用于执行redis操作。
}

func NewCodeCache(cmd redis.Cmdable) *CodeCache {
	return &CodeCache{
		cmd: cmd,
	}
}

// Set 方法用于设置验证码。
func (c *CodeCache) Set(ctx context.Context, biz, phone, code string) error {
	// 使用Lua脚本和提供的参数设置验证码。
	res, err := c.cmd.Eval(ctx, luaSetCode, []string{c.key(biz, phone)}, code).Int()
	if err != nil {
		// 如果执行Redis命令出错，则返回错误。
		return err
	}
	switch res {
	case -2:
		return errors.New("验证码存在，但是没有过期时间") // Lua脚本返回的特定错误。
	case -1:
		return ErrCodeSendTooMany // 发送频繁的错误。
	default:
		return nil // 无错误，设置成功。
	}
}

// Verify 方法用于验证验证码。
func (c *CodeCache) Verify(ctx context.Context, biz, phone, code string) (bool, error) {
	// 使用Lua脚本和提供的参数验证验证码。
	res, err := c.cmd.Eval(ctx, luaVerifyCode, []string{c.key(biz, phone)}, code).Int()
	if err != nil {
		// 如果执行Redis命令出错，则返回错误。
		return false, err
	}
	switch res {
	case -2:
		return false, nil // 验证码不正确。
	case -1:
		return false, ErrCodeVerifyTooMany // 验证请求频繁。
	default:
		return true, nil // 验证码正确。
	}
}

// key 方法用于生成存储在Redis中的键。
func (c *CodeCache) key(biz, phone string) string {
	// 格式化并返回特定的键格式。
	return fmt.Sprintf("phone_code:%s:%s", biz, phone)
}
