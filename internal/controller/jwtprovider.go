package controller

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type (
	Claims struct {
		jwt.RegisteredClaims
		UserID int
	}

	JwtProvider struct {
		TokenExpTime time.Duration
		SecretKey    string
	}
)

func NewJwtProvider(tokenexp time.Duration, seckey string) JwtProvider {
	return JwtProvider{
		TokenExpTime: tokenexp,
		SecretKey:    seckey,
	}
}

func (j *JwtProvider) GetStr(secretKey string, uid int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.TokenExpTime)),
		},
		UserID: uid,
	})

	tokenString, err := token.SignedString([]byte(secretKey))

	if err != nil {
		return "", fmt.Errorf("CAN'T CREATE SIGNED STRING: [%w]", err)
	}
	return tokenString, nil
}

func (j *JwtProvider) GetUserID(tokenString string) (int, error) {
	// создаём экземпляр структуры с утверждениями
	claims := &Claims{}
	// парсим из строки токена tokenString в структуру claims
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(j.SecretKey), nil
	})

	if err != nil {
		return -1, fmt.Errorf("CAN'T ParseWithClaims: [%w]", err)
	}

	if !token.Valid {
		return -1, fmt.Errorf("TOKEN IS'T VALID")
	}

	// возвращаем ID пользователя в читаемом виде

	return claims.UserID, nil
}
