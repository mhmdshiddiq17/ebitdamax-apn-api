// Package twofactor menangani TOTP 2FA: enrollment, konfirmasi, recovery codes,
// verifikasi saat login, dan challenge sementara di Redis.
package twofactor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/crypto"
	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/session"
)

const (
	challengeKeyPrefix = "twofactor:challenge:"
	ChallengeTTL       = 5 * time.Minute
	maxChallengeTries  = 5
	recoveryCodeCount  = 8
	recoveryCodeChars  = 10
	recoveryCharset    = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

var (
	ErrAlreadyEnabled    = errors.New("2FA sudah aktif")
	ErrNotEnabled        = errors.New("2FA belum aktif")
	ErrInvalidCode       = errors.New("kode tidak valid")
	ErrTooManyTries      = errors.New("terlalu banyak percobaan")
	ErrChallengeNotFound = errors.New("challenge tidak ditemukan")
	ErrSecretUnsupported = errors.New("secret 2FA tidak dapat dibaca, lakukan enroll ulang")
)

// Challenge adalah state sementara antara login dan verifikasi 2FA.
type Challenge struct {
	UserID   int64 `json:"user_id"`
	Attempts int   `json:"attempts"`
}

// Service mengelola 2FA user.
type Service struct {
	db      *gorm.DB
	session *session.Manager
	key     []byte
	issuer  string
}

// NewService membuat service 2FA.
func NewService(db *gorm.DB, sessions *session.Manager, key []byte, issuer string) *Service {
	return &Service{db: db, session: sessions, key: key, issuer: issuer}
}

// Enabled mengecek apakah 2FA sudah dikonfirmasi aktif.
func (s *Service) Enabled(user *models.User) bool {
	return user.TwoFactorConfirmedAt != nil
}

// Begin membuat secret TOTP baru (status belum dikonfirmasi) dan mengembalikan
// secret + URI otpauth untuk QR code.
func (s *Service) Begin(ctx context.Context, user *models.User) (string, string, error) {
	if s.Enabled(user) {
		return "", "", ErrAlreadyEnabled
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: user.Email,
	})
	if err != nil {
		return "", "", err
	}

	encrypted, err := crypto.Encrypt(key.Secret(), s.key)
	if err != nil {
		return "", "", err
	}

	err = s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", user.ID).
		Updates(map[string]any{
			"two_factor_secret":         encrypted,
			"two_factor_confirmed_at":   nil,
			"two_factor_recovery_codes": nil,
		}).Error
	if err != nil {
		return "", "", err
	}

	user.TwoFactorSecret = &encrypted
	user.TwoFactorConfirmedAt = nil
	user.TwoFactorRecoveryCodes = nil

	return key.Secret(), key.URL(), nil
}

// Confirm memverifikasi kode TOTP pertama, mengaktifkan 2FA, dan mengembalikan
// recovery codes (hanya ditampilkan sekali).
func (s *Service) Confirm(ctx context.Context, user *models.User, code string) ([]string, error) {
	secret, err := s.decryptSecret(user)
	if err != nil {
		return nil, err
	}
	if !totp.Validate(strings.TrimSpace(code), secret) {
		return nil, ErrInvalidCode
	}

	codes, err := GenerateRecoveryCodes()
	if err != nil {
		return nil, err
	}
	hashes, err := hashRecoveryCodes(codes)
	if err != nil {
		return nil, err
	}
	encoded, err := encodeRecoveryCodes(hashes)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	err = s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", user.ID).
		Updates(map[string]any{
			"two_factor_confirmed_at":   now,
			"two_factor_recovery_codes": encoded,
		}).Error
	if err != nil {
		return nil, err
	}

	user.TwoFactorConfirmedAt = &now
	user.TwoFactorRecoveryCodes = &encoded

	return codes, nil
}

// Disable menonaktifkan 2FA dan menghapus secret serta recovery codes.
func (s *Service) Disable(ctx context.Context, user *models.User) error {
	err := s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", user.ID).
		Updates(map[string]any{
			"two_factor_secret":         nil,
			"two_factor_confirmed_at":   nil,
			"two_factor_recovery_codes": nil,
		}).Error
	if err != nil {
		return err
	}

	user.TwoFactorSecret = nil
	user.TwoFactorConfirmedAt = nil
	user.TwoFactorRecoveryCodes = nil

	return nil
}

// RegenerateRecoveryCodes membuat ulang recovery codes untuk 2FA yang aktif.
func (s *Service) RegenerateRecoveryCodes(ctx context.Context, user *models.User) ([]string, error) {
	if !s.Enabled(user) {
		return nil, ErrNotEnabled
	}

	codes, err := GenerateRecoveryCodes()
	if err != nil {
		return nil, err
	}
	hashes, err := hashRecoveryCodes(codes)
	if err != nil {
		return nil, err
	}
	encoded, err := encodeRecoveryCodes(hashes)
	if err != nil {
		return nil, err
	}

	err = s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("two_factor_recovery_codes", encoded).Error
	if err != nil {
		return nil, err
	}

	user.TwoFactorRecoveryCodes = &encoded

	return codes, nil
}

// Verify menerima kode TOTP atau recovery code (recovery code yang terpakai
// langsung dihapus).
func (s *Service) Verify(ctx context.Context, user *models.User, code string) error {
	if !s.Enabled(user) {
		return ErrNotEnabled
	}

	secret, err := s.decryptSecret(user)
	if err != nil {
		return err
	}

	if totp.Validate(strings.TrimSpace(code), secret) {
		return nil
	}

	hashes := decodeRecoveryCodes(deref(user.TwoFactorRecoveryCodes))
	remaining, ok := matchRecoveryCode(hashes, code)
	if !ok {
		return ErrInvalidCode
	}

	encoded, err := encodeRecoveryCodes(remaining)
	if err != nil {
		return err
	}

	err = s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("two_factor_recovery_codes", encoded).Error
	if err != nil {
		return err
	}

	user.TwoFactorRecoveryCodes = &encoded

	return nil
}

// CreateChallenge membuat token challenge sementara di Redis.
func (s *Service) CreateChallenge(ctx context.Context, userID int64) (string, error) {
	token, err := session.NewToken()
	if err != nil {
		return "", err
	}

	err = s.session.PutJSON(ctx, challengeKeyPrefix+token, Challenge{UserID: userID}, ChallengeTTL)
	if err != nil {
		return "", err
	}

	return token, nil
}

// ResolveChallenge mengambil state challenge.
func (s *Service) ResolveChallenge(ctx context.Context, token string) (*Challenge, error) {
	var challenge Challenge
	err := s.session.GetJSON(ctx, challengeKeyPrefix+token, &challenge)
	if errors.Is(err, session.ErrNotFound) {
		return nil, ErrChallengeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &challenge, nil
}

// FailChallenge menambah hitungan percobaan; menghapus challenge bila
// melewati batas dan mengembalikan ErrTooManyTries.
func (s *Service) FailChallenge(ctx context.Context, token string, challenge *Challenge) error {
	challenge.Attempts++

	if challenge.Attempts >= maxChallengeTries {
		_ = s.session.DeleteKey(ctx, challengeKeyPrefix+token)
		return ErrTooManyTries
	}

	if err := s.session.PutJSON(ctx, challengeKeyPrefix+token, challenge, ChallengeTTL); err != nil {
		return err
	}

	return ErrInvalidCode
}

// ClearChallenge menghapus challenge setelah verifikasi berhasil.
func (s *Service) ClearChallenge(ctx context.Context, token string) error {
	return s.session.DeleteKey(ctx, challengeKeyPrefix+token)
}

func (s *Service) decryptSecret(user *models.User) (string, error) {
	if user.TwoFactorSecret == nil || *user.TwoFactorSecret == "" {
		return "", ErrNotEnabled
	}

	secret, err := crypto.Decrypt(*user.TwoFactorSecret, s.key)
	if errors.Is(err, crypto.ErrUnsupportedFormat) {
		return "", ErrSecretUnsupported
	}
	if err != nil {
		return "", err
	}

	return secret, nil
}

// GenerateRecoveryCodes membuat 8 kode format "XXXXX-XXXXX".
func GenerateRecoveryCodes() ([]string, error) {
	codes := make([]string, 0, recoveryCodeCount)
	for i := 0; i < recoveryCodeCount; i++ {
		code, err := generateRecoveryCode()
		if err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, nil
}

func generateRecoveryCode() (string, error) {
	chars := make([]byte, 0, recoveryCodeChars)
	max := big.NewInt(int64(len(recoveryCharset)))

	for i := 0; i < recoveryCodeChars; i++ {
		index, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		chars = append(chars, recoveryCharset[index.Int64()])
	}

	half := recoveryCodeChars / 2
	return string(chars[:half]) + "-" + string(chars[half:]), nil
}

// normalizeRecoveryCode menghapus pemisah dan menyeragamkan huruf besar.
func normalizeRecoveryCode(code string) string {
	var builder strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(code)) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func hashRecoveryCodes(codes []string) ([]string, error) {
	hashes := make([]string, 0, len(codes))
	for _, code := range codes {
		hash, err := bcrypt.GenerateFromPassword([]byte(normalizeRecoveryCode(code)), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, string(hash))
	}
	return hashes, nil
}

// matchRecoveryCode mencari hash yang cocok dan mengembalikan sisa hash.
func matchRecoveryCode(hashes []string, code string) ([]string, bool) {
	normalized := normalizeRecoveryCode(code)
	if normalized == "" {
		return hashes, false
	}

	for i, hash := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(normalized)) == nil {
			remaining := make([]string, 0, len(hashes)-1)
			remaining = append(remaining, hashes[:i]...)
			remaining = append(remaining, hashes[i+1:]...)
			return remaining, true
		}
	}

	return hashes, false
}

func encodeRecoveryCodes(hashes []string) (string, error) {
	payload, err := json.Marshal(hashes)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func decodeRecoveryCodes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var hashes []string
	if err := json.Unmarshal([]byte(raw), &hashes); err != nil {
		return nil
	}
	return hashes
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
