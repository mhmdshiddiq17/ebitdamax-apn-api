// Package passkey menangani WebAuthn/passkeys: registrasi, login, dan
// pengelolaan perangkat. Session ceremony disimpan sementara di Redis.
package passkey

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/session"
)

const (
	registerKeyPrefix = "passkey:register:"
	loginKeyPrefix    = "passkey:login:"
	ceremonyTTL       = 5 * time.Minute
	defaultPasskey    = "Perangkat"
)

var (
	ErrDisabled           = errors.New("passkey tidak diaktifkan pada konfigurasi")
	ErrChallengeNotFound  = errors.New("sesi ceremony tidak ditemukan")
	ErrNoPasskey          = errors.New("passkey tidak tersedia untuk akun ini")
	ErrAlreadyRegistered  = errors.New("passkey ini sudah terdaftar")
	ErrPasskeyNotFound    = errors.New("passkey tidak ditemukan")
	ErrVerificationFailed = errors.New("verifikasi passkey gagal")
)

// Config konfigurasi Relying Party WebAuthn.
type Config struct {
	RPID          string
	RPDisplayName string
	RPOrigins     []string
}

// Service mengelola passkey user.
type Service struct {
	db       *gorm.DB
	sessions *session.Manager
	webauthn *webauthn.WebAuthn
}

// NewService membuat service passkey; jika konfigurasi kosong, service
// nonaktif (endpoint mengembalikan 503) sehingga aplikasi tetap berjalan.
func NewService(db *gorm.DB, sessions *session.Manager, cfg Config) (*Service, error) {
	service := &Service{db: db, sessions: sessions}

	if cfg.RPID == "" || len(cfg.RPOrigins) == 0 {
		return service, nil
	}

	instance, err := webauthn.New(&webauthn.Config{
		RPID:          cfg.RPID,
		RPDisplayName: cfg.RPDisplayName,
		RPOrigins:     cfg.RPOrigins,
	})
	if err != nil {
		return nil, err
	}

	service.webauthn = instance
	return service, nil
}

// Enabled mengecek apakah passkey dikonfigurasi.
func (s *Service) Enabled() bool {
	return s.webauthn != nil
}

// List mengembalikan passkey milik user.
func (s *Service) List(ctx context.Context, userID int64) ([]models.Passkey, error) {
	var passkeys []models.Passkey
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&passkeys).Error
	return passkeys, err
}

// BeginRegistration memulai ceremony registrasi passkey untuk user.
func (s *Service) BeginRegistration(ctx context.Context, user *models.User) (*protocol.CredentialCreation, string, error) {
	if !s.Enabled() {
		return nil, "", ErrDisabled
	}

	adapter, err := s.adapterFor(ctx, user)
	if err != nil {
		return nil, "", err
	}

	creation, sessionData, err := s.webauthn.BeginRegistration(adapter)
	if err != nil {
		return nil, "", err
	}

	token, err := s.storeChallenge(ctx, registerKeyPrefix, sessionData)
	if err != nil {
		return nil, "", err
	}

	return creation, token, nil
}

// FinishRegistration menyelesaikan ceremony registrasi dan menyimpan passkey.
func (s *Service) FinishRegistration(ctx context.Context, user *models.User, token string, request *http.Request, name string) (*models.Passkey, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}

	sessionData, err := s.loadChallenge(ctx, registerKeyPrefix, token)
	if err != nil {
		return nil, err
	}

	adapter, err := s.adapterFor(ctx, user)
	if err != nil {
		return nil, err
	}

	credential, err := s.webauthn.FinishRegistration(adapter, *sessionData, request)
	if err != nil {
		return nil, ErrVerificationFailed
	}

	_ = s.sessions.DeleteKey(ctx, registerKeyPrefix+token)

	credentialID := encodeCredentialID(credential.ID)

	var existing int64
	if err := s.db.WithContext(ctx).Model(&models.Passkey{}).
		Where("credential_id = ?", credentialID).
		Count(&existing).Error; err != nil {
		return nil, err
	}
	if existing > 0 {
		return nil, ErrAlreadyRegistered
	}

	payload, err := json.Marshal(credential)
	if err != nil {
		return nil, err
	}

	passkeyName := strings.TrimSpace(name)
	if passkeyName == "" {
		passkeyName = defaultPasskey
	}
	if len([]rune(passkeyName)) > 100 {
		passkeyName = string([]rune(passkeyName)[:100])
	}

	passkey := models.Passkey{
		UserID:       user.ID,
		Name:         passkeyName,
		CredentialID: credentialID,
		Credential:   string(payload),
	}

	if err := s.db.WithContext(ctx).Create(&passkey).Error; err != nil {
		return nil, err
	}

	return &passkey, nil
}

// Delete menghapus passkey milik user.
func (s *Service) Delete(ctx context.Context, userID int64, passkeyID int64) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", passkeyID, userID).
		Delete(&models.Passkey{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPasskeyNotFound
	}
	return nil
}

// BeginLogin memulai ceremony login passkey berdasarkan email.
func (s *Service) BeginLogin(ctx context.Context, email string) (*protocol.CredentialAssertion, string, error) {
	if !s.Enabled() {
		return nil, "", ErrDisabled
	}

	var user models.User
	err := s.db.WithContext(ctx).
		Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(email))).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", ErrNoPasskey
	}
	if err != nil {
		return nil, "", err
	}

	adapter, err := s.adapterFor(ctx, &user)
	if err != nil {
		return nil, "", err
	}
	if len(adapter.credentials) == 0 {
		return nil, "", ErrNoPasskey
	}

	assertion, sessionData, err := s.webauthn.BeginLogin(adapter)
	if err != nil {
		return nil, "", err
	}

	token, err := s.storeChallenge(ctx, loginKeyPrefix, sessionData)
	if err != nil {
		return nil, "", err
	}

	return assertion, token, nil
}

// FinishLogin menyelesaikan ceremony login dan mengembalikan user.
// Counter credential diperbarui agar deteksi clone tetap akurat.
func (s *Service) FinishLogin(ctx context.Context, token string, request *http.Request) (*models.User, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}

	sessionData, err := s.loadChallenge(ctx, loginKeyPrefix, token)
	if err != nil {
		return nil, err
	}

	userID, err := strconv.ParseInt(string(sessionData.UserID), 10, 64)
	if err != nil {
		return nil, ErrChallengeNotFound
	}

	var user models.User
	err = s.db.WithContext(ctx).Preload("Role").First(&user, userID).Error
	if err != nil {
		return nil, ErrNoPasskey
	}

	adapter, err := s.adapterFor(ctx, &user)
	if err != nil {
		return nil, err
	}

	credential, err := s.webauthn.FinishLogin(adapter, *sessionData, request)
	if err != nil {
		return nil, ErrVerificationFailed
	}

	_ = s.sessions.DeleteKey(ctx, loginKeyPrefix+token)

	credentialID := encodeCredentialID(credential.ID)
	payload, err := json.Marshal(credential)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	result := s.db.WithContext(ctx).Model(&models.Passkey{}).
		Where("user_id = ? AND credential_id = ?", user.ID, credentialID).
		Updates(map[string]any{
			"credential":   string(payload),
			"last_used_at": now,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNoPasskey
	}

	return &user, nil
}

// adapterFor memuat credential user dan membungkusnya sebagai webauthn.User.
func (s *Service) adapterFor(ctx context.Context, user *models.User) (*webauthnUser, error) {
	var rows []models.Passkey
	err := s.db.WithContext(ctx).
		Where("user_id = ?", user.ID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	credentials := make([]webauthn.Credential, 0, len(rows))
	for _, row := range rows {
		var credential webauthn.Credential
		if err := json.Unmarshal([]byte(row.Credential), &credential); err != nil {
			continue
		}
		credentials = append(credentials, credential)
	}

	return &webauthnUser{
		id:          user.ID,
		name:        user.Email,
		displayName: user.Name,
		credentials: credentials,
	}, nil
}

func (s *Service) storeChallenge(ctx context.Context, prefix string, data *webauthn.SessionData) (string, error) {
	token, err := session.NewToken()
	if err != nil {
		return "", err
	}

	if err := s.sessions.PutJSON(ctx, prefix+token, data, ceremonyTTL); err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) loadChallenge(ctx context.Context, prefix string, token string) (*webauthn.SessionData, error) {
	if token == "" {
		return nil, ErrChallengeNotFound
	}

	var data webauthn.SessionData
	err := s.sessions.GetJSON(ctx, prefix+token, &data)
	if errors.Is(err, session.ErrNotFound) {
		return nil, ErrChallengeNotFound
	}
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func encodeCredentialID(id []byte) string {
	return base64.RawURLEncoding.EncodeToString(id)
}

// webauthnUser mengimplementasikan interface webauthn.User.
type webauthnUser struct {
	id          int64
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte {
	return []byte(strconv.FormatInt(u.id, 10))
}

func (u *webauthnUser) WebAuthnName() string {
	return u.name
}

func (u *webauthnUser) WebAuthnDisplayName() string {
	return u.displayName
}

func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}
