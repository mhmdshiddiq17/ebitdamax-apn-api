package server

import (
	"errors"
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
)

func TestReplaceSessionInvalidatesPresentedSession(t *testing.T) {
	miniRedis := miniredis.RunT(t)
	manager := session.NewManager(redis.NewClient(&redis.Options{Addr: miniRedis.Addr()}), time.Hour)
	oldSession, err := manager.Create(t.Context(), 7)
	if err != nil {
		t.Fatalf("create old session: %v", err)
	}

	previousDeps := AppDeps
	AppDeps = Deps{Session: manager, SessionCookie: "ebitda_session", SessionTTL: time.Hour}
	t.Cleanup(func() { AppDeps = previousDeps })

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	request.AddCookie(&http.Cookie{Name: "ebitda_session", Value: oldSession})
	context.Request = request

	if err := replaceSession(context, 7); err != nil {
		t.Fatalf("replace session: %v", err)
	}
	if _, err := manager.Get(t.Context(), oldSession); !errors.Is(err, session.ErrNotFound) {
		t.Fatalf("old session still valid: %v", err)
	}

	response := recorder.Result()
	cookies := response.Cookies()
	if len(cookies) != 1 || cookies[0].Value == "" || cookies[0].Value == oldSession {
		t.Fatalf("replacement cookie = %#v", cookies)
	}
	if data, err := manager.Get(t.Context(), cookies[0].Value); err != nil || data.UserID != 7 {
		t.Fatalf("replacement session = %#v, %v", data, err)
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
