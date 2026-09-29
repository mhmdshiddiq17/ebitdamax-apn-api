package lark

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

var ErrIdentity = errors.New("identitas Lark tidak lengkap")

type Config struct {
	Enabled     bool
	AppID       string
	AppSecret   string
	BaseURL     string
	AuthURL     string
	RedirectURI string
	FrontendURL string
	Scopes      string
}

type Client struct {
	Config Config
	HTTP   *http.Client
}

type Identity struct {
	OpenID string
	Email  string
}

func NewClient(cfg Config) (*Client, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if cfg.AppID == "" || cfg.AppSecret == "" || cfg.Scopes == "" {
		return nil, errors.New("konfigurasi SSO Lark belum lengkap")
	}
	for _, raw := range []string{cfg.BaseURL, cfg.AuthURL, cfg.RedirectURI, cfg.FrontendURL} {
		parsed, err := url.Parse(raw)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return nil, errors.New("URL SSO Lark tidak valid")
		}
	}
	return &Client{Config: cfg, HTTP: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (c *Client) AuthorizationURL(state, challenge string) string {
	u, _ := url.Parse(c.Config.AuthURL)
	query := u.Query()
	query.Set("client_id", c.Config.AppID)
	query.Set("response_type", "code")
	query.Set("redirect_uri", c.Config.RedirectURI)
	query.Set("scope", c.Config.Scopes)
	query.Set("state", state)
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")
	u.RawQuery = query.Encode()
	return u.String()
}

// Browser memakai OAuth v2; kode H5 memakai endpoint login-free v1 sesuai panduan H5 Lark.
func (c *Client) ExchangeBrowser(ctx context.Context, code, verifier string) (Identity, error) {
	var token struct {
		Code        json.Number `json:"code"`
		AccessToken string      `json:"access_token"`
	}
	err := c.request(ctx, http.MethodPost, "/open-apis/authen/v2/oauth/token", "", map[string]string{
		"grant_type": "authorization_code", "client_id": c.Config.AppID,
		"client_secret": c.Config.AppSecret, "code": code,
		"redirect_uri": c.Config.RedirectURI, "code_verifier": verifier,
	}, &token)
	if err != nil || token.Code.String() != "0" || token.AccessToken == "" {
		return Identity{}, errors.New("penukaran kode OAuth Lark gagal")
	}
	return c.UserInfo(ctx, token.AccessToken)
}

func (c *Client) ExchangeH5(ctx context.Context, code string) (Identity, error) {
	var app struct {
		Code        json.Number `json:"code"`
		AccessToken string      `json:"app_access_token"`
	}
	err := c.request(ctx, http.MethodPost, "/open-apis/auth/v3/app_access_token/internal", "", map[string]string{
		"app_id": c.Config.AppID, "app_secret": c.Config.AppSecret,
	}, &app)
	if err != nil || app.Code.String() != "0" || app.AccessToken == "" {
		return Identity{}, errors.New("token aplikasi Lark gagal")
	}

	var token struct {
		Code json.Number `json:"code"`
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	err = c.request(ctx, http.MethodPost, "/open-apis/authen/v1/access_token", app.AccessToken, map[string]string{
		"grant_type": "authorization_code", "code": code,
	}, &token)
	if err != nil || token.Code.String() != "0" || token.Data.AccessToken == "" {
		return Identity{}, errors.New("penukaran kode H5 Lark gagal")
	}
	return c.UserInfo(ctx, token.Data.AccessToken)
}

func (c *Client) UserInfo(ctx context.Context, accessToken string) (Identity, error) {
	var response struct {
		Code json.Number `json:"code"`
		Data struct {
			OpenID          string `json:"open_id"`
			Email           string `json:"email"`
			EnterpriseEmail string `json:"enterprise_email"`
		} `json:"data"`
	}
	err := c.request(ctx, http.MethodGet, "/open-apis/authen/v1/user_info", accessToken, nil, &response)
	if err != nil || response.Code.String() != "0" {
		return Identity{}, errors.New("informasi pengguna Lark gagal")
	}
	email := normalizedEmail(response.Data.EnterpriseEmail)
	if email == "" {
		email = normalizedEmail(response.Data.Email)
	}
	if response.Data.OpenID == "" || email == "" {
		return Identity{}, ErrIdentity
	}
	return Identity{OpenID: response.Data.OpenID, Email: email}, nil
}

func normalizedEmail(value string) string {
	email := strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return ""
	}
	return email
}

func (c *Client) request(ctx context.Context, method, path, bearer string, payload any, dest any) error {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.Config.BaseURL, "/")+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("Lark HTTP %d", res.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(dest)
}
