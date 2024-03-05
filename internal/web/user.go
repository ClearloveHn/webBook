package web

import (
	regexp "github.com/dlclark/regexp2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"net/http"
	"time"
	"webBook/internal/domain"
	"webBook/internal/service"
)

const (
	emailRegexPattern    = "^\\w+([-+.]\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$"             // 邮箱格式正则表达式。
	passwordRegexPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,}$` // 密码格式正则表达式，要求字母、数字和特殊字符。
	bizLogin             = "login"                                                            // 登录业务标识符。
)

type UserHandler struct {
	emailRexExp    *regexp.Regexp       // 用于邮箱格式验证的正则表达式。
	passwordRexExp *regexp.Regexp       // 用于密码格式验证的正则表达式。
	svc            *service.UserService // 用户服务实例，处理用户业务逻辑。
	codeSvc        *service.CodeService // 验证码服务实例，处理验证码逻辑。
}

func NewUserHandler(svc *service.UserService, codeSvc *service.CodeService) *UserHandler {
	return &UserHandler{
		emailRexExp:    regexp.MustCompile(emailRegexPattern, regexp.None),
		passwordRexExp: regexp.MustCompile(passwordRegexPattern, regexp.None),
		svc:            svc,
		codeSvc:        codeSvc,
	}
}

func (h *UserHandler) RegisterRoutes(server *gin.Engine) {
	ug := server.Group("/users") // 创建用户相关的路由分组。
	// 注册具体的路由和对应的处理函数。
	ug.POST("/signup", h.SignUp)
	ug.POST("/login", h.LoginJWT)
	ug.POST("/edit", h.Edit)
	ug.GET("/profile", h.Profile)
	// 手机验证码登录相关路由。
	ug.POST("/login_sms/code/send", h.SendSMSLoginCode)
	ug.POST("/login_sms", h.LoginSMS)
}

// LoginSMS 处理手机验证码登录请求，首先验证验证码是否正确，然后查找或创建用户，并设置JWT令牌。
func (h *UserHandler) LoginSMS(ctx *gin.Context) {
	// 定义请求参数结构。
	type Req struct {
		Phone string `json:"phone"` // 用户手机号码。
		Code  string `json:"code"`  // 验证码。
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		// 如果请求参数绑定失败，则直接返回。
		return
	}
	// 验证验证码是否正确。
	ok, err := h.codeSvc.Verify(ctx, bizLogin, req.Phone, req.Code)
	if err != nil {
		// 如果验证码验证出错，则返回系统异常。
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统异常",
		})
		return
	}
	if !ok {
		// 如果验证码不正确，则返回错误信息。
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "验证码不对，请重新输入",
		})
		return
	}
	// 验证码正确，查找或创建用户。
	u, err := h.svc.FindOrCreate(ctx, req.Phone)
	if err != nil {
		// 如果查找或创建用户出错，则返回系统错误。
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		return
	}
	// 设置用户的JWT令牌。
	h.setJWTToken(ctx, u.Id)
	// 返回登录成功的信息。
	ctx.JSON(http.StatusOK, Result{
		Msg: "登录成功",
	})
}

// SendSMSLoginCode 这个方法用于处理发送登录验证码到用户手机的请求：
func (h *UserHandler) SendSMSLoginCode(ctx *gin.Context) {
	type Req struct {
		Phone string `json:"phone"` // 请求体需要包含一个手机号。
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		// 请求参数绑定失败。
		return
	}
	if req.Phone == "" {
		// 手机号为空时返回错误信息。
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "请输入手机号码",
		})
		return
	}
	// 调用验证码服务发送短信。
	err := h.codeSvc.Send(ctx, bizLogin, req.Phone)
	switch err {
	case nil:
		// 发送成功。
		ctx.JSON(http.StatusOK, Result{
			Msg: "发送成功",
		})
	case service.ErrCodeSendTooMany:
		// 发送频率过高。
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "短信发送太频繁，请稍后再试",
		})
	default:
		// 其他错误。
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
	}
}

// SignUp 用户的注册请求：
func (h *UserHandler) SignUp(ctx *gin.Context) {
	type SignUpReq struct {
		Email           string `json:"email"`           // 用户邮箱。
		Password        string `json:"password"`        // 用户密码。
		ConfirmPassword string `json:"confirmPassword"` // 确认密码。
	}

	var req SignUpReq
	if err := ctx.Bind(&req); err != nil {
		// 请求参数绑定失败。
		return
	}
	// 检查邮箱格式。
	isEmail, err := h.emailRexExp.MatchString(req.Email)
	if err != nil || !isEmail {
		// 邮箱格式不正确或者正则匹配出错。
		ctx.String(http.StatusOK, "非法邮箱格式")
		return
	}
	// 检查两次输入的密码是否一致。
	if req.Password != req.ConfirmPassword {
		ctx.String(http.StatusOK, "两次输入密码不对")
		return
	}
	// 检查密码强度。
	isPassword, err := h.passwordRexExp.MatchString(req.Password)
	if err != nil || !isPassword {
		ctx.String(http.StatusOK, "密码必须包含字母、数字、特殊字符，并且不少于八位")
		return
	}
	// 执行注册操作。
	err = h.svc.Signup(ctx, domain.User{
		Email:    req.Email,
		Password: req.Password,
	})
	switch err {
	case nil:
		ctx.String(http.StatusOK, "注册成功")
	case service.ErrDuplicateEmail:
		ctx.String(http.StatusOK, "邮箱冲突，请换一个")
	default:
		ctx.String(http.StatusOK, "系统错误")
	}
}

// LoginJWT 用户的登录请求，并返回JWT令牌：
func (h *UserHandler) LoginJWT(ctx *gin.Context) {
	type Req struct {
		Email    string `json:"email"`    // 用户邮箱。
		Password string `json:"password"` // 用户密码。
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		// 请求参数绑定失败。
		return
	}
	// 执行登录操作。
	u, err := h.svc.Login(ctx, req.Email, req.Password)
	switch err {
	case nil:
		// 登录成功，设置JWT令牌。
		h.setJWTToken(ctx, u.Id)
		ctx.String(http.StatusOK, "登录成功")
	case service.ErrInvalidUserOrPassword:
		// 用户名或密码不正确。
		ctx.String(http.StatusOK, "用户名或者密码不对")
	default:
		// 其他错误。
		ctx.String(http.StatusOK, "系统错误")
	}
}

// 生成并设置JWT令牌
func (h *UserHandler) setJWTToken(ctx *gin.Context, uid int64) {
	uc := UserClaims{
		Uid:       uid,
		UserAgent: ctx.GetHeader("User-Agent"),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)), // 设置过期时间。
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, uc)
	tokenStr, err := token.SignedString(JWTKey)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.Header("x-jwt-token", tokenStr)
}

func (h *UserHandler) Login(ctx *gin.Context) {
	type Req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		return
	}
	u, err := h.svc.Login(ctx, req.Email, req.Password)
	switch err {
	case nil:
		sess := sessions.Default(ctx)
		sess.Set("userId", u.Id)
		sess.Options(sessions.Options{
			// 十分钟
			MaxAge: 30,
		})
		err = sess.Save()
		if err != nil {
			ctx.String(http.StatusOK, "系统错误")
			return
		}
		ctx.String(http.StatusOK, "登录成功")
	case service.ErrInvalidUserOrPassword:
		ctx.String(http.StatusOK, "用户名或者密码不对")
	default:
		ctx.String(http.StatusOK, "系统错误")
	}
}

// Edit 用户编辑个人信息
func (h *UserHandler) Edit(ctx *gin.Context) {
	type Req struct {
		Nickname string `json:"nickname"` // 昵称。
		Birthday string `json:"birthday"` // 生日。
		AboutMe  string `json:"aboutMe"`  // 关于我。
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		// 请求参数绑定失败。
		return
	}
	// 验证用户身份。
	uc, ok := ctx.MustGet("user").(UserClaims)
	if !ok {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	// 验证生日格式。
	birthday, err := time.Parse("2006-01-02", req.Birthday)
	if err != nil {
		ctx.String(http.StatusOK, "生日格式不对")
		return
	}
	// 更新用户信息。
	err = h.svc.UpdateNonSensitiveInfo(ctx, domain.User{
		Id:       uc.Uid,
		Nickname: req.Nickname,
		Birthday: birthday,
		AboutMe:  req.AboutMe,
	})
	if err != nil {
		ctx.String(http.StatusOK, "系统异常")
		return
	}
	ctx.String(http.StatusOK, "更新成功")
}

// Profile 获取用户的个人资料
func (h *UserHandler) Profile(ctx *gin.Context) {
	uc, ok := ctx.MustGet("user").(UserClaims)
	if !ok {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	u, err := h.svc.FindById(ctx, uc.Uid)
	if err != nil {
		ctx.String(http.StatusOK, "系统异常")
		return
	}
	type User struct {
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		AboutMe  string `json:"aboutMe"`
		Birthday string `json:"birthday"`
	}
	ctx.JSON(http.StatusOK, User{
		Nickname: u.Nickname,
		Email:    u.Email,
		AboutMe:  u.AboutMe,
		Birthday: u.Birthday.Format(time.DateOnly),
	})
}

var JWTKey = []byte("k6CswdUm77WKcbM68UQUuxVsHSpTCwgK")

type UserClaims struct {
	jwt.RegisteredClaims        // 内嵌的RegisteredClaims结构体，包含了所有的标准JWT声明。
	Uid                  int64  // 用户的唯一标识符。
	UserAgent            string // 用户的代理信息。
}
