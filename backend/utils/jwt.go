package utils

import (
	"log"
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
	log.Printf("[JWT] Validating token: %s...\n", tokenStr[:20])

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		key := jwtKey()
		log.Printf("[JWT] Key used for verification: %d bytes\n", len(key))
		return key, nil
	})

	if err != nil {
		log.Printf("[JWT] Parse error: %v\n", err)
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		log.Printf("[JWT] Claims cast failed\n")
		return nil, jwt.ErrTokenNotValidYet
	}

	log.Printf("[JWT] Token valid: %v, Claims: UserID=%d, Role=%s\n", token.Valid, claims.UserID, claims.Role)

	if !token.Valid {
		log.Printf("[JWT] Token marked as invalid\n")
		return nil, jwt.ErrTokenNotValidYet
	}

	return claims, nil
}
