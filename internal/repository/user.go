package repository

import (
	"context"
	"database/sql"
	"log"
	"time"
	"webBook/internal/domain"
	"webBook/internal/repository/cache"
	"webBook/internal/repository/dao"
)

var (
	ErrDuplicateUser = dao.ErrDuplicateEmail // 导出错误，表示用户信息冲突（如邮箱重复）。
	ErrUserNotFound  = dao.ErrRecordNotFound // 导出错误，表示未找到用户记录。
)

type UserRepository interface {
	Create(ctx context.Context, u domain.User) error
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	UpdateNonZeroFields(ctx context.Context, user domain.User) error
	FindByPhone(ctx context.Context, phone string) (domain.User, error)
	FindById(ctx context.Context, uid int64) (domain.User, error)
	FindByWechat(ctx context.Context, openId string) (domain.User, error)
}

type CachedUserRepository struct {
	dao   dao.UserDAO     // dao字段，指向UserDAO结构体实例，用于数据库操作。
	cache cache.UserCache // cache字段，指向UserCache结构体实例，用于缓存操作。
}

func NewCachedUserRepository(dao dao.UserDAO, c cache.UserCache) UserRepository {
	return &CachedUserRepository{
		dao:   dao,
		cache: c,
	}
}

// Create 方法，创建新用户。
func (repo *CachedUserRepository) Create(ctx context.Context, u domain.User) error {
	return repo.dao.Insert(ctx, repo.toEntity(u))
}

// FindByEmail 方法，通过邮箱查找用户。
func (repo *CachedUserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	u, err := repo.dao.FindByEmail(ctx, email)
	if err != nil {
		return domain.User{}, err
	}
	return repo.toDomain(u), nil
}

// toDomain 方法，将dao层的User转换为domain层的User。
func (repo *CachedUserRepository) toDomain(u dao.User) domain.User {
	return domain.User{
		Id:       u.Id,
		Email:    u.Email.String, // 注意处理sql.NullString。
		Phone:    u.Phone.String, // 注意处理sql.NullString。
		Password: u.Password,
		AboutMe:  u.AboutMe,
		Nickname: u.Nickname,
		Birthday: time.UnixMilli(u.Birthday), // 将Unix时间毫秒数转换为time.Time对象。
		Ctime:    time.UnixMilli(u.Ctime),
		WechatInfo: domain.WechatInfo{
			OpenId:  u.WechatOpenId.String,
			UnionId: u.WechatUnionId.String,
		},
	}
}

// toEntity 方法，将domain层的User转换为dao层的User。
func (repo *CachedUserRepository) toEntity(u domain.User) dao.User {
	return dao.User{
		Id: u.Id,
		Email: sql.NullString{
			String: u.Email,
			Valid:  u.Email != "",
		},
		Phone: sql.NullString{
			String: u.Phone,
			Valid:  u.Phone != "",
		},
		Password: u.Password,
		Birthday: u.Birthday.UnixMilli(), // 将time.Time对象转换为Unix时间毫秒数。
		AboutMe:  u.AboutMe,
		Nickname: u.Nickname,
		WechatUnionId: sql.NullString{
			String: u.WechatInfo.UnionId,
			Valid:  u.WechatInfo.UnionId != "",
		},
		WechatOpenId: sql.NullString{
			String: u.WechatInfo.OpenId,
			Valid:  u.WechatInfo.OpenId != "",
		},
	}
}

// UpdateNonZeroFields 方法，更新用户信息中的非零字段。
func (repo *CachedUserRepository) UpdateNonZeroFields(ctx context.Context, user domain.User) error {
	return repo.dao.UpdateById(ctx, repo.toEntity(user))
}

// FindById 方法，通过ID查找用户，首先尝试从缓存中获取，失败则从数据库获取。
func (repo *CachedUserRepository) FindById(ctx context.Context, uid int64) (domain.User, error) {
	du, err := repo.cache.Get(ctx, uid)
	if err == nil {
		return du, nil
	}

	u, err := repo.dao.FindById(ctx, uid)
	if err != nil {
		return domain.User{}, err
	}
	du = repo.toDomain(u)
	err = repo.cache.Set(ctx, du)
	if err != nil {
		log.Println(err)
	}
	return du, nil
}

// FindByPhone 方法，通过电话号码查找用户。
func (repo *CachedUserRepository) FindByPhone(ctx context.Context, phone string) (domain.User, error) {
	u, err := repo.dao.FindByPhone(ctx, phone)
	if err != nil {
		return domain.User{}, err
	}
	return repo.toDomain(u), nil
}

func (repo *CachedUserRepository) FindByWechat(ctx context.Context, openId string) (domain.User, error) {
	ue, err := repo.dao.FindByWechat(ctx, openId)
	if err != nil {
		return domain.User{}, err
	}
	return repo.toDomain(ue), nil
}
