package jwt

import (
	"fmt"

	"github.com/golang-jwt/jwt"
)

type JWT struct {
	secret []byte
}

func New(secret []byte) *JWT {
	return &JWT{
		secret: secret,
	}
}

func (s *JWT) Create(key, value string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		key: value,
	})
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("failed signe token: %w", err)
	}

	return tokenString, nil
}

func (s *JWT) Verify(signedData string, key string) (string, bool, error) {
	token, err := jwt.Parse(signedData, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unknown signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		return "", false, fmt.Errorf("failed parse jwt token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if uniqueID, ok := claims[key].(string); ok {
			if uniqueID != "" {
				return uniqueID, true, nil
			}
		}
	}

	return "", false, nil
}
