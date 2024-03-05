package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"webBook/internal/repository"
	"webBook/internal/service/sms"
)

var ErrCodeSendTooMany = repository.ErrCodeVerifyTooMany // 导出错误，表示验证码发送过于频繁。

type CodeService struct {
	repo *repository.CodeRepository // repo字段，指向CodeRepository结构体实例，用于仓库层操作。
	sms  sms.Service                // sms字段，指向sms.Service结构体实例，用于短信服务层操作。
}

func NewCodeService(repo *repository.CodeRepository, smsSvc sms.Service) *CodeService {
	return &CodeService{
		repo: repo,
		sms:  smsSvc,
	}
}

// Send 方法，发送短信验证码。
func (svc *CodeService) Send(ctx context.Context, biz, phone string) error {
	code := svc.generate()                     // 生成验证码。
	err := svc.repo.Set(ctx, biz, phone, code) // 将验证码存储到redis中。
	if err != nil {
		return err // 如果存储过程中出现错误，直接返回错误。
	}
	const codeTplId = "1877556"                                // 短信模板ID，通常从配置中获取。
	return svc.sms.Send(ctx, codeTplId, []string{code}, phone) // 发送短信。
}

// Verify 方法，验证输入的验证码是否正确。
func (svc *CodeService) Verify(ctx context.Context, biz, phone, inputCode string) (bool, error) {
	ok, err := svc.repo.Verify(ctx, biz, phone, inputCode) // 在仓库层验证验证码。
	if errors.Is(err, repository.ErrCodeVerifyTooMany) {
		// 如果错误是因为验证次数过多，对外隐藏具体错误细节，只返回不成功的验证。
		return false, nil
	}
	return ok, err // 返回验证结果和可能的错误。
}

// generate 方法，生成六位随机验证码。
func (svc *CodeService) generate() string {
	code := rand.Intn(1000000)       // 生成一个0到999999之间的随机数。
	return fmt.Sprintf("%06d", code) // 格式化为六位字符串，不足前面补零。
}
