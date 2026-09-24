package server

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/kdkmp"
	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
)

const historyPerPage = 15

// TaskHistoryHandler godoc
//
//	@Summary      Riwayat task selesai
//	@Description  Riwayat laporan task selesai 14 hari terakhir, dikelompokkan per tanggal dengan ringkasan harian (tepat waktu, terlambat, tidak dikerjakan).
//	@Tags         Task Dashboard
//	@Produce      json
//	@Security     CookieAuth
//	@Param        page  query  int  false  "Halaman (per 15 hari)"
//	@Success      200   {object}  map[string]any
//	@Failure      401   {object}  map[string]string
//	@Failure      403   {object}  map[string]string
//	@Router       /api/v1/task-dashboard/completed [get]
func TaskHistoryHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	ctx := c.Request.Context()
	location := kdkmp.Location()
	now := time.Now().In(location)
	historyStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location).AddDate(0, 0, -13)
	historyEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, location)

	isSuperadmin := user.IsSuperadmin()

	query := AppDeps.DB.WithContext(ctx).
		Model(&models.TaskReport{}).
		Preload("Task.TaskCategory").
		Preload("Task.Roles").
		Preload("User.Role").
		Where("status = ?", models.TaskReportCompleted).
		Where("finished_at IS NOT NULL").
		Where("finished_at BETWEEN ? AND ?", historyStart.UTC(), historyEnd.UTC())

	if !isSuperadmin {
		query = query.Where("user_id = ?", user.ID)
	}

	var reports []models.TaskReport
	if err := query.Order("finished_at DESC").Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat riwayat task"})
		return
	}

	tasksByRole, err := activeTasksByRoleQuery(AppDeps.DB.WithContext(ctx))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat task aktif"})
		return
	}

	// Pilihan task KDKMP per entry & tanggal (untuk perhitungan task yang tidak dikerjakan).
	entryIDs := make([]int64, 0)
	seenEntries := make(map[int64]bool)
	usersByID := make(map[int64]*models.User)
	for i := range reports {
		report := &reports[i]
		if report.User != nil {
			usersByID[report.User.ID] = report.User
			if report.User.IsKdkmpManager() && report.User.SDMKdkmpEntryID != nil && !seenEntries[*report.User.SDMKdkmpEntryID] {
				seenEntries[*report.User.SDMKdkmpEntryID] = true
				entryIDs = append(entryIDs, *report.User.SDMKdkmpEntryID)
			}
		}
	}

	selectedByEntryDate, err := AppDeps.Selection.DailySelectedTaskIDsByKdkmpEntryAndDate(ctx, entryIDs, historyStart, historyEnd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat pemilihan task"})
		return
	}

	reportsByDate := make(map[string][]models.TaskReport)
	for _, report := range reports {
		if report.FinishedAt == nil {
			continue
		}
		date := report.FinishedAt.In(location).Format("2006-01-02")
		reportsByDate[date] = append(reportsByDate[date], report)
	}

	dates := make([]string, 0, len(reportsByDate))
	for date := range reportsByDate {
		dates = append(dates, date)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	total := len(dates)
	totalPages := int((int64(total) + historyPerPage - 1) / historyPerPage)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	startIndex := (page - 1) * historyPerPage
	endIndex := startIndex + historyPerPage
	if startIndex > total {
		startIndex = total
	}
	if endIndex > total {
		endIndex = total
	}

	data := make([]gin.H, 0, endIndex-startIndex)
	for _, date := range dates[startIndex:endIndex] {
		data = append(data, transformTaskDailySummary(
			date,
			reportsByDate[date],
			user,
			isSuperadmin,
			tasksByRole,
			selectedByEntryDate,
			usersByID,
		))
	}

	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"page":        page,
			"per_page":    historyPerPage,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func transformTaskDailySummary(
	date string,
	dayReports []models.TaskReport,
	user *models.User,
	isSuperadmin bool,
	tasksByRole map[int64][]models.Task,
	selectedByEntryDate map[string]map[int64]bool,
	usersByID map[int64]*models.User,
) gin.H {
	expected := expectedTasksForDay(date, dayReports, user, isSuperadmin, tasksByRole, selectedByEntryDate, usersByID)

	completedReports := make([]models.TaskReport, 0, len(dayReports))
	seen := make(map[string]bool)
	for _, report := range dayReports {
		key := taskAssignmentKey(report.UserID, report.TaskID)
		if seen[key] {
			continue
		}
		seen[key] = true
		completedReports = append(completedReports, report)
	}

	onTime := make([]models.TaskReport, 0)
	late := make([]models.TaskReport, 0)
	for _, report := range completedReports {
		if isTaskReportOnTime(&report) {
			onTime = append(onTime, report)
			continue
		}
		late = append(late, report)
	}

	notWorked := make([]gin.H, 0)
	for key, assignment := range expected {
		if seen[key] {
			continue
		}
		notWorked = append(notWorked, gin.H{
			"task": taskReference(assignment.task),
			"user": userReference(assignment.user),
		})
	}

	completed := make([]gin.H, 0, len(completedReports))
	for i := range completedReports {
		completed = append(completed, completedReportResponse(&completedReports[i], isSuperadmin))
	}

	return gin.H{
		"date":              date,
		"total_tasks":       len(expected),
		"on_time_tasks":     len(onTime),
		"late_tasks":        len(late),
		"not_worked_tasks":  notWorked,
		"completed_reports": completed,
	}
}

type taskAssignment struct {
	userID int64
	task   *models.Task
	user   *models.User
}

func expectedTasksForDay(
	date string,
	dayReports []models.TaskReport,
	user *models.User,
	isSuperadmin bool,
	tasksByRole map[int64][]models.Task,
	selectedByEntryDate map[string]map[int64]bool,
	usersByID map[int64]*models.User,
) map[string]taskAssignment {
	expected := make(map[string]taskAssignment)

	userIDs := make([]int64, 0)
	if isSuperadmin {
		for userID := range usersByID {
			userIDs = append(userIDs, userID)
		}
	} else if user.RoleID != nil {
		userIDs = append(userIDs, user.ID)
	}

	for _, userID := range userIDs {
		participant := usersByID[userID]
		roleID := (*int64)(nil)
		if isSuperadmin {
			if participant != nil {
				roleID = participant.RoleID
			}
		} else {
			roleID = user.RoleID
		}
		if roleID == nil {
			continue
		}

		assigned := tasksByRole[*roleID]
		manager := user
		if isSuperadmin {
			manager = participant
		}

		if manager != nil && manager.IsKdkmpManager() && manager.SDMKdkmpEntryID != nil {
			selected := selectedByEntryDate[fmt.Sprintf("%d|%s", *manager.SDMKdkmpEntryID, date)]
			filtered := make([]models.Task, 0, len(assigned))
			for i := range assigned {
				task := assigned[i]
				if task.IsMandatory || selected[task.ID] {
					filtered = append(filtered, task)
				}
			}
			assigned = filtered
		}

		for i := range assigned {
			task := assigned[i]
			assignmentUser := (*models.User)(nil)
			if isSuperadmin {
				assignmentUser = participant
			}
			expected[taskAssignmentKey(userID, task.ID)] = taskAssignment{
				userID: userID,
				task:   &task,
				user:   assignmentUser,
			}
		}
	}

	return expected
}

func activeTasksByRoleQuery(db *gorm.DB) (map[int64][]models.Task, error) {
	var tasks []models.Task
	err := db.
		Preload("TaskCategory").
		Preload("Roles").
		Where("is_active = ?", true).
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}

	byRole := make(map[int64][]models.Task)
	for i := range tasks {
		for _, role := range tasks[i].Roles {
			byRole[role.ID] = append(byRole[role.ID], tasks[i])
		}
	}

	return byRole, nil
}

func taskAssignmentKey(userID int64, taskID int64) string {
	return fmt.Sprintf("%d|%d", userID, taskID)
}

func isTaskReportOnTime(report *models.TaskReport) bool {
	return report.DurationMinutes != nil && report.Task != nil && *report.DurationMinutes <= report.Task.TimeRequire
}

func completedReportResponse(report *models.TaskReport, isSuperadmin bool) gin.H {
	onTime := isTaskReportOnTime(report)

	response := gin.H{
		"id":                    report.ID,
		"uuid":                  report.UUID,
		"started_at":            report.StartedAt,
		"finished_at":           report.FinishedAt,
		"duration_minutes":      report.DurationMinutes,
		"manager_self_assigned": report.ManagerSelfAssigned,
		"status_label":          models.TaskReportStatusLabel(report.Status),
		"timing_status":         map[bool]string{true: "on_time", false: "late"}[onTime],
		"timing_label":          map[bool]string{true: "Tepat Waktu", false: "Terlambat"}[onTime],
		"documents":             transformReportDocuments(report),
		"task":                  nil,
		"user":                  nil,
	}

	if report.Task != nil {
		response["task"] = taskReference(report.Task)
	}
	if isSuperadmin {
		response["user"] = userReference(report.User)
	}

	return response
}

func taskReference(task *models.Task) gin.H {
	roles := make([]gin.H, 0, len(task.Roles))
	for _, role := range task.Roles {
		roles = append(roles, roleSummary(role))
	}

	response := gin.H{
		"id":            task.ID,
		"uuid":          task.UUID,
		"name":          task.Name,
		"description":   task.Description,
		"time_require":  task.TimeRequire,
		"roles":         roles,
		"task_category": nil,
	}

	if task.TaskCategory != nil {
		response["task_category"] = gin.H{
			"id":   task.TaskCategory.ID,
			"name": task.TaskCategory.Name,
			"slug": task.TaskCategory.Slug,
		}
	}

	return response
}

func userReference(user *models.User) gin.H {
	if user == nil {
		return nil
	}

	return gin.H{
		"id":       user.ID,
		"name":     user.Name,
		"username": user.Username,
		"email":    user.Email,
	}
}
