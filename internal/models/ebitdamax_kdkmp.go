package models

import "time"

// Role operasional KDKMP (urutan tampilan mengikuti aplikasi lama).
var OperationalAttendanceRoleKeys = []string{
	"pramuniaga",
	"kasir",
	"karyawan_umkm",
	"security",
	"driver_truck",
	"driver_pickup",
	"driver_motor_roda_tiga",
}

// Label role operasional KDKMP.
var OperationalAttendanceRoleLabels = map[string]string{
	"pramuniaga":             "Pramuniaga",
	"kasir":                  "Kasir",
	"karyawan_umkm":          "Karyawan UMKM",
	"security":               "Security",
	"driver_truck":           "Driver Truck",
	"driver_pickup":          "Driver Pickup",
	"driver_motor_roda_tiga": "Driver Motor Roda Tiga",
}

// EbitdamaxKdkmp merepresentasikan tabel `ebitdamax_kdkmp` (data harian KDKMP).
// Kolom uang disimpan sebagai string (mengikuti skema existing).
type EbitdamaxKdkmp struct {
	ID                           int64      `gorm:"primaryKey" json:"id"`
	SDMKdkmpEntryID              int64      `gorm:"column:sdm_kdkmp_entry_id" json:"sdm_kdkmp_entry_id"`
	ReportDate                   time.Time  `gorm:"column:report_date" json:"report_date"`
	TargetRevenue                *string    `gorm:"column:target_revenue" json:"target_revenue"`
	PlanRevenue                  *string    `gorm:"column:plan_revenue" json:"plan_revenue"`
	PlanRevenueRequiresReview    bool       `gorm:"column:plan_revenue_requires_review" json:"plan_revenue_requires_review"`
	ActualRevenue                *string    `gorm:"column:actual_revenue" json:"actual_revenue"`
	TargetCost                   *string    `gorm:"column:target_cost" json:"target_cost"`
	PlanCost                     *string    `gorm:"column:plan_cost" json:"plan_cost"`
	ActualCost                   *string    `gorm:"column:actual_cost" json:"actual_cost"`
	ActualVariableCost           *string    `gorm:"column:actual_variable_cost" json:"actual_variable_cost"`
	TargetEbitda                 *string    `gorm:"column:target_ebitda" json:"target_ebitda"`
	PlanEbitda                   *string    `gorm:"column:plan_ebitda" json:"plan_ebitda"`
	ActualEbitda                 *string    `gorm:"column:actual_ebitda" json:"actual_ebitda"`
	TargetEbitdaMargin           *string    `gorm:"column:target_ebitda_margin" json:"target_ebitda_margin"`
	ActualEbitdaMargin           *string    `gorm:"column:actual_ebitda_margin" json:"actual_ebitda_margin"`
	TotalDuration                *string    `gorm:"column:total_duration" json:"total_duration"`
	PerformanceScoring           *string    `gorm:"column:performance_scoring" json:"performance_scoring"`
	OperationalAttendance        JSONIntMap `gorm:"column:operational_attendance;type:jsonb" json:"-"`
	OperationalAttendanceSavedAt *time.Time `gorm:"column:operational_attendance_saved_at" json:"operational_attendance_saved_at"`
	SelectedTaskIDs              IntList    `gorm:"column:selected_task_ids;type:jsonb" json:"-"`
	CreatedBy                    *int64     `gorm:"column:created_by" json:"created_by"`
	UpdatedBy                    *int64     `gorm:"column:updated_by" json:"updated_by"`
	CreatedAt                    time.Time  `json:"created_at"`
	UpdatedAt                    time.Time  `json:"updated_at"`
}

// TableName memetakan struct ke tabel existing.
func (EbitdamaxKdkmp) TableName() string {
	return "ebitdamax_kdkmp"
}

// NormalizeOperationalAttendance menyeragamkan nilai kehadiran per role
// menjadi integer >= 0.
func NormalizeOperationalAttendance(attendance map[string]int) map[string]int {
	normalized := make(map[string]int, len(OperationalAttendanceRoleKeys))
	for _, role := range OperationalAttendanceRoleKeys {
		value := attendance[role]
		if value < 0 {
			value = 0
		}
		normalized[role] = value
	}
	return normalized
}

// HasConfirmedOperationalAttendance mengecek kehadiran sudah dikonfirmasi.
func (e *EbitdamaxKdkmp) HasConfirmedOperationalAttendance() bool {
	return e.OperationalAttendanceSavedAt != nil
}
