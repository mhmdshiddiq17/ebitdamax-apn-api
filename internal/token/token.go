// Package token menerbitkan & memverifikasi JWT access token.
package token

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalid = errors.New("token tidak valid")
	ErrExpired = errors.New("token sudah kedaluwarsa")
)

// Claims adalah data yang diambil dari access token.
type Claims struct {
	UserID    int64
	SessionID string
	JTI       string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// Service menerbitkan & memverifikasi access token (HS256).
type Service struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
}

// NewService membuat service token.
func NewService(secret string, issuer string, accessTTL time.Duration) *Service {
	return &Service{
		secret:    []byte(secret),
		issuer:    issuer,
		accessTTL: accessTTL,
	}
}

// AccessTTL mengembalikan masa berlaku access token.
func (s *Service) AccessTTL() time.Duration {
	return s.accessTTL
}

// IssueAccess menerbitkan access token untuk user pada satu sesi (family).
func (s *Service) IssueAccess(userID int64, sessionID string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(s.accessTTL)

	claims := jwt.MapClaims{
		"sub": strconv.FormatInt(userID, 10),
		"sid": sessionID,
		"jti": uuid.NewString(),
		"iss": s.issuer,
		"iat": now.Unix(),
		"exp": expiresAt.Unix(),
	}

	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return raw, expiresAt, nil
}

// Verify memvalidasi signature, issuer, dan masa berlaku token.
func (s *Service) Verify(raw string) (*Claims, error) {
	parsed, err := jwt.Parse(
		raw,
		func(t *jwt.Token) (any, error) {
			return s.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(s.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(60*time.Second),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpired
		}
		return nil, ErrInvalid
	}

	mapClaims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalid
	}

	subject, err := mapClaims.GetSubject()
	if err != nil {
		return nil, ErrInvalid
	}

	userID, err := strconv.ParseInt(subject, 10, 64)
	if err != nil {
		return nil, ErrInvalid
	}

	sessionID, _ := mapClaims["sid"].(string)
	jti, _ := mapClaims["jti"].(string)

	issuedAt := time.Time{}
	if value, err := mapClaims.GetIssuedAt(); err == nil && value != nil {
		issuedAt = value.Time
	}

	expiresAt := time.Time{}
	if value, err := mapClaims.GetExpirationTime(); err == nil && value != nil {
		expiresAt = value.Time
	}

	return &Claims{
		UserID:    userID,
		SessionID: sessionID,
		JTI:       jti,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}, nil
}
