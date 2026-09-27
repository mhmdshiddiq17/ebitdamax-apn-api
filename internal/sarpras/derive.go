package sarpras

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"agrinaspangan/ebitda-api/internal/models"
)

// derivedUpdateColumns adalah kolom sdm_kdkmp_entries yang boleh ditimpa derive.
// Field manual/terlindungi (jumlah_karyawan, catatan, created_by, updated_by,
// created_at) sengaja tidak termasuk.
var derivedUpdateColumns = []string{
	"nama_koperasi",
	"provinsi",
	"nama_kodim",
	"desa",
	"kecamatan",
	"kota_kabupaten",
	"batch",
	"updated_at",
}

// DeriveSummary merangkum hasil derivasi titik sarpras ke data SDM KDKMP.
type DeriveSummary struct {
	SourcePoints int
	ToUpdate     int
	ToInsert     int
	TotalAfter   int
	Applied      bool
}

// DeriveService menurunkan titik sarpras (Sarpras Esensial 1 lengkap) ke
// sdm_kdkmp_entries berdasarkan NIK.
type DeriveService struct {
	db  *gorm.DB
	now func() time.Time
}

// NewDeriveService membuat service derivasi SDM.
func NewDeriveService(db *gorm.DB) *DeriveService {
	return &DeriveService{db: db, now: time.Now}
}

// Derive menghitung (dan bila apply=true menulis) data SDM dari titik sarpras.
func (s *DeriveService) Derive(ctx context.Context, apply bool) (DeriveSummary, error) {
	var points []models.KoperasiSarprasStatusPoint
	err := s.db.WithContext(ctx).
		Where("sarpras_primary_lengkap = ?", true).
		Where("nik IS NOT NULL AND nik <> ''").
		Order("synced_at DESC").
		Order("updated_at DESC").
		Order("id DESC").
		Find(&points).Error
	if err != nil {
		return DeriveSummary{}, fmt.Errorf("memuat titik sarpras: %w", err)
	}

	selected := selectLatestByNIK(points)
	summary := DeriveSummary{SourcePoints: len(selected), Applied: apply}
	if len(selected) == 0 {
		return summary, nil
	}

	if invalid := countEmptyNames(selected); invalid > 0 {
		return summary, fmt.Errorf("derivasi dibatalkan: %d data sumber tidak memiliki nama koperasi", invalid)
	}

	niks := make([]string, 0, len(selected))
	for _, point := range selected {
		niks = append(niks, *point.NIK)
	}

	var existing int64
	if err := s.db.WithContext(ctx).
		Model(&models.SdmKdkmpEntry{}).
		Where("nik IN ?", niks).
		Count(&existing).Error; err != nil {
		return summary, fmt.Errorf("menghitung data SDM existing: %w", err)
	}
	summary.ToUpdate = int(existing)
	summary.ToInsert = len(selected) - summary.ToUpdate

	var total int64
	if err := s.db.WithContext(ctx).Model(&models.SdmKdkmpEntry{}).Count(&total).Error; err != nil {
		return summary, fmt.Errorf("menghitung total data SDM: %w", err)
	}
	summary.TotalAfter = int(total) + summary.ToInsert

	if !apply {
		return summary, nil
	}

	records := deriveRecords(selected, s.now().Truncate(time.Second))
	if err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "nik"}},
			DoUpdates: clause.AssignmentColumns(derivedUpdateColumns),
		}).
		CreateInBatches(&records, 500).Error; err != nil {
		return summary, fmt.Errorf("menyimpan data SDM: %w", err)
	}

	return summary, nil
}

// selectLatestByNIK memilih satu titik per NIK (yang pertama = paling baru
// sesuai urutan query), mengikuti `unique('nik')` aplikasi lama.
func selectLatestByNIK(points []models.KoperasiSarprasStatusPoint) []models.KoperasiSarprasStatusPoint {
	seen := make(map[string]bool, len(points))
	selected := make([]models.KoperasiSarprasStatusPoint, 0, len(points))

	for _, point := range points {
		if point.NIK == nil || *point.NIK == "" || seen[*point.NIK] {
			continue
		}
		seen[*point.NIK] = true
		selected = append(selected, point)
	}

	return selected
}

func countEmptyNames(points []models.KoperasiSarprasStatusPoint) int {
	count := 0
	for _, point := range points {
		if point.NamaKoperasi == nil || strings.TrimSpace(*point.NamaKoperasi) == "" {
			count++
		}
	}
	return count
}

func deriveRecords(points []models.KoperasiSarprasStatusPoint, timestamp time.Time) []models.SdmKdkmpEntry {
	records := make([]models.SdmKdkmpEntry, 0, len(points))
	for _, point := range points {
		records = append(records, models.SdmKdkmpEntry{
			NIK:           point.NIK,
			NamaKoperasi:  point.NamaKoperasi,
			Provinsi:      point.Provinsi,
			KotaKabupaten: point.KotaKabupaten,
			Kecamatan:     point.Kecamatan,
			Desa:          point.Desa,
			Batch:         point.Batch,
			NamaKodim:     point.Kodim,
			CreatedAt:     timestamp,
			UpdatedAt:     timestamp,
		})
	}
	return records
}
