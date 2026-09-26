package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/session"
	"agrinaspangan/ebitda-api/internal/token"
)

func TestIssueAuthSessionSetsValidToken(t *testing.T) {
	tokenService := token.NewService("rahasia-test", "ebitda-max-apn", time.Hour)
	miniRedis := miniredis.RunT(t)
	refreshStore := session.NewRefreshStore(
		redis.NewClient(&redis.Options{Addr: miniRedis.Addr()}),
		time.Hour,
	)

	previousDeps := AppDeps
	AppDeps = Deps{
		Tokens:        tokenService,
		Refresh:       refreshStore,
		AccessCookie:  "ebitda_access",
		RefreshCookie: "ebitda_refresh",
	}
	t.Cleanup(func() { AppDeps = previousDeps })

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	context.Request = context.Request.WithContext(context.Request.Context())

	if err := issueAuthSession(context, 7); err != nil {
		t.Fatalf("issue auth session: %v", err)
	}

	cookies := recorder.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookie (access + refresh), got %#v", cookies)
	}

	byName := make(map[string]*http.Cookie)
	for _, cookie := range cookies {
		byName[cookie.Name] = cookie
		if !cookie.HttpOnly || cookie.Value == "" {
			t.Fatalf("cookie tidak sesuai: %#v", cookie)
		}
	}

	claims, err := tokenService.Verify(byName["ebitda_access"].Value)
	if err != nil {
		t.Fatalf("token tidak valid: %v", err)
	}
	if claims.UserID != 7 || claims.SessionID == "" {
		t.Fatalf("claims tidak sesuai: %+v", claims)
	}

	refreshData, err := refreshStore.Get(context.Request.Context(), byName["ebitda_refresh"].Value)
	if err != nil || refreshData.UserID != 7 || refreshData.FamilyID != claims.SessionID {
		t.Fatalf("refresh token tidak sesuai: %+v, %v", refreshData, err)
	}
}

func TestRequireKdkmpManagerRejectsOtherRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name  string
		user  *models.User
		allow bool
	}{
		{name: "manager kdkmp", user: &models.User{Role: &models.Role{Domain: models.RoleDomainKdkmp, Slug: models.RoleSlugManager}}, allow: true},
		{name: "manager wilayah", user: &models.User{Role: &models.Role{Domain: models.RoleDomainKdkmp, Slug: models.RoleSlugRegionalManager}}},
		{name: "superadmin", user: &models.User{Role: &models.Role{Domain: models.RoleDomainApn, Slug: models.RoleSlugSuperadmin, Level: models.RoleLevelSuperadmin}}},
		{name: "tanpa user"},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if test.user != nil {
				context.Set(middleware.ContextUserKey, test.user)
			}
			RequireKdkmpManager()(context)
			if test.allow == context.IsAborted() {
				t.Fatalf("aborted = %t, want allow=%t", context.IsAborted(), test.allow)
			}
			if !test.allow && recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
			}
		})
	}
}
