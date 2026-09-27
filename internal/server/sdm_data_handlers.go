package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
)

const sdmDataPerPage = 25

type updateSdmDataRequest struct {
	JumlahKaryawan *int `json:"jumlah_karyawan"`
}

// ListSdmDataHandler godoc
//
//	@Summary	Daftar data SDM KDKMP
//	@Description	Daftar KDKMP dari data pembangunan beserta jumlah karyawan (superadmin).
//	@Tags		SDM Data
//	@Produce	json
//	@Security	CookieAuth
//	@Param		search	query	string	false	"Cari nama koperasi, NIK, kodim, atau wilayah"
//	@Param		page	query	int		false	"Halaman"
//	@Success	200	{object}	map[string]any
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	422	{object}	map[string]string
//	@Router		/api/v1/admin/sdm-data [get]
func ListSdmDataHandler(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 255 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Kata kunci pencarian terlalu panjang"})
		return
	}

	page := 1
	if requested, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && requested > 0 {
		page = requested
	}

	ctx := c.Request.Context()
	query := AppDeps.DB.WithContext(ctx).Model(&models.SdmKdkmpEntry{})

	if search != "" {
		like := "%" + search + "%"
		query = query.Where(
			`(nama_koperasi ILIKE ? OR nik ILIKE ? OR nama_kodim ILIKE ?
			  OR kota_kabupaten ILIKE ? OR kecamatan ILIKE ? OR provinsi ILIKE ?)`,
			like, like, like, like, like, like,
		)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghitung data SDM"})
		return
	}

	var entries []models.SdmKdkmpEntry
	err := query.
		Order("nama_koperasi").
		Offset((page - 1) * sdmDataPerPage).
		Limit(sdmDataPerPage).
		Find(&entries).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat data SDM"})
		return
	}

	var added int64
	if err := AppDeps.DB.WithContext(ctx).
		Model(&models.SdmKdkmpEntry{}).
		Where("jumlah_karyawan > 0").
		Count(&added).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghitung ringkasan SDM"})
		return
	}

	var totalKaryawan int64
	if err := AppDeps.DB.WithContext(ctx).
		Model(&models.SdmKdkmpEntry{}).
		Select("COALESCE(SUM(jumlah_karyawan), 0)").
		Scan(&totalKaryawan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghitung ringkasan SDM"})
		return
	}

	payload := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		payload = append(payload, gin.H{
			"id":              entry.ID,
			"nik":             entry.NIK,
			"nama_koperasi":   entry.NamaKoperasi,
			"provinsi":        entry.Provinsi,
			"kota_kabupaten":  entry.KotaKabupaten,
			"kecamatan":       entry.Kecamatan,
			"jumlah_karyawan": entry.JumlahKaryawan,
			"updated_at":      entry.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": payload,
		"meta": gin.H{
			"page":        page,
			"per_page":    sdmDataPerPage,
			"total":       total,
			"total_pages": sdmDataTotalPages(total),
		},
		"summary": gin.H{
			"jumlah_kdkmp_ditambahkan": added,
			"total_karyawan":           totalKaryawan,
		},
		"filters": gin.H{"search": search},
	})
}

// UpdateSdmDataHandler godoc
//
//	@Summary	Simpan jumlah karyawan KDKMP
//	@Description	Memperbarui jumlah karyawan satu data SDM KDKMP (superadmin).
//	@Tags		SDM Data
//	@Accept		json
//	@Produce	json
//	@Security	CookieAuth
//	@Param		id		path	int						true	"ID data SDM"
//	@Param		payload	body	updateSdmDataRequest	true	"Jumlah karyawan"
//	@Success	200	{object}	map[string]string
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Failure	422	{object}	map[string]string
//	@Router		/api/v1/admin/sdm-data/{id} [put]
func UpdateSdmDataHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	entryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || entryID <= 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Data SDM tidak ditemukan"})
		return
	}

	var request updateSdmDataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Jumlah karyawan tidak valid"})
		return
	}

	value, message := validateJumlahKaryawan(request.JumlahKaryawan)
	if message != "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": message})
		return
	}

	var entry models.SdmKdkmpEntry
	err = AppDeps.DB.WithContext(c.Request.Context()).First(&entry, entryID).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Data SDM tidak ditemukan"})
		return
	}

	err = AppDeps.DB.WithContext(c.Request.Context()).
		Model(&models.SdmKdkmpEntry{}).
		Where("id = ?", entryID).
		Updates(map[string]any{
			"jumlah_karyawan": value,
			"updated_by":      user.ID,
		}).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan jumlah karyawan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jumlah karyawan berhasil disimpan."})
}

func validateJumlahKaryawan(value *int) (int, string) {
	if value == nil {
		return 0, "Jumlah karyawan wajib diisi."
	}
	if *value < 0 {
		return 0, "Jumlah karyawan tidak boleh negatif."
	}
	return *value, ""
}

func sdmDataTotalPages(total int64) int {
	if total <= 0 {
		return 1
	}
	return int((total + sdmDataPerPage - 1) / sdmDataPerPage)
}
