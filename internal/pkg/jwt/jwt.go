package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const TokenExpireDuration = time.Hour * 1

var CustomSecret = []byte("opshub_secret_key_for_jwt_nsfhqo75qsjd3e4hlslap2s")

// CustomClaims 自定义声明类型 并内嵌jwt.RegisteredClaims
// jwt包自带的jwt.RegisteredClaims只包含了官方字段
// 假设我们这里需要额外记录一个username字段，所以要自定义结构体
// 如果想要保存更多信息，都可以添加到这个结构体中
type CustomClaims struct {
	// 可根据需要自行添加字段
	UserID               int64  `json:"user_id"`
	Username             string `json:"username"`
	jwt.RegisteredClaims        // 内嵌标准的声明
}

// GenToken 生成JWT
func GenToken(UserID int64, username string) (string, error) {
	// 创建一个我们自己的声明的数据
	claims := CustomClaims{
		UserID,
		username, // 自定义字段
		jwt.RegisteredClaims{
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpireDuration)), // 过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "opshub-iam", // 签发人 发行人
		},
	}
	// 使用指定的签名方法创建签名对象
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// 使用指定的secret签名并获得完整的编码后的字符串token
	return token.SignedString(CustomSecret)
}

// ParseToken 解析JWT
func ParseToken(tokenString string) (*CustomClaims, error) {
	// 解析token
	var claims = new(CustomClaims)
	// 如果是自定义Claim结构体则需要使用 ParseWithClaims 方法
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (i interface{}, err error) {
		// 直接使用标准的Claim则可以直接使用Parse方法
		//token, err := jwt.Parse(tokenString, func(token *jwt.Token) (i interface{}, err error) {
		return CustomSecret, nil
	})
	if err != nil {
		return nil, err
	}
	// 对token对象中的Claim进行类型断言
	if token.Valid { // 校验token
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
