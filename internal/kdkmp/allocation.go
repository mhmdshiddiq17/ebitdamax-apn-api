package kdkmp

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"agrinaspangan/ebitda-api/internal/models"
)

// AllocationService menghitung alokasi personel operasional dari laporan
// task yang sedang berjalan.
type AllocationService struct {
	db *gorm.DB
}

// NewAllocationService membuat service alokasi operasional.
func NewAllocationService(db *gorm.DB) *AllocationService {
	return &AllocationService{db: db}
}

// SummaryForUser mengembalikan alokasi terpakai dan sisa per role operasional.
func (s *AllocationService) SummaryForUser(
	ctx context.Context,
	user *models.User,
	businessDate time.Time,
	attendance map[string]int,
	exceptReportID *int64,
	lockForUpdate bool,
) (map[string]int, map[string]int, error) {
	return s.summary(ctx, s.db, user, businessDate, attendance, exceptReportID, lockForUpdate)
}

// SummaryForUserWithTx menghitung alokasi di dalam transaksi yang sedang berjalan.
func (s *AllocationService) SummaryForUserWithTx(
	tx *gorm.DB,
	user *models.User,
	businessDate time.Time,
	attendance map[string]int,
	exceptReportID *int64,
	lockForUpdate bool,
) (map[string]int, map[string]int, error) {
	return s.summary(context.Background(), tx, user, businessDate, attendance, exceptReportID, lockForUpdate)
}

func (s *AllocationService) summary(
	ctx context.Context,
	db *gorm.DB,
	user *models.User,
	businessDate time.Time,
	attendance map[string]int,
	exceptReportID *int64,
	lockForUpdate bool,
) (map[string]int, map[string]int, error) {
	totalAttendance := models.NormalizeOperationalAttendance(attendance)

	allocated, err := s.allocatedForUser(ctx, db, user, businessDate, exceptReportID, lockForUpdate)
	if err != nil {
		return nil, nil, err
	}

	available := make(map[string]int, len(totalAttendance))
	for role, total := range totalAttendance {
		remaining := total - allocated[role]
		if remaining < 0 {
			remaining = 0
		}
		available[role] = remaining
	}

	return allocated, available, nil
}

func (s *AllocationService) allocatedForUser(
	ctx context.Context,
	db *gorm.DB,
	user *models.User,
	businessDate time.Time,
	exceptReportID *int64,
	lockForUpdate bool,
) (map[string]int, error) {
	start, end := DayRange(businessDate)

	query := db.WithContext(ctx).
		Model(&models.TaskReport{}).
		Where("user_id = ? AND status = ?", user.ID, models.TaskReportInProgress).
		Where("started_at BETWEEN ? AND ?", start, end)

	if exceptReportID != nil {
		query = query.Where("id <> ?", *exceptReportID)
	}

	if lockForUpdate {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var reports []models.TaskReport
	if err := query.Select("id", "member_allocations").Find(&reports).Error; err != nil {
		return nil, err
	}

	allocated := models.NormalizeOperationalAttendance(nil)
	for _, report := range reports {
		normalized := models.NormalizeOperationalAttendance(report.MemberAllocations)
		for role, value := range normalized {
			allocated[role] += value
		}
	}

	return allocated, nil
}
