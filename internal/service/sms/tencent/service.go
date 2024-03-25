package tencent

import (
	"context"
	"fmt"
	"github.com/ecodeclub/ekit"
	"github.com/ecodeclub/ekit/slice"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
	"go.uber.org/zap"
)

// Service 结构体定义，包含腾讯云SMS客户端和相关配置。
type Service struct {
	client   *sms.Client // client字段，指向腾讯云SMS服务的客户端实例。
	appId    *string     // appId字段，存储腾讯云应用ID。
	signName *string     // signName字段，存储短信签名。
}

func NewService(client *sms.Client, appId string, signName string) *Service {
	return &Service{
		client:   client,
		appId:    &appId,
		signName: &signName,
	}
}

// Send 方法，发送短信。
func (s *Service) Send(ctx context.Context, tplId string, args []string, numbers ...string) error {
	request := sms.NewSendSmsRequest()             // 创建发送短信的请求实例。
	request.SetContext(ctx)                        // 设置请求上下文。
	request.SmsSdkAppId = s.appId                  // 设置短信SDK应用ID。
	request.SignName = s.signName                  // 设置短信签名。
	request.TemplateId = ekit.ToPtr[string](tplId) // 设置短信模板ID。
	request.TemplateParamSet = s.toPtrSlice(args)  // 设置短信模板参数。
	request.PhoneNumberSet = s.toPtrSlice(numbers) // 设置接收短信的手机号码。
	response, err := s.client.SendSms(request)     // 调用腾讯云SMS客户端发送短信。
	zap.L().Debug("请求腾讯SendSms接口",
		zap.Any("req", request),
		zap.Any("resp", response))
	// 处理异常。
	if err != nil {
		return err // 如果有错误，直接返回错误。
	}
	// 检查发送状态。
	for _, statusPtr := range response.Response.SendStatusSet {
		if statusPtr == nil {
			continue // 检查到空状态，跳过。
		}
		status := *statusPtr // 解引用状态指针。
		// 如果状态码不是"Ok"，表示短信发送失败。
		if status.Code == nil || *(status.Code) != "Ok" {
			return fmt.Errorf("发送短信失败 code: %s, msg: %s", *status.Code, *status.Message)
		}
	}
	return nil // 所有短信都成功发送。
}

// toPtrSlice 方法，将字符串切片转换为字符串指针切片。
func (s *Service) toPtrSlice(data []string) []*string {
	return slice.Map[string, *string](data,
		func(idx int, src string) *string {
			return &src // 转换为字符串指针。
		})
}
