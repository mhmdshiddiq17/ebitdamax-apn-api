package server

import (
	"testing"

	"agrinaspangan/ebitda-api/internal/lark"
	"agrinaspangan/ebitda-api/internal/models"
)

func TestCanBindLarkOnlyMatchingKdkmpManager(t *testing.T) {
	manager := &models.User{
		Email: "Manager@Example.test",
		Role:  &models.Role{Domain: models.RoleDomainKdkmp, Slug: models.RoleSlugManager},
	}
	identity := lark.Identity{OpenID: "ou_one", Email: "manager@example.test"}
	if !canBindLark(manager, identity) {
		t.Fatal("manager dengan email cocok harus diterima")
	}
	otherOpenID := "ou_other"
	manager.LarkOpenID = &otherOpenID
	if canBindLark(manager, identity) {
		t.Fatal("open_id yang sudah ditautkan tidak boleh ditimpa")
	}
	manager.LarkOpenID = nil
	manager.Role.Domain = "other"
	if canBindLark(manager, identity) {
		t.Fatal("manager di luar domain KDKMP harus ditolak")
	}
	manager.Role.Domain = models.RoleDomainKdkmp
	identity.Email = "another@example.test"
	if canBindLark(manager, identity) {
		t.Fatal("email berbeda harus ditolak")
	}
}
