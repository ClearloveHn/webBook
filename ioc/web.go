package ioc

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
	"webBook/internal/web"
	ijwt "webBook/internal/web/jwt"
	"webBook/internal/web/middleware"
	"webBook/pkg/ginx/middleware/ratelimit"
	"webBook/pkg/limiter"
)

func InitWebServer(mdls []gin.HandlerFunc, userHdl *web.UserHandler, wechatHdl *web.OAuth2WechatHandler) *gin.Engine {
	server := gin.Default()
	server.Use(mdls...)
	userHdl.RegisterRoutes(server)
	wechatHdl.RegisterRoutes(server)
	return server
}

func InitGinMiddlewares(redisClient redis.Cmdable, hdl ijwt.Handler) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		cors.New(cors.Config{
			//AllowAllOrigins: true,
			//AllowOrigins:     []string{"http://localhost:3000"},
			AllowCredentials: true,

			AllowHeaders: []string{"Content-Type", "Authorization"},
			// 这个是允许前端访问你的后端响应中带的头部
			ExposeHeaders: []string{"x-jwt-token", "x-refresh-token"},
			//AllowHeaders: []string{"content-type"},
			//AllowMethods: []string{"POST"},
			AllowOriginFunc: func(origin string) bool {
				if strings.HasPrefix(origin, "http://localhost") {
					//if strings.Contains(origin, "localhost") {
					return true
				}
				return strings.Contains(origin, "your_company.com")
			},
			MaxAge: 12 * time.Hour,
		}),
		func(ctx *gin.Context) {
			println("这是我的 Middleware")
		},
		ratelimit.NewBuilder(limiter.NewRedisSlidingWindowLimiter(redisClient, time.Second, 1000)).Build(),
		middleware.NewLoginJWTMiddlewareBuilder(hdl).CheckLogin(),
	}
}
