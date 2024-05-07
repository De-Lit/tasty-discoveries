package jwt

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

var jwtKey = []byte("secret_key")

type claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func CreateToken(username string) (string, error) {
	now := time.Now()
	tokenLifeTime := now.Add(time.Minute * 5)
	claims := &claims{
		username,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(tokenLifeTime),
			Issuer:    "test",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return ss, nil
}

func ValidateToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(*claims)
	if ok && token.Valid {
		return claims.Username, nil
	}
	return "", err
}
