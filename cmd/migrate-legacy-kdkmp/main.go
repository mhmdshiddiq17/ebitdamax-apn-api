// Command migrate-legacy-kdkmp prepares and applies a one-shot Manager KDKMP
// database migration. It never overwrites scoped target data.
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

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"agrinaspangan/ebitda-api/config"
	"agrinaspangan/ebitda-api/internal/database"
	"agrinaspangan/ebitda-api/internal/models"
)

var scopedTables = []string{
	"sdm_kdkmp_entries",
	"task_categories",
	"tasks",
	"task_additional_fields",
	"task_roles",
	"task_reports",
	"task_report_values",
	"ebitdamax_kdkmp",
	"meeting_minutes",
	"meeting_minute_items",
	"meeting_minute_item_status_histories",
	"meeting_minute_attachments",
}

type sdmEntry struct {
	ID             int64      `gorm:"primaryKey"`
	NamaKoperasi   string     `gorm:"column:nama_koperasi"`
	JumlahKaryawan int        `gorm:"column:jumlah_karyawan"`
	Catatan        *string    `gorm:"column:catatan"`
	CreatedBy      *int64     `gorm:"column:created_by"`
	UpdatedBy      *int64     `gorm:"column:updated_by"`
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

func (sdmEntry) TableName() string { return "sdm_kdkmp_entries" }

type taskRole struct {
	TaskID    int64     `gorm:"column:task_id"`
	RoleID    int64     `gorm:"column:role_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (taskRole) TableName() string { return "task_roles" }

type fileReferences struct {
	ManagerSK            int
	TaskReportPhotos     int
	TaskReportDocuments  int
	AdditionalFieldFiles int
	MeetingAttachments   int
}

func (f fileReferences) Total() int {
	return f.ManagerSK + f.TaskReportPhotos + f.TaskReportDocuments + f.AdditionalFieldFiles + f.MeetingAttachments
}

type migrationPlan struct {
	ManagerRole  models.Role
	Managers     []models.User
	Entries      []sdmEntry
	Categories   []models.TaskCategory
	Tasks        []models.Task
	Fields       []models.TaskAdditionalField
	Reports      []models.TaskReport
	Values       []models.TaskReportValue
	DailyEntries []models.EbitdamaxKdkmp
	Meetings     []models.MeetingMinute
	Items        []models.MeetingMinuteItem
	Histories    []models.MeetingMinuteItemStatusHistory
	Attachments  []models.MeetingMinuteAttachment
	Files        fileReferences
}

func main() {
	apply := flag.Bool("apply", false, "tulis data ke target yang masih kosong")
	flag.Parse()

	config.LoadEnv()
	source, closeSource, err := connectLegacy()
	if err != nil {
		log.Fatal(err)
	}
	defer closeSource()

	plan, err := loadPlan(source)
	if err != nil {
		log.Fatal(err)
	}
	printPlan(plan)
	if !*apply {
		return
	}
	if plan.Files.Total() != 0 {
		log.Fatal("apply diblokir: salin seluruh objek SK, bukti task, dan lampiran meeting terlebih dahulu; metadata tanpa objek tidak boleh dipindahkan")
	}

	target := database.Connect()
	if err := applyPlan(target, plan); err != nil {
		log.Fatal(err)
	}
	log.Println("migrasi Manager KDKMP selesai")
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

func loadPlan(source *gorm.DB) (*migrationPlan, error) {
	plan := &migrationPlan{}
	if err := source.Where("domain = ? AND slug = ?", models.RoleDomainKdkmp, models.RoleSlugManager).First(&plan.ManagerRole).Error; err != nil {
		return nil, fmt.Errorf("memuat role Manager KDKMP: %w", err)
	}
	if err := source.Where("role_id = ?", plan.ManagerRole.ID).Order("id").Find(&plan.Managers).Error; err != nil {
		return nil, fmt.Errorf("memuat akun Manager KDKMP: %w", err)
	}
	if len(plan.Managers) == 0 {
		return nil, errors.New("tidak ada akun Manager KDKMP di legacy")
	}

	managerIDs := userIDs(plan.Managers)
	entryIDs := make([]int64, 0, len(plan.Managers))
	for _, manager := range plan.Managers {
		if manager.SDMKdkmpEntryID != nil {
			entryIDs = append(entryIDs, *manager.SDMKdkmpEntryID)
		}
	}
	if len(entryIDs) > 0 {
		if err := source.Where("id IN ?", uniqueIDs(entryIDs)).Order("id").Find(&plan.Entries).Error; err != nil {
			return nil, fmt.Errorf("memuat data KDKMP: %w", err)
		}
	}

	if err := source.Model(&models.Task{}).
		Distinct("tasks.*").
		Joins("JOIN task_roles ON task_roles.task_id = tasks.id").
		Where("task_roles.role_id = ?", plan.ManagerRole.ID).
		Order("tasks.sort_order NULLS LAST, tasks.id").
		Find(&plan.Tasks).Error; err != nil {
		return nil, fmt.Errorf("memuat task Manager KDKMP: %w", err)
	}
	taskIDs := taskIDs(plan.Tasks)
	if len(taskIDs) > 0 {
		categoryIDs := make([]int64, 0, len(plan.Tasks))
		for _, task := range plan.Tasks {
			categoryIDs = append(categoryIDs, task.TaskCategoryID)
		}
		if err := source.Where("id IN ?", uniqueIDs(categoryIDs)).Order("id").Find(&plan.Categories).Error; err != nil {
			return nil, fmt.Errorf("memuat kategori task: %w", err)
		}
		if err := source.Where("task_id IN ?", taskIDs).Order("task_id, sort_order, id").Find(&plan.Fields).Error; err != nil {
			return nil, fmt.Errorf("memuat field task: %w", err)
		}
		if err := source.Where("user_id IN ? AND task_id IN ?", managerIDs, taskIDs).Order("id").Find(&plan.Reports).Error; err != nil {
			return nil, fmt.Errorf("memuat laporan task: %w", err)
		}
	}
	reportIDs := reportIDs(plan.Reports)
	if len(reportIDs) > 0 {
		if err := source.Where("task_report_id IN ?", reportIDs).Order("id").Find(&plan.Values).Error; err != nil {
			return nil, fmt.Errorf("memuat nilai laporan task: %w", err)
		}
	}
	if len(entryIDs) > 0 {
		if err := source.Where("sdm_kdkmp_entry_id IN ?", uniqueIDs(entryIDs)).Order("id").Find(&plan.DailyEntries).Error; err != nil {
			return nil, fmt.Errorf("memuat dashboard harian: %w", err)
		}
	}
	if err := source.Where("created_by IN ?", managerIDs).Order("id").Find(&plan.Meetings).Error; err != nil {
		return nil, fmt.Errorf("memuat meeting minutes: %w", err)
	}
	meetingIDs := meetingIDs(plan.Meetings)
	if len(meetingIDs) > 0 {
		if err := source.Where("meeting_minute_id IN ?", meetingIDs).Order("meeting_minute_id, sort_order, id").Find(&plan.Items).Error; err != nil {
			return nil, fmt.Errorf("memuat item meeting: %w", err)
		}
		if err := source.Where("meeting_minute_id IN ?", meetingIDs).Order("id").Find(&plan.Attachments).Error; err != nil {
			return nil, fmt.Errorf("memuat lampiran meeting: %w", err)
		}
	}
	itemIDs := meetingItemIDs(plan.Items)
	if len(itemIDs) > 0 {
		if err := source.Where("meeting_minute_item_id IN ?", itemIDs).Order("id").Find(&plan.Histories).Error; err != nil {
			return nil, fmt.Errorf("memuat riwayat action item: %w", err)
		}
	}
	plan.Files = findFileReferences(plan)
	return plan, nil
}

func findFileReferences(plan *migrationPlan) fileReferences {
	refs := fileReferences{MeetingAttachments: len(plan.Attachments)}
	for _, manager := range plan.Managers {
		if manager.ManagerSKDocument != nil && strings.TrimSpace(*manager.ManagerSKDocument) != "" {
			refs.ManagerSK++
		}
	}
	for _, report := range plan.Reports {
		if report.StartedPhoto != nil && *report.StartedPhoto != "" {
			refs.TaskReportPhotos++
		}
		if report.FinishedPhoto != nil && *report.FinishedPhoto != "" {
			refs.TaskReportPhotos++
		}
		refs.TaskReportDocuments += len(report.StartedDocuments) + len(report.FinishedDocuments)
	}
	fields := make(map[int64]models.TaskAdditionalField, len(plan.Fields))
	for _, field := range plan.Fields {
		fields[field.ID] = field
	}
	for _, value := range plan.Values {
		if field, ok := fields[value.TaskAdditionalFieldID]; ok && field.InputType == "file" && value.Value != nil && *value.Value != "" {
			refs.AdditionalFieldFiles++
		}
	}
	return refs
}

func applyPlan(target *gorm.DB, plan *migrationPlan) error {
	if err := verifyTargetReady(target); err != nil {
		return err
	}
	return target.Transaction(func(tx *gorm.DB) error {
		managerRole, err := ensureManagerRole(tx, plan.ManagerRole)
		if err != nil {
			return err
		}
		userMap, err := copyManagers(tx, plan.Managers, managerRole.ID)
		if err != nil {
			return err
		}
		entryMap, err := copyEntries(tx, plan.Entries, userMap)
		if err != nil {
			return err
		}
		if err := attachManagersToEntries(tx, plan.Managers, userMap, entryMap); err != nil {
			return err
		}
		categoryMap, err := copyCategories(tx, plan.Categories)
		if err != nil {
			return err
		}
		taskMap, err := copyTasks(tx, plan.Tasks, categoryMap, managerRole.ID)
		if err != nil {
			return err
		}
		fieldMap, err := copyFields(tx, plan.Fields, taskMap)
		if err != nil {
			return err
		}
		reportMap, err := copyReports(tx, plan.Reports, taskMap, userMap)
		if err != nil {
			return err
		}
		if err := copyReportValues(tx, plan.Values, reportMap, fieldMap); err != nil {
			return err
		}
		if err := copyDailyEntries(tx, plan.DailyEntries, entryMap, taskMap, userMap); err != nil {
			return err
		}
		meetingMap, err := copyMeetings(tx, plan.Meetings, userMap)
		if err != nil {
			return err
		}
		itemMap, err := copyMeetingItems(tx, plan.Items, meetingMap)
		if err != nil {
			return err
		}
		if err := copyItemHistories(tx, plan.Histories, itemMap, userMap); err != nil {
			return err
		}
		return verifyTargetCounts(tx, plan)
	})
}

func verifyTargetReady(target *gorm.DB) error {
	for _, table := range scopedTables {
		var count int64
		if err := target.Table(table).Count(&count).Error; err != nil {
			return fmt.Errorf("memeriksa target %s: %w", table, err)
		}
		if count != 0 {
			return fmt.Errorf("target %s sudah berisi data; migrasi tidak akan menimpa data", table)
		}
	}
	return nil
}

func ensureManagerRole(tx *gorm.DB, source models.Role) (*models.Role, error) {
	var target models.Role
	err := tx.Where("domain = ? AND slug = ?", models.RoleDomainKdkmp, models.RoleSlugManager).First(&target).Error
	if err == nil {
		return &target, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	source.ID = 0
	if err := tx.Create(&source).Error; err != nil {
		return nil, fmt.Errorf("membuat role Manager KDKMP: %w", err)
	}
	return &source, nil
}

func copyManagers(tx *gorm.DB, source []models.User, managerRoleID int64) (map[int64]int64, error) {
	var targetUsers []models.User
	if err := tx.Preload("Role").Find(&targetUsers).Error; err != nil {
		return nil, fmt.Errorf("memuat akun target: %w", err)
	}
	byEmail := make(map[string]models.User, len(targetUsers))
	byUsername := make(map[string]models.User, len(targetUsers))
	for _, user := range targetUsers {
		byEmail[strings.ToLower(user.Email)] = user
		if user.Username != nil {
			byUsername[*user.Username] = user
		}
	}

	mapped := make(map[int64]int64, len(source))
	for _, sourceUser := range source {
		email := strings.ToLower(strings.TrimSpace(sourceUser.Email))
		if existing, ok := byEmail[email]; ok {
			if existing.Role == nil || existing.Role.Domain != models.RoleDomainKdkmp || existing.Role.Slug != models.RoleSlugManager {
				return nil, fmt.Errorf("email target %s bukan akun Manager KDKMP", email)
			}
			if err := updateManager(tx, existing.ID, sourceUser, managerRoleID); err != nil {
				return nil, err
			}
			mapped[sourceUser.ID] = existing.ID
			continue
		}
		if sourceUser.Username != nil {
			if existing, ok := byUsername[*sourceUser.Username]; ok && strings.ToLower(existing.Email) != email {
				return nil, fmt.Errorf("username %s sudah dipakai target oleh email lain", *sourceUser.Username)
			}
		}

		user := sourceUser
		user.ID = 0
		user.RoleID = &managerRoleID
		user.SDMKdkmpEntryID = nil
		user.TwoFactorSecret = nil
		user.TwoFactorRecoveryCodes = nil
		user.TwoFactorConfirmedAt = nil
		user.ManagerSKDocument = nil
		user.Role = nil
		user.RegionalAssignments = nil
		user.SDMKdkmpEntry = nil
		if err := tx.Create(&user).Error; err != nil {
			return nil, fmt.Errorf("membuat akun Manager %s: %w", email, err)
		}
		mapped[sourceUser.ID] = user.ID
	}
	return mapped, nil
}

func updateManager(tx *gorm.DB, targetID int64, source models.User, managerRoleID int64) error {
	return tx.Model(&models.User{}).Where("id = ?", targetID).Updates(map[string]any{
		"role_id":                   managerRoleID,
		"name":                      source.Name,
		"username":                  source.Username,
		"email_verified_at":         source.EmailVerifiedAt,
		"password":                  source.Password,
		"sdm_kdkmp_entry_id":        nil,
		"two_factor_secret":         nil,
		"two_factor_recovery_codes": nil,
		"two_factor_confirmed_at":   nil,
		"manager_sk_document":       nil,
	}).Error
}

func copyEntries(tx *gorm.DB, source []sdmEntry, userMap map[int64]int64) (map[int64]int64, error) {
	mapped := make(map[int64]int64, len(source))
	for _, sourceEntry := range source {
		entry := sourceEntry
		entry.ID = 0
		entry.CreatedBy = remapUserID(sourceEntry.CreatedBy, userMap)
		entry.UpdatedBy = remapUserID(sourceEntry.UpdatedBy, userMap)
		if err := tx.Create(&entry).Error; err != nil {
			return nil, fmt.Errorf("membuat data KDKMP %d: %w", sourceEntry.ID, err)
		}
		mapped[sourceEntry.ID] = entry.ID
	}
	return mapped, nil
}

func attachManagersToEntries(tx *gorm.DB, source []models.User, userMap, entryMap map[int64]int64) error {
	for _, manager := range source {
		if manager.SDMKdkmpEntryID == nil {
			continue
		}
		entryID, ok := entryMap[*manager.SDMKdkmpEntryID]
		if !ok {
			return fmt.Errorf("data KDKMP %d untuk Manager %d tidak ditemukan", *manager.SDMKdkmpEntryID, manager.ID)
		}
		if err := tx.Model(&models.User{}).Where("id = ?", userMap[manager.ID]).Update("sdm_kdkmp_entry_id", entryID).Error; err != nil {
			return fmt.Errorf("menghubungkan Manager %d ke KDKMP: %w", manager.ID, err)
		}
	}
	return nil
}

func copyCategories(tx *gorm.DB, source []models.TaskCategory) (map[int64]int64, error) {
	mapped := make(map[int64]int64, len(source))
	for _, sourceCategory := range source {
		category := sourceCategory
		category.ID = 0
		if err := tx.Create(&category).Error; err != nil {
			return nil, fmt.Errorf("membuat kategori %s: %w", sourceCategory.Slug, err)
		}
		mapped[sourceCategory.ID] = category.ID
	}
	return mapped, nil
}

func copyTasks(tx *gorm.DB, source []models.Task, categoryMap map[int64]int64, managerRoleID int64) (map[int64]int64, error) {
	mapped := make(map[int64]int64, len(source))
	for _, sourceTask := range source {
		categoryID, ok := categoryMap[sourceTask.TaskCategoryID]
		if !ok {
			return nil, fmt.Errorf("kategori task %d tidak ditemukan", sourceTask.ID)
		}
		task := sourceTask
		task.ID = 0
		task.TaskCategoryID = categoryID
		task.Roles = nil
		task.AdditionalFields = nil
		if err := tx.Create(&task).Error; err != nil {
			return nil, fmt.Errorf("membuat task %s: %w", sourceTask.Name, err)
		}
		now := time.Now()
		if err := tx.Create(&taskRole{TaskID: task.ID, RoleID: managerRoleID, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			return nil, fmt.Errorf("menghubungkan role ke task %s: %w", sourceTask.Name, err)
		}
		mapped[sourceTask.ID] = task.ID
	}
	return mapped, nil
}

func copyFields(tx *gorm.DB, source []models.TaskAdditionalField, taskMap map[int64]int64) (map[int64]int64, error) {
	mapped := make(map[int64]int64, len(source))
	for _, sourceField := range source {
		taskID, ok := taskMap[sourceField.TaskID]
		if !ok {
			return nil, fmt.Errorf("task untuk field %d tidak ditemukan", sourceField.ID)
		}
		field := sourceField
		field.ID = 0
		field.TaskID = taskID
		if err := tx.Create(&field).Error; err != nil {
			return nil, fmt.Errorf("membuat field %s: %w", sourceField.FieldName, err)
		}
		mapped[sourceField.ID] = field.ID
	}
	return mapped, nil
}

func copyReports(tx *gorm.DB, source []models.TaskReport, taskMap, userMap map[int64]int64) (map[int64]int64, error) {
	mapped := make(map[int64]int64, len(source))
	for _, sourceReport := range source {
		taskID, taskOK := taskMap[sourceReport.TaskID]
		userID, userOK := userMap[sourceReport.UserID]
		if !taskOK || !userOK {
			return nil, fmt.Errorf("relasi laporan task %d tidak ditemukan", sourceReport.ID)
		}
		report := sourceReport
		report.ID = 0
		report.TaskID = taskID
		report.UserID = userID
		report.Task = nil
		report.User = nil
		if err := tx.Create(&report).Error; err != nil {
			return nil, fmt.Errorf("membuat laporan task %d: %w", sourceReport.ID, err)
		}
		mapped[sourceReport.ID] = report.ID
	}
	return mapped, nil
}

func copyReportValues(tx *gorm.DB, source []models.TaskReportValue, reportMap, fieldMap map[int64]int64) error {
	for _, sourceValue := range source {
		reportID, reportOK := reportMap[sourceValue.TaskReportID]
		fieldID, fieldOK := fieldMap[sourceValue.TaskAdditionalFieldID]
		if !reportOK || !fieldOK {
			return fmt.Errorf("relasi nilai laporan %d tidak ditemukan", sourceValue.ID)
		}
		value := sourceValue
		value.ID = 0
		value.TaskReportID = reportID
		value.TaskAdditionalFieldID = fieldID
		if err := tx.Create(&value).Error; err != nil {
			return fmt.Errorf("membuat nilai laporan %d: %w", sourceValue.ID, err)
		}
	}
	return nil
}

func copyDailyEntries(tx *gorm.DB, source []models.EbitdamaxKdkmp, entryMap, taskMap, userMap map[int64]int64) error {
	for _, sourceEntry := range source {
		entryID, ok := entryMap[sourceEntry.SDMKdkmpEntryID]
		if !ok {
			return fmt.Errorf("KDKMP dashboard %d tidak memiliki relasi entry", sourceEntry.ID)
		}
		selected, err := remapTaskIDs(sourceEntry.SelectedTaskIDs, taskMap)
		if err != nil {
			return fmt.Errorf("pilihan task dashboard %d: %w", sourceEntry.ID, err)
		}
		entry := sourceEntry
		entry.ID = 0
		entry.SDMKdkmpEntryID = entryID
		entry.SelectedTaskIDs = selected
		entry.CreatedBy = remapUserID(sourceEntry.CreatedBy, userMap)
		entry.UpdatedBy = remapUserID(sourceEntry.UpdatedBy, userMap)
		if err := tx.Create(&entry).Error; err != nil {
			return fmt.Errorf("membuat dashboard harian %d: %w", sourceEntry.ID, err)
		}
	}
	return nil
}

func copyMeetings(tx *gorm.DB, source []models.MeetingMinute, userMap map[int64]int64) (map[int64]int64, error) {
	mapped := make(map[int64]int64, len(source))
	for _, sourceMeeting := range source {
		meeting := sourceMeeting
		meeting.ID = 0
		meeting.CreatedBy = remapUserID(sourceMeeting.CreatedBy, userMap)
		meeting.UpdatedBy = remapUserID(sourceMeeting.UpdatedBy, userMap)
		meeting.Items = nil
		meeting.Attachments = nil
		if err := tx.Create(&meeting).Error; err != nil {
			return nil, fmt.Errorf("membuat meeting %d: %w", sourceMeeting.ID, err)
		}
		mapped[sourceMeeting.ID] = meeting.ID
	}
	return mapped, nil
}

func copyMeetingItems(tx *gorm.DB, source []models.MeetingMinuteItem, meetingMap map[int64]int64) (map[int64]int64, error) {
	mapped := make(map[int64]int64, len(source))
	for _, sourceItem := range source {
		meetingID, ok := meetingMap[sourceItem.MeetingMinuteID]
		if !ok {
			return nil, fmt.Errorf("meeting untuk item %d tidak ditemukan", sourceItem.ID)
		}
		item := sourceItem
		item.ID = 0
		item.MeetingMinuteID = meetingID
		item.MeetingMinute = nil
		item.StatusHistories = nil
		if err := tx.Create(&item).Error; err != nil {
			return nil, fmt.Errorf("membuat item meeting %d: %w", sourceItem.ID, err)
		}
		mapped[sourceItem.ID] = item.ID
	}
	return mapped, nil
}

func copyItemHistories(tx *gorm.DB, source []models.MeetingMinuteItemStatusHistory, itemMap, userMap map[int64]int64) error {
	for _, sourceHistory := range source {
		itemID, ok := itemMap[sourceHistory.MeetingMinuteItemID]
		if !ok {
			return fmt.Errorf("item untuk riwayat %d tidak ditemukan", sourceHistory.ID)
		}
		history := sourceHistory
		history.ID = 0
		history.MeetingMinuteItemID = itemID
		history.ChangedBy = remapUserID(sourceHistory.ChangedBy, userMap)
		history.ChangedByUser = nil
		if err := tx.Create(&history).Error; err != nil {
			return fmt.Errorf("membuat riwayat action item %d: %w", sourceHistory.ID, err)
		}
	}
	return nil
}

func verifyTargetCounts(tx *gorm.DB, plan *migrationPlan) error {
	expected := map[string]int{
		"sdm_kdkmp_entries":                    len(plan.Entries),
		"task_categories":                      len(plan.Categories),
		"tasks":                                len(plan.Tasks),
		"task_additional_fields":               len(plan.Fields),
		"task_roles":                           len(plan.Tasks),
		"task_reports":                         len(plan.Reports),
		"task_report_values":                   len(plan.Values),
		"ebitdamax_kdkmp":                      len(plan.DailyEntries),
		"meeting_minutes":                      len(plan.Meetings),
		"meeting_minute_items":                 len(plan.Items),
		"meeting_minute_item_status_histories": len(plan.Histories),
		"meeting_minute_attachments":           0,
	}
	for table, want := range expected {
		var got int64
		if err := tx.Table(table).Count(&got).Error; err != nil {
			return err
		}
		if got != int64(want) {
			return fmt.Errorf("verifikasi %s = %d, ingin %d", table, got, want)
		}
	}
	return nil
}

func remapUserID(source *int64, mapped map[int64]int64) *int64 {
	if source == nil {
		return nil
	}
	target, ok := mapped[*source]
	if !ok {
		return nil
	}
	return &target
}

func remapTaskIDs(source models.IntList, mapped map[int64]int64) (models.IntList, error) {
	result := make(models.IntList, 0, len(source))
	for _, sourceID := range source {
		targetID, ok := mapped[sourceID]
		if !ok {
			return nil, fmt.Errorf("task source %d di luar scope Manager KDKMP", sourceID)
		}
		result = append(result, targetID)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func userIDs(users []models.User) []int64 {
	ids := make([]int64, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	return ids
}

func taskIDs(tasks []models.Task) []int64 {
	ids := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	return ids
}

func reportIDs(reports []models.TaskReport) []int64 {
	ids := make([]int64, 0, len(reports))
	for _, report := range reports {
		ids = append(ids, report.ID)
	}
	return ids
}

func meetingIDs(meetings []models.MeetingMinute) []int64 {
	ids := make([]int64, 0, len(meetings))
	for _, meeting := range meetings {
		ids = append(ids, meeting.ID)
	}
	return ids
}

func meetingItemIDs(items []models.MeetingMinuteItem) []int64 {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func uniqueIDs(values []int64) []int64 {
	seen := make(map[int64]bool, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func printPlan(plan *migrationPlan) {
	fmt.Printf("Manager=%d KDKMP=%d kategori=%d task=%d field=%d laporan=%d nilai=%d dashboard=%d meeting=%d item=%d riwayat=%d\n",
		len(plan.Managers), len(plan.Entries), len(plan.Categories), len(plan.Tasks), len(plan.Fields), len(plan.Reports), len(plan.Values), len(plan.DailyEntries), len(plan.Meetings), len(plan.Items), len(plan.Histories))
	fmt.Printf("referensi file: SK=%d foto=%d dokumen=%d field-file=%d lampiran-meeting=%d\n",
		plan.Files.ManagerSK, plan.Files.TaskReportPhotos, plan.Files.TaskReportDocuments, plan.Files.AdditionalFieldFiles, plan.Files.MeetingAttachments)
}
