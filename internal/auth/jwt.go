package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

var errTokenInvalid = errors.New("invalid token")

func secretKey() []byte {
	return []byte(viper.GetString("web.password"))
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 24 小时有效的内部 token
func GenerateToken() (string, error) {
	return GenerateTokenForUser("system")
}

// GenerateTokenForUser 为登录用户生成 token
func GenerateTokenForUser(user string) (string, error) {
	claims := Claims{
		Username: user,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey())
}

// JWTAuthMiddleware Gin 鉴权中间件
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
			return
		}
		tokenStr := auth
		if strings.HasPrefix(auth, "Bearer ") {
			tokenStr = strings.TrimPrefix(auth, "Bearer ")
		} else if strings.HasPrefix(auth, "JWT ") {
			tokenStr = strings.TrimPrefix(auth, "JWT ")
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errTokenInvalid
			}
			return secretKey(), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: " + err.Error()})
			return
		}
		c.Set("claims", claims)
		c.Next()
	}
}
