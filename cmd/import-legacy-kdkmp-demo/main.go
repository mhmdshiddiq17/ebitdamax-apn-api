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
	"agrinaspangan/ebitda-api/internal/kdkmp"
	"agrinaspangan/ebitda-api/internal/models"
)

const demoNIK = "DEMO-KDKMP-LEGACY"

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

type importPlan struct {
	SourceEntryID int64
	SourceEntry   sdmEntry
	Categories    []models.TaskCategory
	Tasks         []models.Task
	Fields        []models.TaskAdditionalField
	Reports       []models.TaskReport
	ReportValues  []models.TaskReportValue
	DailyEntries  []models.EbitdamaxKdkmp
	SourceLastDay time.Time
	TargetDay     time.Time
	ShiftDays     int
}

func main() {
	apply := flag.Bool("apply", false, "tulis fixture demo ke database refactor")
	sourceEntryID := flag.Int64("source-sdm-entry-id", 64, "ID SDM KDKMP sumber legacy")
	flag.Parse()

	config.LoadEnv()
	target := database.Connect()
	source, closeSource, err := connectSource()
	if err != nil {
		log.Fatal(err)
	}
	defer closeSource()

	plan, err := loadPlan(source, *sourceEntryID, kdkmp.BusinessDate())
	if err != nil {
		log.Fatal(err)
	}
	manager, err := preflightTarget(target)
	if err != nil {
		log.Fatal(err)
	}

	printPlan(plan, *apply)
	if !*apply {
		return
	}

	if err := applyPlan(target, manager, plan); err != nil {
		log.Fatal(err)
	}
	log.Println("fixture demo legacy berhasil diimpor")
}

func connectSource() (*gorm.DB, func(), error) {
	dsn := legacyDSN()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, nil, fmt.Errorf("koneksi database legacy gagal: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("akses koneksi database legacy gagal: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, nil, fmt.Errorf("ping database legacy gagal: %w", err)
	}
	return db, func() { _ = sqlDB.Close() }, nil
}

func legacyDSN() string {
	username := legacyEnv("LEGACY_DB_USER", "shiddiq")
	password := config.GetEnv("LEGACY_DB_PASSWORD", "")
	endpoint := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(legacyEnv("LEGACY_DB_HOST", "127.0.0.1"), legacyEnv("LEGACY_DB_PORT", "5432")),
		Path:   legacyEnv("LEGACY_DB_NAME", "ebitda"),
	}
	if password == "" {
		endpoint.User = url.User(username)
	} else {
		endpoint.User = url.UserPassword(username, password)
	}
	query := endpoint.Query()
	query.Set("sslmode", legacyEnv("LEGACY_DB_SSLMODE", "disable"))
	endpoint.RawQuery = query.Encode()
	return endpoint.String()
}

func legacyEnv(key, fallback string) string {
	if value := strings.TrimSpace(config.GetEnv(key, "")); value != "" {
		return value
	}
	return fallback
}

func loadPlan(source *gorm.DB, sourceEntryID int64, targetDay time.Time) (*importPlan, error) {
	var user models.User
	err := source.Preload("Role").
		Where("sdm_kdkmp_entry_id = ?", sourceEntryID).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("SDM KDKMP source %d tidak memiliki akun manager", sourceEntryID)
	}
	if err != nil {
		return nil, fmt.Errorf("memuat akun manager source: %w", err)
	}
	if !user.IsKdkmpManager() || user.RoleID == nil {
		return nil, fmt.Errorf("SDM KDKMP source %d bukan Manager KDKMP", sourceEntryID)
	}

	var sourceEntry sdmEntry
	if err := source.First(&sourceEntry, sourceEntryID).Error; err != nil {
		return nil, fmt.Errorf("memuat SDM KDKMP source: %w", err)
	}

	var dailyEntries []models.EbitdamaxKdkmp
	if err := source.Where("sdm_kdkmp_entry_id = ?", sourceEntryID).Order("report_date").Find(&dailyEntries).Error; err != nil {
		return nil, fmt.Errorf("memuat dashboard KDKMP source: %w", err)
	}
	if len(dailyEntries) == 0 {
		return nil, fmt.Errorf("SDM KDKMP source %d tidak memiliki dashboard harian", sourceEntryID)
	}

	var tasks []models.Task
	if err := source.Model(&models.Task{}).
		Distinct("tasks.*").
		Joins("JOIN task_roles ON task_roles.task_id = tasks.id").
		Where("task_roles.role_id = ?", *user.RoleID).
		Order("tasks.sort_order NULLS LAST, tasks.id").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("memuat task source: %w", err)
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("akun Manager KDKMP source tidak memiliki task")
	}

	taskIDs := make([]int64, 0, len(tasks))
	taskByID := make(map[int64]models.Task, len(tasks))
	categoryIDs := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
		taskByID[task.ID] = task
		categoryIDs = append(categoryIDs, task.TaskCategoryID)
	}

	var categories []models.TaskCategory
	if err := source.Where("id IN ?", uniqueIDs(categoryIDs)).Order("id").Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("memuat kategori task source: %w", err)
	}
	var fields []models.TaskAdditionalField
	if err := source.Where("task_id IN ?", taskIDs).Order("task_id, sort_order, id").Find(&fields).Error; err != nil {
		return nil, fmt.Errorf("memuat field task source: %w", err)
	}

	var reports []models.TaskReport
	if err := source.Where("user_id = ? AND status = ?", user.ID, models.TaskReportCompleted).Order("finished_at, id").Find(&reports).Error; err != nil {
		return nil, fmt.Errorf("memuat laporan task source: %w", err)
	}
	if len(reports) == 0 {
		return nil, fmt.Errorf("akun Manager KDKMP source tidak memiliki laporan selesai")
	}
	reportIDs := make([]int64, 0, len(reports))
	for _, report := range reports {
		if _, ok := taskByID[report.TaskID]; !ok {
			return nil, fmt.Errorf("laporan task source %d tidak terkait task Manager KDKMP", report.ID)
		}
		if report.FinishedAt == nil {
			return nil, fmt.Errorf("laporan task source %d belum memiliki waktu selesai", report.ID)
		}
		reportIDs = append(reportIDs, report.ID)
	}

	var reportValues []models.TaskReportValue
	if err := source.Where("task_report_id IN ?", reportIDs).Order("task_report_id, id").Find(&reportValues).Error; err != nil {
		return nil, fmt.Errorf("memuat nilai laporan source: %w", err)
	}

	sourceLastDay := dailyEntries[len(dailyEntries)-1].ReportDate
	shiftDays := calendarDaysBetween(sourceLastDay, targetDay)
	return &importPlan{
		SourceEntryID: sourceEntryID,
		SourceEntry:   sourceEntry,
		Categories:    categories,
		Tasks:         tasks,
		Fields:        fields,
		Reports:       reports,
		ReportValues:  reportValues,
		DailyEntries:  dailyEntries,
		SourceLastDay: sourceLastDay,
		TargetDay:     targetDay,
		ShiftDays:     shiftDays,
	}, nil
}

func preflightTarget(target *gorm.DB) (*models.User, error) {
	email := strings.ToLower(strings.TrimSpace(config.GetEnv("SEED_MANAGER_EMAIL", "manager@agrinas.test")))
	var manager models.User
	if err := target.Preload("Role").Where("LOWER(email) = ?", email).First(&manager).Error; err != nil {
		return nil, fmt.Errorf("akun Manager seed tidak ditemukan: %w", err)
	}
	if !manager.IsKdkmpManager() {
		return nil, fmt.Errorf("akun Manager seed tidak memiliki role Manager KDKMP")
	}
	if manager.SDMKdkmpEntryID != nil {
		return nil, fmt.Errorf("akun Manager seed sudah terhubung ke data KDKMP; import dibatalkan")
	}

	for _, table := range []string{"task_categories", "tasks", "task_additional_fields", "task_roles", "task_reports", "task_report_values", "ebitdamax_kdkmp"} {
		var count int64
		if err := target.Table(table).Count(&count).Error; err != nil {
			return nil, fmt.Errorf("memeriksa tabel %s: %w", table, err)
		}
		if count != 0 {
			return nil, fmt.Errorf("tabel %s sudah memiliki data; import demo tidak akan menimpa data", table)
		}
	}

	var markerCount int64
	if err := target.Model(&sdmEntry{}).Where("nik = ?", demoNIK).Count(&markerCount).Error; err != nil {
		return nil, fmt.Errorf("memeriksa marker demo: %w", err)
	}
	if markerCount != 0 {
		return nil, fmt.Errorf("fixture demo legacy sudah pernah diimpor")
	}
	return &manager, nil
}

func applyPlan(target *gorm.DB, manager *models.User, plan *importPlan) error {
	return target.Transaction(func(tx *gorm.DB) error {
		now := time.Now().In(kdkmp.Location())
		managerID := manager.ID
		marker := demoNIK
		demoName := "KDKMP Demo Legacy"
		demoEntry := sdmEntry{
			NamaKoperasi:   demoName,
			JumlahKaryawan: plan.SourceEntry.JumlahKaryawan,
			CreatedBy:      &managerID,
			UpdatedBy:      &managerID,
			CreatedAt:      &now,
			UpdatedAt:      &now,
			NIK:            &marker,
			NamaKodam:      plan.SourceEntry.NamaKodam,
			NamaKorem:      plan.SourceEntry.NamaKorem,
			NamaKodim:      plan.SourceEntry.NamaKodim,
			Desa:           plan.SourceEntry.Desa,
			Kecamatan:      plan.SourceEntry.Kecamatan,
			KotaKabupaten:  plan.SourceEntry.KotaKabupaten,
			Batch:          plan.SourceEntry.Batch,
			Provinsi:       plan.SourceEntry.Provinsi,
		}
		if err := tx.Create(&demoEntry).Error; err != nil {
			return fmt.Errorf("membuat SDM KDKMP demo: %w", err)
		}
		if err := tx.Model(&models.User{}).Where("id = ?", manager.ID).Update("sdm_kdkmp_entry_id", demoEntry.ID).Error; err != nil {
			return fmt.Errorf("menghubungkan akun Manager dengan demo: %w", err)
		}

		categoryIDs := make(map[int64]int64, len(plan.Categories))
		for _, sourceCategory := range plan.Categories {
			category := sourceCategory
			category.ID = 0
			if err := tx.Create(&category).Error; err != nil {
				return fmt.Errorf("membuat kategori task: %w", err)
			}
			categoryIDs[sourceCategory.ID] = category.ID
		}

		taskIDs := make(map[int64]int64, len(plan.Tasks))
		tasksBySourceID := make(map[int64]models.Task, len(plan.Tasks))
		for _, sourceTask := range plan.Tasks {
			categoryID, ok := categoryIDs[sourceTask.TaskCategoryID]
			if !ok {
				return fmt.Errorf("kategori source untuk task %d tidak ditemukan", sourceTask.ID)
			}
			task := sourceTask
			task.ID = 0
			task.TaskCategoryID = categoryID
			if err := tx.Create(&task).Error; err != nil {
				return fmt.Errorf("membuat task: %w", err)
			}
			if err := tx.Create(&taskRole{TaskID: task.ID, RoleID: *manager.RoleID, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
				return fmt.Errorf("menghubungkan task dengan role Manager: %w", err)
			}
			taskIDs[sourceTask.ID] = task.ID
			tasksBySourceID[sourceTask.ID] = sourceTask
		}

		fieldIDs := make(map[int64]int64, len(plan.Fields))
		fieldBySourceID := make(map[int64]models.TaskAdditionalField, len(plan.Fields))
		for _, sourceField := range plan.Fields {
			taskID, ok := taskIDs[sourceField.TaskID]
			if !ok {
				return fmt.Errorf("task source untuk field %d tidak ditemukan", sourceField.ID)
			}
			field := sourceField
			field.ID = 0
			field.TaskID = taskID
			if err := tx.Create(&field).Error; err != nil {
				return fmt.Errorf("membuat field task: %w", err)
			}
			fieldIDs[sourceField.ID] = field.ID
			fieldBySourceID[sourceField.ID] = sourceField
		}

		reportIDs := make(map[int64]int64, len(plan.Reports))
		for _, sourceReport := range plan.Reports {
			taskID, ok := taskIDs[sourceReport.TaskID]
			if !ok {
				return fmt.Errorf("task source untuk laporan %d tidak ditemukan", sourceReport.ID)
			}
			task := tasksBySourceID[sourceReport.TaskID]
			report := sourceReport
			report.ID = 0
			report.TaskID = taskID
			report.UserID = manager.ID
			report.StartedPhoto = nil
			report.StartedDocuments = nil
			report.FinishedPhoto = nil
			report.FinishedDocuments = nil
			report.StartedAt = shiftTime(sourceReport.StartedAt, plan.ShiftDays)
			report.FinishedAt = shiftTime(sourceReport.FinishedAt, plan.ShiftDays)
			report.PeriodKey = rebasedPeriodKey(task.Period, report.FinishedAt, sourceReport.PeriodKey)
			report.CreatedAt = shiftValue(sourceReport.CreatedAt, plan.ShiftDays)
			report.UpdatedAt = shiftValue(sourceReport.UpdatedAt, plan.ShiftDays)
			if err := tx.Create(&report).Error; err != nil {
				return fmt.Errorf("membuat laporan task: %w", err)
			}
			reportIDs[sourceReport.ID] = report.ID
		}

		for _, sourceValue := range plan.ReportValues {
			reportID, ok := reportIDs[sourceValue.TaskReportID]
			if !ok {
				return fmt.Errorf("laporan source untuk nilai %d tidak ditemukan", sourceValue.ID)
			}
			fieldID, ok := fieldIDs[sourceValue.TaskAdditionalFieldID]
			if !ok {
				return fmt.Errorf("field source untuk nilai %d tidak ditemukan", sourceValue.ID)
			}
			value := sourceValue
			value.ID = 0
			value.TaskReportID = reportID
			value.TaskAdditionalFieldID = fieldID
			if fieldBySourceID[sourceValue.TaskAdditionalFieldID].InputType == "file" {
				value.Value = nil
			}
			value.CreatedAt = shiftValue(sourceValue.CreatedAt, plan.ShiftDays)
			value.UpdatedAt = shiftValue(sourceValue.UpdatedAt, plan.ShiftDays)
			if err := tx.Create(&value).Error; err != nil {
				return fmt.Errorf("membuat nilai laporan task: %w", err)
			}
		}

		for _, sourceDaily := range plan.DailyEntries {
			daily := sourceDaily
			daily.ID = 0
			daily.SDMKdkmpEntryID = demoEntry.ID
			daily.ReportDate = shiftValue(sourceDaily.ReportDate, plan.ShiftDays)
			daily.CreatedBy = &managerID
			daily.UpdatedBy = &managerID
			daily.CreatedAt = shiftValue(sourceDaily.CreatedAt, plan.ShiftDays)
			daily.UpdatedAt = shiftValue(sourceDaily.UpdatedAt, plan.ShiftDays)
			selected, err := selectedTaskIDs(sourceDaily, plan.Reports, tasksBySourceID, taskIDs)
			if err != nil {
				return err
			}
			daily.SelectedTaskIDs = selected
			if err := tx.Create(&daily).Error; err != nil {
				return fmt.Errorf("membuat dashboard harian KDKMP: %w", err)
			}
		}
		return nil
	})
}

func selectedTaskIDs(daily models.EbitdamaxKdkmp, reports []models.TaskReport, tasksBySourceID map[int64]models.Task, taskIDs map[int64]int64) (models.IntList, error) {
	selected := make(map[int64]bool)
	for _, sourceTaskID := range daily.SelectedTaskIDs {
		targetTaskID, ok := taskIDs[sourceTaskID]
		if !ok {
			return nil, fmt.Errorf("task pilihan source %d tidak termasuk data Manager KDKMP", sourceTaskID)
		}
		selected[targetTaskID] = true
	}

	for _, report := range reports {
		if report.FinishedAt == nil || dateKey(*report.FinishedAt) != dateKey(daily.ReportDate) {
			continue
		}
		task := tasksBySourceID[report.TaskID]
		if !task.IsMandatory {
			selected[taskIDs[report.TaskID]] = true
		}
	}

	ids := make(models.IntList, 0, len(selected))
	for id := range selected {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func rebasedPeriodKey(period string, finishedAt *time.Time, sourceKey *string) *string {
	if period != "daily" || finishedAt == nil {
		return sourceKey
	}
	value := dateKey(*finishedAt)
	return &value
}

func shiftTime(value *time.Time, days int) *time.Time {
	if value == nil {
		return nil
	}
	shifted := shiftValue(*value, days)
	return &shifted
}

func shiftValue(value time.Time, days int) time.Time {
	return value.AddDate(0, 0, days)
}

func calendarDaysBetween(from, to time.Time) int {
	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(end.Sub(start).Hours() / 24)
}

func dateKey(value time.Time) string {
	return value.In(kdkmp.Location()).Format("2006-01-02")
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

func printPlan(plan *importPlan, apply bool) {
	mode := "dry-run"
	if apply {
		mode = "apply"
	}
	fmt.Printf("%s fixture demo legacy: SDM source %d, %s → %s (%+d hari)\n",
		mode,
		plan.SourceEntryID,
		dateKey(plan.SourceLastDay),
		dateKey(plan.TargetDay),
		plan.ShiftDays,
	)
	fmt.Printf("kategori=%d task=%d field=%d laporan=%d nilai=%d dashboard=%d\n",
		len(plan.Categories),
		len(plan.Tasks),
		len(plan.Fields),
		len(plan.Reports),
		len(plan.ReportValues),
		len(plan.DailyEntries),
	)
}
