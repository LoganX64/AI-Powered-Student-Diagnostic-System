package utils

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret   string
	jwtExpiryStr string
	jwtIssuer   string
)

func InitJWTConfig(secret, expiry, issuer string) {
	jwtSecret = secret
	jwtExpiryStr = expiry
	jwtIssuer = issuer
}

func jwtKey() []byte {
	return []byte(jwtSecret)
}

func jwtExpiry() time.Duration {
	if jwtExpiryStr == "" {
		jwtExpiryStr = "4h"
	}
	expiry, err := time.ParseDuration(jwtExpiryStr)
	if err != nil {
		log.Printf("[JWT] Invalid JWT_EXPIRY format '%s', using default 4h: %v\n", jwtExpiryStr, err)
		return 4 * time.Hour
	}
	return expiry
}

type Claims struct {
	UserID    int    `json:"user_id"`
	Role      string `json:"role"`       // admin | coach | student
	StudentID int    `json:"student_id"` // only for students
	TenantID  int    `json:"tenant_id"`

	jwt.RegisteredClaims
}

func GenerateToken(userID int, role string, studentID int, tenantID int) (string, error) {
	expiry := jwtExpiry()
	claims := Claims{
		UserID:    userID,
		Role:      role,
		StudentID: studentID,
		TenantID:  tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{jwtIssuer},
			Issuer:    jwtIssuer,
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
		log.Printf("[JWT] Parse error: %v\n", err)
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, jwt.ErrTokenNotValidYet
	}

	if !token.Valid {
		log.Printf("[JWT] Token marked as invalid for UserID=%d\n", claims.UserID)
		return nil, jwt.ErrTokenNotValidYet
	}

	if claims.Issuer != jwtIssuer {
		log.Printf("[JWT] Invalid issuer: %s\n", claims.Issuer)
		return nil, fmt.Errorf("invalid issuer")
	}
	audValid := false
	for _, a := range claims.Audience {
		if a == jwtIssuer {
			audValid = true
			break
		}
	}
	if !audValid {
		log.Printf("[JWT] Invalid audience: %v\n", claims.Audience)
		return nil, fmt.Errorf("invalid audience")
	}

	return claims, nil
}
