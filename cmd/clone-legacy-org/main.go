// Command clone-legacy-org menyalin data organisasi (entry KDKMP, akun Manager
// KDKMP, dan akun Manager Wilayah demo) dari database legacy ke database
// refactor secara incremental & idempotent (upsert by NIK/email).
//
// Jalankan: go run ./cmd/clone-legacy-org [--apply] [--limit N]
//
//	[--reset-manager-passwords N] [--regional-managers N]
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"agrinaspangan/ebitda-api/config"
	"agrinaspangan/ebitda-api/internal/database"
	"agrinaspangan/ebitda-api/internal/models"
)

const (
	qaPassword       = "password123"
	qaEmailDomain    = "agrinas.test"
	insertBatchSize  = 200
	regionalUsername = "manager-wilayah"
)

type cloneOptions struct {
	Apply                 bool
	Limit                 int
	ResetManagerPasswords int
	RegionalManagers      int
}

type legacyEntry struct {
	ID             int64      `gorm:"primaryKey"`
	NamaKoperasi   string     `gorm:"column:nama_koperasi"`
	JumlahKaryawan int        `gorm:"column:jumlah_karyawan"`
	Catatan        *string    `gorm:"column:catatan"`
	CreatedAt      *time.Time `gorm:"column:created_at"`
	UpdatedAt      *time.Time `gorm:"column:updated_at"`
	NIK            *string    `gorm:"column:nik"`
	NamaKodam      *string    `gorm:"column:nama_kodam"`
	NamaKorem      *string    `gorm:"column:nama_korem"`
	NamaKodim      *string    `gorm:"column:nama_kodim"`
	Desa           *string    `gorm:"column:desa"`
	Kecamatan      *string    `gorm:"column:kecamatan"`
	KotaKabupaten  *string    `gorm:"column:kota_kabupaten"`
	Batch          *string    `gorm:"column:batch"`
	Provinsi       *string    `gorm:"column:provinsi"`
}

func (legacyEntry) TableName() string { return "sdm_kdkmp_entries" }

// managerPlan merencanakan satu akun manager: link ke akun target yang sudah
// ada (ExistingID != 0) atau buat akun baru.
type managerPlan struct {
	Legacy     models.User
	ExistingID int64
	EntryNIK   string
	Username   string
}

type regionalPlan struct {
	Provinsi string
	Email    string
	Name     string
	Username string
}

type clonePlan struct {
	Entries         []legacyEntry
	EntriesSkipped  int
	Managers        []managerPlan
	ManagersSkipped int
	Regional        []regionalPlan
	Reset           []managerPlan
}

type sourceState struct {
	ManagerRole  models.Role
	Managers     []models.User
	Entries      []legacyEntry
	ProvinceRows map[string]int
}

type targetState struct {
	EntryNIKs    map[string]bool
	UsersByEmail map[string]models.User
	Usernames    map[string]string
}

func main() {
	apply := flag.Bool("apply", false, "tulis data ke database refactor")
	limit := flag.Int("limit", 0, "batasi jumlah entry & manager yang diproses (0 = semua)")
	resetPasswords := flag.Int("reset-manager-passwords", 0, "reset password N manager pertama (punya entry) ke password123")
	regionalManagers := flag.Int("regional-managers", 0, "buat N akun manager wilayah demo (scope province)")
	flag.Parse()

	config.LoadEnv()
	options := cloneOptions{
		Apply:                 *apply,
		Limit:                 *limit,
		ResetManagerPasswords: *resetPasswords,
		RegionalManagers:      *regionalManagers,
	}
	if err := options.validate(); err != nil {
		log.Fatal(err)
	}

	source, closeSource, err := connectLegacy()
	if err != nil {
		log.Fatal(err)
	}
	defer closeSource()

	state, err := loadSourceState(source, options.Limit)
	if err != nil {
		log.Fatal(err)
	}

	target := database.Connect()
	targetData, err := loadTargetState(target)
	if err != nil {
		log.Fatal(err)
	}

	plan := buildPlan(state, targetData, options)
	printPlan(plan, options)
	if !options.Apply {
		return
	}

	if err := applyPlan(target, state, plan, options); err != nil {
		log.Fatal(err)
	}
	log.Println("clone data organisasi selesai")
}

func (options cloneOptions) validate() error {
	if options.Limit < 0 || options.ResetManagerPasswords < 0 || options.RegionalManagers < 0 {
		return errors.New("nilai flag tidak boleh negatif")
	}
	return nil
}

func connectLegacy() (*gorm.DB, func(), error) {
	db, err := gorm.Open(postgres.Open(legacyDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, nil, fmt.Errorf("koneksi database legacy gagal: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("akses koneksi legacy gagal: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, nil, fmt.Errorf("ping database legacy gagal: %w", err)
	}
	return db, func() { _ = sqlDB.Close() }, nil
}

func legacyDSN() string {
	endpoint := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(config.GetEnv("LEGACY_DB_HOST", "127.0.0.1"), config.GetEnv("LEGACY_DB_PORT", "5432")),
		Path:   config.GetEnv("LEGACY_DB_NAME", "ebitda"),
	}
	username := config.GetEnv("LEGACY_DB_USER", "shiddiq")
	if password := config.GetEnv("LEGACY_DB_PASSWORD", ""); password != "" {
		endpoint.User = url.UserPassword(username, password)
	} else {
		endpoint.User = url.User(username)
	}
	query := endpoint.Query()
	query.Set("sslmode", config.GetEnv("LEGACY_DB_SSLMODE", "disable"))
	endpoint.RawQuery = query.Encode()
	return endpoint.String()
}

func loadSourceState(source *gorm.DB, limit int) (*sourceState, error) {
	state := &sourceState{ProvinceRows: map[string]int{}}

	if err := source.Where("domain = ? AND slug = ?", models.RoleDomainKdkmp, models.RoleSlugManager).First(&state.ManagerRole).Error; err != nil {
		return nil, fmt.Errorf("memuat role Manager KDKMP legacy: %w", err)
	}

	managerQuery := source.
		Select("id, role_id, name, username, email, email_verified_at, password, sdm_kdkmp_entry_id, has_completed_onboarding, created_at, updated_at").
		Where("role_id = ?", state.ManagerRole.ID).
		Order("id")
	if limit > 0 {
		managerQuery = managerQuery.Limit(limit)
	}
	if err := managerQuery.Find(&state.Managers).Error; err != nil {
		return nil, fmt.Errorf("memuat akun Manager legacy: %w", err)
	}

	entryQuery := source.Order("id")
	if limit > 0 {
		entryQuery = entryQuery.Limit(limit)
	}
	if err := entryQuery.Find(&state.Entries).Error; err != nil {
		return nil, fmt.Errorf("memuat entry KDKMP legacy: %w", err)
	}

	var provinceRows []struct {
		Provinsi string `gorm:"column:provinsi"`
		Total    int    `gorm:"column:total"`
	}
	if err := source.
		Table("sdm_kdkmp_entries").
		Select("provinsi, count(*) AS total").
		Where("provinsi IS NOT NULL AND provinsi <> ''").
		Group("provinsi").
		Scan(&provinceRows).Error; err != nil {
		return nil, fmt.Errorf("memuat sebaran provinsi legacy: %w", err)
	}
	for _, row := range provinceRows {
		state.ProvinceRows[row.Provinsi] = row.Total
	}

	return state, nil
}

func loadTargetState(target *gorm.DB) (*targetState, error) {
	state := &targetState{
		EntryNIKs:    map[string]bool{},
		UsersByEmail: map[string]models.User{},
		Usernames:    map[string]string{},
	}

	var entries []legacyEntry
	if err := target.Select("id, nik").Where("nik IS NOT NULL AND nik <> ''").Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("memuat entry target: %w", err)
	}
	for _, entry := range entries {
		state.EntryNIKs[*entry.NIK] = true
	}

	var users []models.User
	if err := target.Preload("Role").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("memuat akun target: %w", err)
	}
	for _, user := range users {
		email := normalizeEmail(user.Email)
		state.UsersByEmail[email] = user
		if user.Username != nil {
			state.Usernames[*user.Username] = email
		}
	}

	return state, nil
}

func buildPlan(state *sourceState, target *targetState, options cloneOptions) *clonePlan {
	plan := &clonePlan{}

	for _, entry := range state.Entries {
		if entry.NIK == nil || *entry.NIK == "" {
			plan.EntriesSkipped++
			continue
		}
		if target.EntryNIKs[*entry.NIK] {
			plan.EntriesSkipped++
			continue
		}
		plan.Entries = append(plan.Entries, entry)
	}

	takenUsernames := make(map[string]string, len(target.Usernames))
	for username, email := range target.Usernames {
		takenUsernames[username] = email
	}
	for _, manager := range state.Managers {
		email := normalizeEmail(manager.Email)
		if email == "" {
			plan.ManagersSkipped++
			continue
		}

		entryNIK := ""
		if manager.SDMKdkmpEntryID != nil {
			entryNIK = state.entryNIK(*manager.SDMKdkmpEntryID)
		}

		if existing, ok := target.UsersByEmail[email]; ok {
			username := ""
			if existing.Username != nil {
				username = *existing.Username
			}
			plan.Managers = append(plan.Managers, managerPlan{Legacy: manager, ExistingID: existing.ID, EntryNIK: entryNIK, Username: username})
			continue
		}

		username := uniqueUsername(slugify(deref(manager.Username)), takenUsernames)
		takenUsernames[username] = email
		plan.Managers = append(plan.Managers, managerPlan{Legacy: manager, EntryNIK: entryNIK, Username: username})
	}

	existingEmails := make(map[string]bool, len(target.UsersByEmail))
	for email := range target.UsersByEmail {
		existingEmails[email] = true
	}

	plan.Reset = resetTargets(plan.Managers, options.ResetManagerPasswords)
	plan.Regional = planRegionalManagers(state.ProvinceRows, options.RegionalManagers, existingEmails, takenUsernames)

	return plan
}

func (state *sourceState) entryNIK(entryID int64) string {
	for _, entry := range state.Entries {
		if entry.ID == entryID && entry.NIK != nil {
			return *entry.NIK
		}
	}
	return ""
}

// resetTargets memilih N manager pertama (urut ID legacy) yang punya entry.
func resetTargets(managers []managerPlan, count int) []managerPlan {
	if count <= 0 {
		return nil
	}
	targets := make([]managerPlan, 0, count)
	for _, manager := range managers {
		if manager.EntryNIK == "" {
			continue
		}
		targets = append(targets, manager)
		if len(targets) == count {
			break
		}
	}
	return targets
}

// planRegionalManagers membuat akun manager wilayah demo per provinsi,
// diurutkan dari provinsi dengan KDKMP terbanyak.
func planRegionalManagers(provinceRows map[string]int, count int, existingEmails map[string]bool, takenUsernames map[string]string) []regionalPlan {
	if count <= 0 {
		return nil
	}

	provinces := make([]string, 0, len(provinceRows))
	for provinsi := range provinceRows {
		provinces = append(provinces, provinsi)
	}
	sort.Slice(provinces, func(i, j int) bool {
		if provinceRows[provinces[i]] != provinceRows[provinces[j]] {
			return provinceRows[provinces[i]] > provinceRows[provinces[j]]
		}
		return provinces[i] < provinces[j]
	})

	plans := make([]regionalPlan, 0, count)
	for _, provinsi := range provinces {
		email := fmt.Sprintf("%s-%s@%s", regionalUsername, slugify(provinsi), qaEmailDomain)
		if existingEmails[email] {
			continue
		}
		username := uniqueUsername(slugify(regionalUsername+"-"+provinsi), takenUsernames)
		takenUsernames[username] = email
		plans = append(plans, regionalPlan{
			Provinsi: provinsi,
			Email:    email,
			Name:     "Manager Wilayah " + provinsi,
			Username: username,
		})
		if len(plans) == count {
			break
		}
	}

	return plans
}

func printPlan(plan *clonePlan, options cloneOptions) {
	fmt.Printf("rencana clone data organisasi (limit=%d, apply=%t)\n", options.Limit, options.Apply)
	fmt.Printf("  entry baru      : %d (dilewati: %d)\n", len(plan.Entries), plan.EntriesSkipped)
	inserted, linked := 0, 0
	for _, manager := range plan.Managers {
		if manager.ExistingID == 0 {
			inserted++
		} else {
			linked++
		}
	}
	fmt.Printf("  manager baru    : %d (sudah ada: %d)\n", inserted, linked)
	fmt.Printf("  manager wilayah: %d\n", len(plan.Regional))
	fmt.Printf("  reset password  : %d\n", len(plan.Reset))
	for index, manager := range plan.Entries {
		if index == 3 {
			fmt.Println("  ...")
			break
		}
		fmt.Printf("    + entry %s | %s | %s\n", deref(manager.NIK), manager.NamaKoperasi, deref(manager.Provinsi))
	}
	for index, manager := range plan.Managers {
		if index == 3 {
			fmt.Println("  ...")
			break
		}
		fmt.Printf("    + manager %s | %s | entry=%s\n", manager.Legacy.Email, manager.Legacy.Name, manager.EntryNIK)
	}
	for _, regional := range plan.Regional {
		fmt.Printf("    + %s | %s\n", regional.Email, regional.Provinsi)
	}
	for _, manager := range plan.Reset {
		fmt.Printf("    ~ reset password: %s\n", manager.Legacy.Email)
	}
}

func applyPlan(target *gorm.DB, state *sourceState, plan *clonePlan, options cloneOptions) error {
	return target.Transaction(func(tx *gorm.DB) error {
		managerRole, err := ensureRole(tx, models.RoleDomainKdkmp, models.RoleSlugManager, "Kepala Toko / Manager", models.RoleLevelManager)
		if err != nil {
			return err
		}
		regionalRole, err := ensureRole(tx, models.RoleDomainKdkmp, models.RoleSlugRegionalManager, "Manager Wilayah", models.RoleLevelManager)
		if err != nil {
			return err
		}

		if err := insertEntries(tx, plan.Entries); err != nil {
			return err
		}

		entryIDsByNIK, err := loadEntryIDsByNIK(tx)
		if err != nil {
			return err
		}

		if err := insertManagers(tx, plan.Managers, managerRole.ID, entryIDsByNIK); err != nil {
			return err
		}
		if err := linkExistingManagers(tx, plan.Managers, managerRole.ID, entryIDsByNIK); err != nil {
			return err
		}
		if err := resetManagerPasswords(tx, plan.Reset); err != nil {
			return err
		}
		if err := insertRegionalManagers(tx, plan.Regional, regionalRole.ID); err != nil {
			return err
		}

		return nil
	})
}

func ensureRole(tx *gorm.DB, domain, slug, name, level string) (*models.Role, error) {
	var role models.Role
	err := tx.Where("domain = ? AND slug = ?", domain, slug).First(&role).Error
	if err == nil {
		return &role, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("memuat role %s: %w", slug, err)
	}

	role = models.Role{UUID: uuid.NewString(), Name: name, Slug: slug, Level: level, Domain: domain}
	if err := tx.Create(&role).Error; err != nil {
		return nil, fmt.Errorf("membuat role %s: %w", slug, err)
	}
	return &role, nil
}

func insertEntries(tx *gorm.DB, entries []legacyEntry) error {
	if len(entries) == 0 {
		return nil
	}
	prepared := prepareEntryInserts(entries)
	if err := tx.CreateInBatches(&prepared, insertBatchSize).Error; err != nil {
		return fmt.Errorf("menyimpan entry KDKMP: %w", err)
	}
	return nil
}

// prepareEntryInserts menyalin entry tanpa ID legacy agar target memakai
// sequence-nya sendiri (relasi tetap dipetakan via NIK).
func prepareEntryInserts(entries []legacyEntry) []legacyEntry {
	prepared := make([]legacyEntry, len(entries))
	for index, entry := range entries {
		entry.ID = 0
		prepared[index] = entry
	}
	return prepared
}

func loadEntryIDsByNIK(tx *gorm.DB) (map[string]int64, error) {
	var rows []struct {
		ID  int64  `gorm:"column:id"`
		NIK string `gorm:"column:nik"`
	}
	if err := tx.Table("sdm_kdkmp_entries").Select("id, nik").Where("nik IS NOT NULL AND nik <> ''").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("memuat ID entry target: %w", err)
	}
	result := make(map[string]int64, len(rows))
	for _, row := range rows {
		result[row.NIK] = row.ID
	}
	return result, nil
}

func insertManagers(tx *gorm.DB, managers []managerPlan, managerRoleID int64, entryIDsByNIK map[string]int64) error {
	users := make([]models.User, 0, len(managers))
	for _, manager := range managers {
		if manager.ExistingID != 0 {
			continue
		}

		user := manager.Legacy
		user.ID = 0
		user.RoleID = &managerRoleID
		user.Role = nil
		user.Email = normalizeEmail(manager.Legacy.Email)
		user.Username = stringPointer(manager.Username)
		user.SDMKdkmpEntryID = entryIDPointer(manager.EntryNIK, entryIDsByNIK)
		user.TwoFactorSecret = nil
		user.TwoFactorRecoveryCodes = nil
		user.TwoFactorConfirmedAt = nil
		user.ManagerSKDocument = nil
		user.RegionalAssignments = nil
		user.SDMKdkmpEntry = nil
		users = append(users, user)
	}
	if len(users) == 0 {
		return nil
	}
	if err := tx.CreateInBatches(&users, insertBatchSize).Error; err != nil {
		return fmt.Errorf("menyimpan akun manager: %w", err)
	}
	return nil
}

func linkExistingManagers(tx *gorm.DB, managers []managerPlan, managerRoleID int64, entryIDsByNIK map[string]int64) error {
	for _, manager := range managers {
		if manager.ExistingID == 0 {
			continue
		}
		entryID := entryIDPointer(manager.EntryNIK, entryIDsByNIK)
		if entryID == nil {
			continue
		}
		if err := tx.Model(&models.User{}).
			Where("id = ? AND sdm_kdkmp_entry_id IS NULL", manager.ExistingID).
			Updates(map[string]any{"sdm_kdkmp_entry_id": *entryID, "role_id": managerRoleID}).Error; err != nil {
			return fmt.Errorf("menghubungkan manager %s ke entry: %w", manager.Legacy.Email, err)
		}
	}
	return nil
}

func resetManagerPasswords(tx *gorm.DB, managers []managerPlan) error {
	if len(managers) == 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(qaPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("membuat hash password QA: %w", err)
	}
	for _, manager := range managers {
		if manager.ExistingID != 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", manager.ExistingID).Update("password", string(hash)).Error; err != nil {
				return fmt.Errorf("reset password %s: %w", manager.Legacy.Email, err)
			}
			continue
		}
		if err := tx.Model(&models.User{}).
			Where("LOWER(email) = ?", normalizeEmail(manager.Legacy.Email)).
			Update("password", string(hash)).Error; err != nil {
			return fmt.Errorf("reset password %s: %w", manager.Legacy.Email, err)
		}
	}
	return nil
}

func insertRegionalManagers(tx *gorm.DB, managers []regionalPlan, regionalRoleID int64) error {
	if len(managers) == 0 {
		return nil
	}
	now := time.Now()
	hash, err := bcrypt.GenerateFromPassword([]byte(qaPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("membuat hash password manager wilayah: %w", err)
	}

	for _, manager := range managers {
		user := models.User{
			RoleID:                 &regionalRoleID,
			Name:                   manager.Name,
			Username:               stringPointer(manager.Username),
			Email:                  manager.Email,
			EmailVerifiedAt:        &now,
			Password:               string(hash),
			HasCompletedOnboarding: true,
		}
		if err := tx.Create(&user).Error; err != nil {
			return fmt.Errorf("membuat akun manager wilayah %s: %w", manager.Email, err)
		}

		var existing int64
		if err := tx.Model(&models.UserRegionalAssignment{}).
			Where("user_id = ? AND scope_level = ? AND provinsi = ?", user.ID, models.RegionalScopeProvince, manager.Provinsi).
			Count(&existing).Error; err != nil {
			return fmt.Errorf("memeriksa assignment %s: %w", manager.Email, err)
		}
		if existing > 0 {
			continue
		}
		assignment := models.UserRegionalAssignment{
			UserID:     user.ID,
			ScopeLevel: models.RegionalScopeProvince,
			Provinsi:   manager.Provinsi,
		}
		if err := tx.Create(&assignment).Error; err != nil {
			return fmt.Errorf("membuat assignment %s: %w", manager.Email, err)
		}
	}
	return nil
}

func entryIDPointer(nik string, entryIDsByNIK map[string]int64) *int64 {
	if nik == "" {
		return nil
	}
	id, ok := entryIDsByNIK[nik]
	if !ok {
		return nil
	}
	return &id
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func stringPointer(value string) *string { return &value }

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func uniqueUsername(base string, taken map[string]string) string {
	if base == "" {
		base = "manager"
	}
	candidate := base
	for i := 2; ; i++ {
		if _, exists := taken[candidate]; !exists {
			return candidate
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
