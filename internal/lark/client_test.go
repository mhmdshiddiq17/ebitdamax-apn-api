package lark

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestExchangeCodesAndIdentity(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/open-apis/authen/v2/oauth/token":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["code_verifier"] != "verifier" || body["redirect_uri"] != "https://api.example.test/callback" {
				t.Errorf("payload OAuth tidak sesuai: %v, %v", body, err)
			}
			_, _ = w.Write([]byte(`{"code":"0","access_token":"browser-token"}`))
		case "/open-apis/auth/v3/app_access_token/internal":
			_, _ = w.Write([]byte(`{"code":0,"app_access_token":"app-token"}`))
		case "/open-apis/authen/v1/access_token":
			if r.Header.Get("Authorization") != "Bearer app-token" {
				t.Errorf("token aplikasi tidak digunakan")
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"access_token":"h5-token"}}`))
		case "/open-apis/authen/v1/user_info":
			if r.Header.Get("Authorization") != "Bearer browser-token" && r.Header.Get("Authorization") != "Bearer h5-token" {
				t.Errorf("token pengguna tidak digunakan")
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"open_id":"ou_test","enterprise_email":"Manager@Example.test"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Enabled: true, AppID: "app", AppSecret: "secret", Scopes: "contact:user.email:readonly",
		BaseURL: server.URL, AuthURL: server.URL + "/authorize",
		RedirectURI: "https://api.example.test/callback", FrontendURL: "https://web.example.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	authorize, err := url.Parse(client.AuthorizationURL("state", "challenge"))
	if err != nil || authorize.Query().Get("state") != "state" || authorize.Query().Get("code_challenge") != "challenge" || authorize.Query().Get("response_type") != "code" {
		t.Fatalf("URL OAuth tidak valid: %s, %v", authorize, err)
	}
	for _, exchange := range []func() (Identity, error){
		func() (Identity, error) {
			return client.ExchangeBrowser(context.Background(), "browser-code", "verifier")
		},
		func() (Identity, error) { return client.ExchangeH5(context.Background(), "h5-code") },
	} {
		identity, err := exchange()
		if err != nil || identity.OpenID != "ou_test" || identity.Email != "manager@example.test" {
			t.Fatalf("identitas Lark salah: %+v, %v", identity, err)
		}
	}
	if len(calls) != 5 {
		t.Fatalf("jumlah panggilan Lark salah: %v", calls)
	}
}

func TestMissingIdentityIsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"data":{"open_id":"ou_test"}}`))
	}))
	defer server.Close()
	client := &Client{Config: Config{BaseURL: server.URL}, HTTP: server.Client()}
	if _, err := client.UserInfo(context.Background(), "token"); err != ErrIdentity {
		t.Fatalf("email kosong harus ditolak, didapat %v", err)
	}
}
