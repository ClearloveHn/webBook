package service

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"webBook/internal/domain"
	"webBook/internal/repository"
)

var (
	ErrDuplicateEmail        = repository.ErrDuplicateUser // 导出错误，表示邮箱已存在。
	ErrInvalidUserOrPassword = errors.New("用户不存在或者密码不对")   // 自定义错误，表示登录失败。
)

type UserService interface {
	Signup(ctx context.Context, u domain.User) error
	Login(ctx context.Context, email string, password string) (domain.User, error)
	UpdateNonSensitiveInfo(ctx context.Context,
		user domain.User) error
	FindById(ctx context.Context,
		uid int64) (domain.User, error)
	FindOrCreate(ctx context.Context, phone string) (domain.User, error)
}

type userService struct {
	repo repository.UserRepository // repo字段，指向UserRepository结构体实例，用于仓库层操作。
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{
		repo: repo,
	}
}

// Signup 方法，用户注册。
func (svc *userService) Signup(ctx context.Context, u domain.User) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost) // 对密码进行加密。
	if err != nil {
		return err // 如果加密过程中出现错误，直接返回错误。
	}
	u.Password = string(hash) // 将加密后的密码回写。
	return svc.repo.Create(ctx, u)
}

// Login 方法，用户登录。
func (svc *userService) Login(ctx context.Context, email string, password string) (domain.User, error) {
	u, err := svc.repo.FindByEmail(ctx, email)      // 从仓库层通过邮箱查找用户。
	if errors.Is(err, repository.ErrUserNotFound) { // 用户未找到。
		return domain.User{}, ErrInvalidUserOrPassword
	}
	if err != nil {
		return domain.User{}, err // 如果有其他错误，直接返回错误。
	}
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) // 比较密码。
	if err != nil {
		return domain.User{}, ErrInvalidUserOrPassword // 密码不匹配。
	}
	return u, nil // 登录成功，返回用户信息。
}

// UpdateNonSensitiveInfo 方法，更新用户非敏感信息。
func (svc *userService) UpdateNonSensitiveInfo(ctx context.Context, user domain.User) error {
	return svc.repo.UpdateNonZeroFields(ctx, user) // 在仓库层更新用户非敏感信息。
}

// FindById 方法，通过ID查找用户。
func (svc *userService) FindById(ctx context.Context, uid int64) (domain.User, error) {
	return svc.repo.FindById(ctx, uid) // 在仓库层通过ID查找用户。
}

// FindOrCreate 方法，通过电话查找用户，如果不存在则创建新用户。
func (svc *userService) FindOrCreate(ctx context.Context, phone string) (domain.User, error) {
	u, err := svc.repo.FindByPhone(ctx, phone) // 首先尝试从仓库层通过电话查找用户。
	if !errors.Is(err, repository.ErrUserNotFound) {
		return u, err // 如果用户存在或者有其他错误，直接返回结果。
	}
	err = svc.repo.Create(ctx, domain.User{Phone: phone}) // 用户未找到，创建新用户。
	if err != nil && !errors.Is(err, repository.ErrDuplicateUser) {
		return domain.User{}, err // 如果创建过程中出现非重复错误，返回错误。
	}
	return svc.repo.FindByPhone(ctx, phone) // 再次尝试从仓库层通过电话查找用户。
}
