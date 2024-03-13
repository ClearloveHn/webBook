package dao

import (
	"context"
	"database/sql"
	"errors"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"time"
)

var (
	ErrDuplicateEmail = errors.New("邮箱冲突")     // 定义邮箱冲突的错误。
	ErrRecordNotFound = gorm.ErrRecordNotFound // 将GORM的记录未找到错误直接赋值给ErrRecordNotFound。
)

type UserDAO interface {
	Insert(ctx context.Context, u User) error
	FindByEmail(ctx context.Context, email string) (User, error)
	UpdateById(ctx context.Context, entity User) error
	FindById(ctx context.Context, uid int64) (User, error)
	FindByPhone(ctx context.Context, phone string) (User, error)
	FindByWechat(ctx context.Context, openId string) (User, error)
}

type GORMUserDAO struct {
	db *gorm.DB // db字段，类型为*gorm.DB，指定数据库连接。
}

func NewUserDAO(db *gorm.DB) UserDAO {
	return &GORMUserDAO{
		db: db,
	}
}

// User 定义User结构体，映射数据库中的用户表。
type User struct {
	Id            int64          `gorm:"primaryKey,autoIncrement"` // 主键，自动增长。
	Email         sql.NullString `gorm:"unique"`                   // Email字段，唯一性约束。
	Password      string         // 密码字段。
	Nickname      string         `gorm:"type=varchar(128)"` // 昵称字段，指定类型为varchar(128)。
	Birthday      int64          // 生日字段。
	AboutMe       string         `gorm:"type=varchar(4096)"` // 自我介绍字段，指定类型为varchar(4096)。
	Phone         sql.NullString `gorm:"unique"`             // 电话字段，唯一性约束。
	Ctime         int64          // 创建时间。
	Utime         int64          // 更新时间。
	WechatOpenId  sql.NullString `gorm:"unique"`
	WechatUnionId sql.NullString
}

// Insert Insert方法，插入新的用户记录。
func (dao *GORMUserDAO) Insert(ctx context.Context, u User) error {
	now := time.Now().UnixMilli()
	u.Ctime = now
	u.Utime = now
	err := dao.db.WithContext(ctx).Create(&u).Error // 在数据库中创建新的用户记录。
	var me *mysql.MySQLError
	if errors.As(err, &me) { // 错误类型断言。
		const duplicateErr uint16 = 1062 // MySQL的重复键错误码。
		if me.Number == duplicateErr {   // 判断是否是因为邮箱冲突。
			return ErrDuplicateEmail // 返回邮箱冲突的错误。
		}
	}
	return err // 返回其他类型的错误。
}

// FindByEmail 通过Email查找用户。
func (dao *GORMUserDAO) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := dao.db.WithContext(ctx).Where("email=?", email).First(&u).Error // 查询数据库。
	return u, err
}

// UpdateById 根据ID更新用户信息。
func (dao *GORMUserDAO) UpdateById(ctx context.Context, entity User) error {
	// 使用GORM的Model和Where方法定位记录，并使用Updates方法更新记录。
	return dao.db.WithContext(ctx).Model(&entity).Where("id = ?", entity.Id).
		Updates(map[string]any{
			"utime":    time.Now().UnixMilli(),
			"nickname": entity.Nickname,
			"birthday": entity.Birthday,
			"about_me": entity.AboutMe,
		}).Error
}

// FindById 通过ID查找用户
func (dao *GORMUserDAO) FindById(ctx context.Context, uid int64) (User, error) {
	var res User
	err := dao.db.WithContext(ctx).Where("id = ?", uid).First(&res).Error // 查询数据库。
	return res, err
}

// FindByPhone 通过电话号码查找用户。
func (dao *GORMUserDAO) FindByPhone(ctx context.Context, phone string) (User, error) {
	var res User
	err := dao.db.WithContext(ctx).Where("phone = ?", phone).First(&res).Error // 查询数据库。
	return res, err
}

func (dao *GORMUserDAO) FindByWechat(ctx context.Context, openId string) (User, error) {
	var u User
	err := dao.db.WithContext(ctx).Where("wechat_open_id=?", openId).First(&u).Error
	return u, err
}
