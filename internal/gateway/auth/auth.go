package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID       int64  `json:"uid"`
	Username     string `json:"username"`
	TokenVersion int    `json:"tv"`
	jwt.RegisteredClaims
}

// func Login(ctx *fasthttp.RequestCtx) {
// 	var req struct {
// 		Username string `json:"username"`
// 		Password string `json:"password"`
// 	}
// 	json.Unmarshal(ctx.PostBody(), &req)

// 	user, err := repo.GetUserByUsername(req.Username)
// 	if err != nil || !CheckPassword(req.Password, user.PasswordHash) {
// 		ctx.SetStatusCode(401)
// 		return
// 	}

// 	token, err := GenerateJWT(user)
// 	if err != nil {
// 		ctx.SetStatusCode(500)
// 		return
// 	}

// 	ctx.SetContentType("application/json")
// 	ctx.SetBodyString(fmt.Sprintf(`{"access_token":"%s"}`, token))
// }

// func Logout(ctx *fasthttp.RequestCtx) {
// 	claims := ctx.UserValue("claims").(*Claims)

// 	// token_version + 1
// 	err := repo.IncreaseTokenVersion(claims.UserID)
// 	if err != nil {
// 		ctx.SetStatusCode(500)
// 		return
// 	}

// 	ctx.SetStatusCode(200)
// }

// func GenerateJWT(user *User) (string, error) {
// 	claims := Claims{
// 		UserID:       user.ID,
// 		Username:     user.Username,
// 		TokenVersion: user.TokenVersion,
// 		RegisteredClaims: jwt.RegisteredClaims{
// 			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
// 			IssuedAt:  jwt.NewNumericDate(time.Now()),
// 			Issuer:    "opshub-iam",
// 		},
// 	}

// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
// 	return token.SignedString([]byte(jwtSecret))
// }
