package kdkmp

import (
	"context"
	"sort"
	"strings"

	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/models"
)

// Level konsolidasi wilayah (mirror level pada KdkmpConsolidationService).
const (
	ConsolidationLevelNational = "national"
	ConsolidationLevelProvince = "province"
	ConsolidationLevelRegency  = "regency"
	ConsolidationLevelDistrict = "district"
	ConsolidationLevelVillage  = "village"
)

// ConsolidationRow adalah ringkasan satu wilayah (mirror KdkmpConsolidationService::forEntries).
type ConsolidationRow struct {
	Key           string   `json:"key"`
	Label         string   `json:"label"`
	Provinsi      *string  `json:"provinsi"`
	KotaKabupaten *string  `json:"kota_kabupaten"`
	Kecamatan     *string  `json:"kecamatan"`
	Desa          *string  `json:"desa"`
	TotalKdkmp    int      `json:"total_kdkmp"`
	CompleteKdkmp int      `json:"complete_kdkmp"`
	PlanRevenue   *float64 `json:"plan_revenue"`
	ActualRevenue *float64 `json:"actual_revenue"`
	Gap           *float64 `json:"gap"`
}

// ConsolidationService menyusun konsolidasi KDKMP per level wilayah.
type ConsolidationService struct {
	db *gorm.DB
}

// NewConsolidationService membuat service konsolidasi.
func NewConsolidationService(db *gorm.DB) *ConsolidationService {
	return &ConsolidationService{db: db}
}

// ForEntries mengambil entry (dengan filter yang sudah diterapkan pemanggil) beserta
// record harian pada reportDate, lalu mengonsolidasikannya per level.
func (s *ConsolidationService) ForEntries(
	ctx context.Context,
	entriesQuery *gorm.DB,
	reportDate string,
	level string,
) ([]ConsolidationRow, error) {
	var entries []models.SdmKdkmpEntry
	err := entriesQuery.
		Preload("DailyEbitdaRecords", func(db *gorm.DB) *gorm.DB {
			return db.Where("report_date = ?", reportDate)
		}).
		Find(&entries).Error
	if err != nil {
		return nil, err
	}

	return ConsolidateEntries(entries, level), nil
}

// ConsolidateEntries mengelompokkan entry (dengan DailyEbitdaRecords terfilter)
// menjadi baris konsolidasi; baris diurutkan natural case-insensitive per label.
func ConsolidateEntries(entries []models.SdmKdkmpEntry, level string) []ConsolidationRow {
	type accumulator struct {
		first       *models.SdmKdkmpEntry
		total       int
		complete    int
		planSum     float64
		planCount   int
		actualSum   float64
		actualCount int
	}

	groups := make(map[string]*accumulator)
	order := make([]string, 0, len(entries))

	for index := range entries {
		entry := &entries[index]
		key := consolidationKey(entry, level)

		group, exists := groups[key]
		if !exists {
			group = &accumulator{first: entry}
			groups[key] = group
			order = append(order, key)
		}

		group.total++

		for recordIndex := range entry.DailyEbitdaRecords {
			record := &entry.DailyEbitdaRecords[recordIndex]
			group.complete++

			if value, ok := numericValue(deref(record.PlanRevenue)); ok {
				group.planSum += value
				group.planCount++
			}
			if value, ok := numericValue(deref(record.ActualRevenue)); ok {
				group.actualSum += value
				group.actualCount++
			}
		}
	}

	rows := make([]ConsolidationRow, 0, len(groups))
	for _, key := range order {
		group := groups[key]

		row := ConsolidationRow{
			Key:           key,
			Label:         consolidationLabel(group.first, level),
			Provinsi:      group.first.Provinsi,
			KotaKabupaten: group.first.KotaKabupaten,
			Kecamatan:     group.first.Kecamatan,
			Desa:          group.first.Desa,
			TotalKdkmp:    group.total,
			CompleteKdkmp: group.complete,
		}

		if group.planCount > 0 {
			value := roundMoney(group.planSum)
			row.PlanRevenue = &value
		}
		if group.actualCount > 0 {
			value := roundMoney(group.actualSum)
			row.ActualRevenue = &value
		}
		if row.PlanRevenue != nil && row.ActualRevenue != nil {
			gap := roundMoney(*row.ActualRevenue - *row.PlanRevenue)
			row.Gap = &gap
		}

		rows = append(rows, row)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		return naturalLess(rows[i].Label, rows[j].Label)
	})

	return rows
}

func consolidationKey(entry *models.SdmKdkmpEntry, level string) string {
	switch level {
	case ConsolidationLevelProvince:
		return deref(entry.Provinsi)
	case ConsolidationLevelRegency:
		return strings.Join([]string{deref(entry.Provinsi), deref(entry.KotaKabupaten)}, "|")
	case ConsolidationLevelDistrict:
		return strings.Join([]string{deref(entry.Provinsi), deref(entry.KotaKabupaten), deref(entry.Kecamatan)}, "|")
	case ConsolidationLevelVillage:
		return strings.Join([]string{deref(entry.Provinsi), deref(entry.KotaKabupaten), deref(entry.Kecamatan), deref(entry.Desa)}, "|")
	default:
		return "national"
	}
}

func consolidationLabel(entry *models.SdmKdkmpEntry, level string) string {
	switch level {
	case ConsolidationLevelProvince:
		return labelOrDash(entry.Provinsi)
	case ConsolidationLevelRegency:
		return labelOrDash(entry.KotaKabupaten)
	case ConsolidationLevelDistrict:
		return labelOrDash(entry.Kecamatan)
	case ConsolidationLevelVillage:
		return labelOrDash(entry.Desa)
	default:
		return "Indonesia"
	}
}

func labelOrDash(value *string) string {
	if value == nil || *value == "" {
		return "-"
	}
	return *value
}

// naturalLess membandingkan dua string secara natural (angka dibandingkan sebagai
// angka) dan case-insensitive — meniru SORT_NATURAL | SORT_FLAG_CASE.
func naturalLess(a string, b string) bool {
	left := strings.ToLower(a)
	right := strings.ToLower(b)

	i, j := 0, 0
	for i < len(left) && j < len(right) {
		leftIsDigit := isASCIIDigit(left[i])
		rightIsDigit := isASCIIDigit(right[j])

		if leftIsDigit && rightIsDigit {
			leftStart, rightStart := i, j
			for i < len(left) && isASCIIDigit(left[i]) {
				i++
			}
			for j < len(right) && isASCIIDigit(right[j]) {
				j++
			}

			leftNumber := strings.TrimLeft(left[leftStart:i], "0")
			rightNumber := strings.TrimLeft(right[rightStart:j], "0")

			if len(leftNumber) != len(rightNumber) {
				return len(leftNumber) < len(rightNumber)
			}
			if leftNumber != rightNumber {
				return leftNumber < rightNumber
			}
			continue
		}

		if left[i] != right[j] {
			return left[i] < right[j]
		}
		i++
		j++
	}

	return (len(left) - i) < (len(right) - j)
}

func isASCIIDigit(value byte) bool {
	return value >= '0' && value <= '9'
}
