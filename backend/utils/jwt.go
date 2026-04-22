package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwtKey is called at use-time so JWT_SECRET is read AFTER the .env is loaded.
func jwtKey() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

type Claims struct {
	UserID    int    `json:"user_id"`
	Role      string `json:"role"`       // admin | coach | student
	StudentID int    `json:"student_id"` // only for students

	jwt.RegisteredClaims
}

func GenerateToken(userID int, role string, studentID int) (string, error) {
	expiry := 24 * time.Hour
	claims := Claims{
		UserID:    userID,
		Role:      role,
		StudentID: studentID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey())
}

func ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey(), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, err
	}

	return claims, nil
}
