package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/config"
	"agrinaspangan/ebitda-api/internal/database"
	"agrinaspangan/ebitda-api/internal/models"
)

// Seeder data awal (idempotent): roles + akun superadmin & manager contoh.
// Jalankan: go run ./cmd/seed
func main() {
	config.LoadEnv()
	db := database.Connect()

	roles := []models.Role{
		{Name: "Superadmin", Slug: models.RoleSlugSuperadmin, Level: models.RoleLevelSuperadmin, Domain: models.RoleDomainApn},
		{Name: "Kepala Toko / Manager", Slug: models.RoleSlugManager, Level: models.RoleLevelManager, Domain: models.RoleDomainKdkmp},
		{Name: "Manager Wilayah", Slug: models.RoleSlugRegionalManager, Level: models.RoleLevelManager, Domain: models.RoleDomainKdkmp},
	}

	roleIDs := make(map[string]int64)
	for _, role := range roles {
		saved, err := upsertRole(db, role)
		if err != nil {
			log.Fatalf("seed role %s: %v", role.Slug, err)
		}
		roleIDs[saved.Slug] = saved.ID
		fmt.Printf("role   %-18s id=%d\n", saved.Slug, saved.ID)
	}

	accounts := []struct {
		Name  string
		Email string
		Pass  string
		Role  string
	}{
		{
			Name:  "Superadmin",
			Email: config.GetEnv("SEED_SUPERADMIN_EMAIL", "superadmin@agrinas.test"),
			Pass:  config.GetEnv("SEED_SUPERADMIN_PASSWORD", "password123"),
			Role:  models.RoleSlugSuperadmin,
		},
		{
			Name:  "Manager KDKMP Contoh",
			Email: config.GetEnv("SEED_MANAGER_EMAIL", "manager@agrinas.test"),
			Pass:  config.GetEnv("SEED_MANAGER_PASSWORD", "password123"),
			Role:  models.RoleSlugManager,
		},
	}

	for _, account := range accounts {
		user, created, err := seedUser(db, account.Name, account.Email, account.Pass, roleIDs[account.Role])
		if err != nil {
			log.Fatalf("seed user %s: %v", account.Email, err)
		}
		if created {
			fmt.Printf("user   %-32s id=%d (dibuat, role=%s)\n", account.Email, user.ID, account.Role)
		} else {
			fmt.Printf("user   %-32s id=%d (sudah ada, dilewati)\n", account.Email, user.ID)
		}
	}

	fmt.Println()
	fmt.Println("seed selesai — kredensial development:")
	fmt.Printf("  superadmin : %s / %s\n", config.GetEnv("SEED_SUPERADMIN_EMAIL", "superadmin@agrinas.test"), config.GetEnv("SEED_SUPERADMIN_PASSWORD", "password123"))
	fmt.Printf("  manager    : %s / %s\n", config.GetEnv("SEED_MANAGER_EMAIL", "manager@agrinas.test"), config.GetEnv("SEED_MANAGER_PASSWORD", "password123"))
}

func upsertRole(db *gorm.DB, role models.Role) (*models.Role, error) {
	var existing models.Role
	err := db.Where("slug = ?", role.Slug).First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		role.UUID = uuid.NewString()
		if err := db.Create(&role).Error; err != nil {
			return nil, err
		}
		return &role, nil
	}
	if err != nil {
		return nil, err
	}

	existing.Name = role.Name
	existing.Level = role.Level
	existing.Domain = role.Domain
	if existing.CreatedAt.IsZero() {
		existing.CreatedAt = time.Now()
	}
	if err := db.Save(&existing).Error; err != nil {
		return nil, err
	}
	return &existing, nil
}

func seedUser(db *gorm.DB, name string, email string, password string, roleID int64) (*models.User, bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	var existing models.User
	err := db.Where("LOWER(email) = ?", email).First(&existing).Error
	if err == nil {
		return &existing, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, false, err
	}

	username, err := uniqueUsername(db, name)
	if err != nil {
		return nil, false, err
	}

	now := time.Now()
	user := models.User{
		RoleID:          &roleID,
		Name:            name,
		Username:        &username,
		Email:           email,
		EmailVerifiedAt: &now,
		Password:        string(hash),
	}

	if err := db.Create(&user).Error; err != nil {
		return nil, false, err
	}

	return &user, true, nil
}

func uniqueUsername(db *gorm.DB, name string) (string, error) {
	base := slugify(name)
	if base == "" {
		base = "user"
	}

	candidate := base
	for i := 2; ; i++ {
		var count int64
		if err := db.Model(&models.User{}).Where("username = ?", candidate).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

func slugify(value string) string {
	var builder strings.Builder
	lastDash := false

	for _, r := range strings.ToLower(value) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		case !lastDash:
			builder.WriteRune('-')
			lastDash = true
		}
	}

	return strings.Trim(builder.String(), "-")
}
