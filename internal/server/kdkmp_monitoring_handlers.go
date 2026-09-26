package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/kdkmp"
	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
)

const monitoringPerPage = 25

var monitoringStatusValues = map[string]bool{
	"all":             true,
	"complete":        true,
	"not_filled":      true,
	"requires_review": true,
}

var monitoringLevelValues = map[string]bool{
	kdkmp.ConsolidationLevelNational: true,
	kdkmp.ConsolidationLevelProvince: true,
	kdkmp.ConsolidationLevelRegency:  true,
	kdkmp.ConsolidationLevelDistrict: true,
	kdkmp.ConsolidationLevelVillage:  true,
}

type monitoringParams struct {
	monthStart time.Time
	month      string
	detailDate *string
	search     string
	status     string
	level      string
	region     map[string]string
	page       int
}

// RequireMonitoringAccess membatasi monitoring untuk superadmin & manager wilayah.
func RequireMonitoringAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.CurrentUser(c)
		if user == nil || (!user.IsSuperadmin() && !user.IsRegionalManager()) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Monitoring KDKMP hanya untuk superadmin dan manager wilayah"})
			return
		}
		c.Next()
	}
}

// KdkmpMonitoringHandler godoc
//
//	@Summary	Monitoring dashboard KDKMP
//	@Description	Ringkasan pengisian harian, konsolidasi wilayah, dan grafik biaya bulanan KDKMP.
//	@Tags		KDKMP Monitoring
//	@Produce	json
//	@Security	CookieAuth
//	@Param		month				query	string	false	"Bulan grafik (YYYY-MM)"
//	@Param		date				query	string	false	"Legacy: tanggal (YYYY-MM-DD)"
//	@Param		detail_date			query	string	false	"Tanggal rincian (YYYY-MM-DD)"
//	@Param		search				query	string	false	"Cari nama/NIK/manager/wilayah"
//	@Param		status				query	string	false	"Status pengisian (all|complete|not_filled|requires_review)"
//	@Param		consolidation_level	query	string	false	"Level konsolidasi (national|province|regency|district|village)"
//	@Param		provinsi			query	string	false	"Filter provinsi"
//	@Param		kota_kabupaten		query	string	false	"Filter kota/kabupaten"
//	@Param		kecamatan			query	string	false	"Filter kecamatan"
//	@Param		desa				query	string	false	"Filter desa"
//	@Param		page				query	int		false	"Halaman daftar KDKMP"
//	@Success	200	{object}	map[string]any
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	422	{object}	map[string]string
//	@Router		/api/v1/admin/kdkmp-dashboard [get]
func KdkmpMonitoringHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	businessDate := kdkmp.BusinessDate()
	params, ok := parseMonitoringParams(c, businessDate)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	db := AppDeps.DB
	regional := kdkmp.NewRegionalService(db)

	access, err := regional.FilterContext(ctx, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat cakupan akses"})
		return
	}

	regionFilters := make(map[string]string, len(kdkmp.RegionFields))
	for _, field := range kdkmp.RegionFields {
		value := params.region[field]
		if value == "" {
			value = lockedFilterValue(access.LockedFilters, field)
		}
		regionFilters[field] = value
	}

	level := params.level
	if !access.IsNational && level == kdkmp.ConsolidationLevelNational {
		level = kdkmp.ConsolidationLevelProvince
	}

	periodStart := params.monthStart
	periodEnd := periodStart.AddDate(0, 1, -1)
	if periodStart.Year() == businessDate.Year() && periodStart.Month() == businessDate.Month() {
		periodEnd = businessDate
	}

	baseQuery, err := kdkmp.AccessibleManagedKdkmpQuery(ctx, db, user, regionFilters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat data KDKMP"})
		return
	}

	selected, ok := monitoringSelectedEntry(c, baseQuery, regionFilters)
	if !ok {
		return
	}

	var detailDate *string
	if selected != nil && params.detailDate != nil {
		date, _ := time.ParseInLocation("2006-01-02", *params.detailDate, kdkmp.Location())
		if !date.Before(periodStart) && !date.After(periodEnd) {
			detailDate = params.detailDate
		}
	}
	reportDate := kdkmp.DateString(periodEnd)
	if detailDate != nil {
		reportDate = *detailDate
	}

	var total, filled, requiresReview int64
	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghitung ringkasan KDKMP"})
		return
	}
	if err := monitoringDailyRecordQuery(baseQuery, reportDate, false).Count(&filled).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghitung ringkasan KDKMP"})
		return
	}
	if err := monitoringDailyRecordQuery(baseQuery, reportDate, true).Count(&requiresReview).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghitung ringkasan KDKMP"})
		return
	}

	entries, entriesTotal, ok := monitoringEntriesPage(c, baseQuery, params, reportDate)
	if !ok {
		return
	}
	payload, ok := monitoringEntriesPayload(c, entries, reportDate)
	if !ok {
		return
	}

	consolidationRows, err := kdkmp.NewConsolidationService(db).
		ForEntries(ctx, baseQuery.Session(&gorm.Session{}), reportDate, level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat konsolidasi wilayah"})
		return
	}

	var matrix *kdkmp.MonthlyFinancialMatrix
	if selected != nil {
		result, err := kdkmp.MonthlyFinancialMatrixForEntry(ctx, db, selected.ID, periodStart, periodEnd)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat grafik biaya bulanan"})
			return
		}
		matrix = &result
	}

	regionOptions, err := regional.RegionOptions(ctx, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat opsi wilayah"})
		return
	}

	filterPayload := gin.H{
		"month":               params.month,
		"detail_date":         detailDate,
		"search":              params.search,
		"status":              params.status,
		"consolidation_level": level,
	}
	for _, field := range kdkmp.RegionFields {
		filterPayload[field] = nullableRegionValue(regionFilters[field])
	}

	c.JSON(http.StatusOK, gin.H{
		"business_date": kdkmp.DateString(businessDate),
		"entries": gin.H{
			"data": payload,
			"meta": gin.H{
				"page":        params.page,
				"per_page":    monitoringPerPage,
				"total":       entriesTotal,
				"total_pages": monitoringTotalPages(entriesTotal),
			},
		},
		"summary": gin.H{
			"total":           total,
			"complete":        filled,
			"not_filled":      total - filled,
			"requires_review": requiresReview,
		},
		"filters":                  filterPayload,
		"region_options":           regionOptions,
		"regional_access":          access,
		"consolidation":            gin.H{"level": level, "rows": consolidationRows},
		"selected_kdkmp":           monitoringSelectedPayload(selected),
		"monthly_financial_matrix": matrix,
	})
}

func parseMonitoringParams(c *gin.Context, businessDate time.Time) (monitoringParams, bool) {
	params := monitoringParams{
		status: "all",
		level:  kdkmp.ConsolidationLevelNational,
		page:   1,
		region: make(map[string]string, len(kdkmp.RegionFields)),
	}

	monthValue := strings.TrimSpace(c.Query("month"))
	if monthValue == "" {
		if legacy := strings.TrimSpace(c.Query("date")); legacy != "" {
			date, err := time.ParseInLocation("2006-01-02", legacy, kdkmp.Location())
			if err != nil || date.After(businessDate) {
				return params, monitoringValidationError(c, "Tanggal monitoring tidak boleh melewati hari ini")
			}
			monthValue = legacy[:7]
		}
	}
	if monthValue == "" {
		monthValue = businessDate.Format("2006-01")
	}

	monthStart, err := time.ParseInLocation("2006-01", monthValue, kdkmp.Location())
	if err != nil {
		return params, monitoringValidationError(c, "Bulan monitoring harus berformat YYYY-MM")
	}
	if monthStart.After(businessDate) {
		return params, monitoringValidationError(c, "Bulan monitoring tidak boleh melewati bulan berjalan")
	}
	params.month = monthValue
	params.monthStart = monthStart

	if detail := strings.TrimSpace(c.Query("detail_date")); detail != "" {
		date, err := time.ParseInLocation("2006-01-02", detail, kdkmp.Location())
		if err != nil || date.After(businessDate) {
			return params, monitoringValidationError(c, "Tanggal rincian tidak boleh melewati hari ini")
		}
		params.detailDate = &detail
	}

	params.search = strings.TrimSpace(c.Query("search"))
	if len(params.search) > 255 {
		return params, monitoringValidationError(c, "Kata kunci pencarian terlalu panjang")
	}

	if status := strings.TrimSpace(c.Query("status")); status != "" {
		if !monitoringStatusValues[status] {
			return params, monitoringValidationError(c, "Status pengisian tidak dikenal")
		}
		params.status = status
	}

	if level := strings.TrimSpace(c.Query("consolidation_level")); level != "" {
		if !monitoringLevelValues[level] {
			return params, monitoringValidationError(c, "Level konsolidasi tidak dikenal")
		}
		params.level = level
	}

	for _, field := range kdkmp.RegionFields {
		value := strings.TrimSpace(c.Query(field))
		if len(value) > 255 {
			return params, monitoringValidationError(c, "Filter wilayah terlalu panjang")
		}
		params.region[field] = value
	}

	if requested, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && requested > 0 {
		params.page = requested
	}

	return params, true
}

func monitoringValidationError(c *gin.Context, message string) bool {
	c.JSON(http.StatusUnprocessableEntity, gin.H{"message": message})
	return false
}

func lockedFilterValue(locked kdkmp.LockedFilters, field string) string {
	switch field {
	case "provinsi":
		return derefString(locked.Provinsi)
	case "kota_kabupaten":
		return derefString(locked.KotaKabupaten)
	case "kecamatan":
		return derefString(locked.Kecamatan)
	case "desa":
		return derefString(locked.Desa)
	default:
		return ""
	}
}

func nullableRegionValue(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// monitoringDailyRecordQuery membatasi entry yang punya record harian pada tanggal.
func monitoringDailyRecordQuery(query *gorm.DB, reportDate string, requiresReview bool) *gorm.DB {
	subQuery := "EXISTS (SELECT 1 FROM ebitdamax_kdkmp d WHERE d.sdm_kdkmp_entry_id = sdm_kdkmp_entries.id AND d.report_date = ?"
	args := []any{reportDate}
	if requiresReview {
		subQuery += " AND d.plan_revenue_requires_review = ?"
		args = append(args, true)
	}
	subQuery += ")"

	return query.Session(&gorm.Session{}).Where(subQuery, args...)
}

func monitoringSelectedEntry(c *gin.Context, baseQuery *gorm.DB, regionFilters map[string]string) (*models.SdmKdkmpEntry, bool) {
	if regionFilters["desa"] == "" {
		return nil, true
	}

	var entry models.SdmKdkmpEntry
	err := baseQuery.Session(&gorm.Session{}).
		Select("id, nik, nama_koperasi, desa, kecamatan, kota_kabupaten, provinsi").
		First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, true
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat KDKMP terpilih"})
		return nil, false
	}
	return &entry, true
}

func monitoringEntriesPage(c *gin.Context, baseQuery *gorm.DB, params monitoringParams, reportDate string) ([]models.SdmKdkmpEntry, int64, bool) {
	query := baseQuery.Session(&gorm.Session{})

	if params.search != "" {
		like := "%" + params.search + "%"
		query = query.Where(
			`(sdm_kdkmp_entries.nama_koperasi ILIKE ? OR sdm_kdkmp_entries.nik ILIKE ?
			  OR sdm_kdkmp_entries.desa ILIKE ? OR sdm_kdkmp_entries.kecamatan ILIKE ?
			  OR sdm_kdkmp_entries.kota_kabupaten ILIKE ? OR sdm_kdkmp_entries.provinsi ILIKE ?
			  OR EXISTS (SELECT 1 FROM users mu WHERE mu.sdm_kdkmp_entry_id = sdm_kdkmp_entries.id
			             AND (mu.name ILIKE ? OR mu.email ILIKE ?)))`,
			like, like, like, like, like, like, like, like,
		)
	}

	switch params.status {
	case "complete":
		query = monitoringDailyRecordQuery(query, reportDate, false)
	case "not_filled":
		query = query.Where(
			"NOT EXISTS (SELECT 1 FROM ebitdamax_kdkmp d WHERE d.sdm_kdkmp_entry_id = sdm_kdkmp_entries.id AND d.report_date = ?)",
			reportDate,
		)
	case "requires_review":
		query = monitoringDailyRecordQuery(query, reportDate, true)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghitung daftar KDKMP"})
		return nil, 0, false
	}

	var entries []models.SdmKdkmpEntry
	err := query.
		Order("sdm_kdkmp_entries.nama_koperasi").
		Offset((params.page - 1) * monitoringPerPage).
		Limit(monitoringPerPage).
		Find(&entries).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat daftar KDKMP"})
		return nil, 0, false
	}

	return entries, total, true
}

func monitoringEntriesPayload(c *gin.Context, entries []models.SdmKdkmpEntry, reportDate string) ([]gin.H, bool) {
	entryIDs := make([]int64, 0, len(entries))
	for _, entry := range entries {
		entryIDs = append(entryIDs, entry.ID)
	}

	managersByEntry := make(map[int64]models.User, len(entries))
	managerIDs := make([]int64, 0, len(entries))
	recordsByEntry := make(map[int64]models.EbitdamaxKdkmp, len(entries))

	if len(entryIDs) > 0 {
		ctx := c.Request.Context()

		var managers []models.User
		err := AppDeps.DB.WithContext(ctx).
			Select("id, name, email, username, sdm_kdkmp_entry_id").
			Where("sdm_kdkmp_entry_id IN ?", entryIDs).
			Order("id").
			Find(&managers).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat akun manager"})
			return nil, false
		}
		for _, manager := range managers {
			if manager.SDMKdkmpEntryID == nil {
				continue
			}
			if _, exists := managersByEntry[*manager.SDMKdkmpEntryID]; exists {
				continue
			}
			managersByEntry[*manager.SDMKdkmpEntryID] = manager
			managerIDs = append(managerIDs, manager.ID)
		}

		var records []models.EbitdamaxKdkmp
		err = AppDeps.DB.WithContext(ctx).
			Where("sdm_kdkmp_entry_id IN ? AND report_date = ?", entryIDs, reportDate).
			Order("id").
			Find(&records).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat data harian KDKMP"})
			return nil, false
		}
		for _, record := range records {
			if _, exists := recordsByEntry[record.SDMKdkmpEntryID]; !exists {
				recordsByEntry[record.SDMKdkmpEntryID] = record
			}
		}
	}

	reportBusinessDate, err := time.ParseInLocation("2006-01-02", reportDate, kdkmp.Location())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Tanggal rincian tidak valid"})
		return nil, false
	}
	metricsByUser, err := kdkmp.MetricsForUsers(c.Request.Context(), AppDeps.DB, managerIDs, reportBusinessDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghitung metrik KDKMP"})
		return nil, false
	}

	payload := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		manager, hasManager := managersByEntry[entry.ID]
		record, hasRecord := recordsByEntry[entry.ID]

		var metrics kdkmp.Metrics
		if hasManager {
			metrics = metricsByUser[manager.ID]
		}

		payload = append(payload, monitoringEntryPayload(entry, manager, hasManager, record, hasRecord, metrics))
	}

	return payload, true
}

func monitoringEntryPayload(
	entry models.SdmKdkmpEntry,
	manager models.User,
	hasManager bool,
	record models.EbitdamaxKdkmp,
	hasRecord bool,
	metrics kdkmp.Metrics,
) gin.H {
	managerPayload := any(nil)
	if hasManager {
		managerPayload = gin.H{"name": manager.Name, "email": manager.Email, "username": manager.Username}
	}

	dailyPayload := any(nil)
	if hasRecord {
		dailyPayload = gin.H{
			"is_complete":                  true,
			"plan_revenue_requires_review": record.PlanRevenueRequiresReview,
			"updated_at":                   record.UpdatedAt,
			"target_revenue":               strconv.Itoa(kdkmp.TargetRevenue),
			"plan_revenue":                 record.PlanRevenue,
			"actual_revenue":               record.ActualRevenue,
			"variable_cost":                record.PlanCost,
			"actual_cost":                  record.ActualCost,
			"actual_ebitda_margin":         kdkmp.CalculateActualEbitdaMargin(derefString(record.ActualRevenue)),
			"total_duration":               record.TotalDuration,
			"performance_scoring":          kdkmp.CalculatePerformanceScoring(derefString(record.PlanRevenue), derefString(record.ActualRevenue), metrics.CompletionRate, metrics.TimeComplianceRate),
		}
	}

	return gin.H{
		"id":             entry.ID,
		"nik":            entry.NIK,
		"name":           entry.NamaKoperasi,
		"desa":           entry.Desa,
		"kecamatan":      entry.Kecamatan,
		"kota_kabupaten": entry.KotaKabupaten,
		"provinsi":       entry.Provinsi,
		"manager":        managerPayload,
		"metrics":        gin.H{"task_completion_rate": metrics.CompletionRate},
		"daily_entry":    dailyPayload,
	}
}

func monitoringSelectedPayload(entry *models.SdmKdkmpEntry) any {
	if entry == nil {
		return nil
	}
	return gin.H{
		"id":             entry.ID,
		"nik":            entry.NIK,
		"name":           entry.NamaKoperasi,
		"desa":           entry.Desa,
		"kecamatan":      entry.Kecamatan,
		"kota_kabupaten": entry.KotaKabupaten,
		"provinsi":       entry.Provinsi,
	}
}

func monitoringTotalPages(total int64) int {
	if total <= 0 {
		return 1
	}
	return int((total + monitoringPerPage - 1) / monitoringPerPage)
}
