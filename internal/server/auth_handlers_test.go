package server

import (
	"testing"

	"github.com/gin-gonic/gin"

	"agrinaspangan/ebitda-api/internal/models"
)

func TestUserResponseIncludesManagerSKDocumentForManager(t *testing.T) {
	metadata := `{"original_name":"sk-manager.pdf","size":2048,"uploaded_at":"2026-09-25T10:00:00Z"}`
	user := models.User{
		ID:                7,
		Name:              "Manager KDKMP",
		ManagerSKDocument: &metadata,
		Role: &models.Role{
			Domain: models.RoleDomainKdkmp,
			Slug:   models.RoleSlugManager,
		},
	}

	response := userResponse(&user)
	document, ok := response["manager_sk_document"].(gin.H)
	if !ok {
		t.Fatalf("manager_sk_document = %#v, want document metadata", response["manager_sk_document"])
	}
	if document["name"] != "sk-manager.pdf" || document["preview_url"] != "/users/7/manager-sk-document" {
		t.Fatalf("unexpected document response: %#v", document)
	}

	user.Role.Slug = models.RoleSlugRegionalManager
	if got, ok := userResponse(&user)["manager_sk_document"].(gin.H); !ok || got != nil {
		t.Fatalf("non-manager manager_sk_document = %#v, want nil", got)
	}
}
