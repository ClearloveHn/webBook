package domain

import "time"

// User 定义了一个用户类型(用于应用程序的内部逻辑和数据传输,以及在代码和数据库之间映射数据)
type User struct {
	Id       int64     // 用户的唯一标识符，通常为自增长的整数
	Email    string    // 用户的电子邮件地址，用于登录和通信
	Password string    // 用户的密码，应该是加密存储的
	Nickname string    // 用户的昵称，可以是用户的非正式名称
	Birthday time.Time // 用户的生日，使用Go的time包中的Time类型表示日期和时间
	AboutMe  string    // 用户自我介绍的文本
	Phone    string    // 用户的电话号码
	Ctime    time.Time // 用户创建时间，记录用户账号的创建时间
}
