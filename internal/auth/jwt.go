package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"jevon/internal/config"
)

type Claims struct {
	UserID   string `json:"user_id"`
	RoleName string `json:"role_name"`
	RoleID   int    `json:"role_id"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

type Service struct {
	cfg config.JWTConfig
}

func NewService(cfg config.JWTConfig) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) GenerateAccessToken(userID, email, roleName string, roleID int) (string, error) {
	claims := &Claims{
		UserID:   userID,
		RoleName: roleName,
		RoleID:   roleID,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.AccessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(s.cfg.AccessSecret))
}

func (s *Service) GenerateRefreshToken(userID string) (string, error) {
	claims := &jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.RefreshTTL)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(s.cfg.RefreshSecret))
}

func (s *Service) ParseAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.AccessSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (s *Service) ParseRefreshToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{},
		func(t *jwt.Token) (interface{}, error) {
			return []byte(s.cfg.RefreshSecret), nil
		})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid refresh token")
	}
	return claims.Subject, nil
}
