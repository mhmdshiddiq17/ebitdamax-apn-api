package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"agrinaspangan/ebitda-api/internal/kdkmp"
	"agrinaspangan/ebitda-api/internal/meetingminutes"
	"agrinaspangan/ebitda-api/internal/models"
)

type updateMeetingActionItemPayload struct {
	Status  string  `json:"status"`
	Remarks *string `json:"remarks"`
}

// ListMeetingActionItemsHandler godoc
//
//	@Summary	Daftar Action Items milik Manager KDKMP
//	@Tags		Meeting Minutes
//	@Produce	json
//	@Security	CookieAuth
//	@Param		search	query	string	false	"Cari subjek, aksi, PIC, atau meeting"
//	@Param		status	query	string	false	"open|in_progress|completed|cancelled"
//	@Param		overdue	query	bool	false	"Hanya item terlambat"
//	@Param		page	query	int		false	"Halaman"
//	@Success	200	{object}	map[string]any
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	422	{object}	map[string]string
//	@Router		/api/v1/meeting-minutes/action-items [get]
func ListMeetingActionItemsHandler(c *gin.Context) {
	user, ok := meetingMinuteUser(c)
	if !ok {
		return
	}

	status := strings.TrimSpace(c.Query("status"))
	if status != "" && !models.IsMeetingMinuteItemStatus(status) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Status action item tidak valid"})
		return
	}
	search, ok := meetingSearch(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	today := kdkmp.DateString(kdkmp.BusinessDate())
	result, err := AppDeps.Meetings.ListActionItems(c.Request.Context(), user.ID, meetingminutes.ActionItemFilters{
		Search:  search,
		Status:  status,
		Overdue: c.Query("overdue") == "true" || c.Query("overdue") == "1",
		Page:    page,
		PerPage: 15,
		Today:   today,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat action items"})
		return
	}

	data := make([]gin.H, 0, len(result.Items))
	for index := range result.Items {
		data = append(data, actionItemResponse(&result.Items[index], today))
	}
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"page":        result.Page,
			"per_page":    result.PerPage,
			"total":       result.Total,
			"total_pages": result.TotalPages,
		},
		"summary": gin.H{
			"total":       result.Summary.Total,
			"open":        result.Summary.Open,
			"in_progress": result.Summary.InProgress,
			"completed":   result.Summary.Completed,
			"overdue":     result.Summary.Overdue,
		},
	})
}

// UpdateMeetingActionItemHandler godoc
//
//	@Summary	Perbarui status Action Item milik Manager KDKMP
//	@Tags		Meeting Minutes
//	@Accept		json
//	@Produce	json
//	@Security	CookieAuth
//	@Param		itemID	path	int					true	"ID Action Item"
//	@Param		payload	body	updateMeetingActionItemPayload	true	"Status dan catatan"
//	@Success	200	{object}	map[string]any
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Failure	422	{object}	map[string]string
//	@Router		/api/v1/meeting-minutes/action-items/{itemID} [patch]
func UpdateMeetingActionItemHandler(c *gin.Context) {
	user, ok := meetingMinuteUser(c)
	if !ok {
		return
	}
	itemID, err := strconv.ParseInt(c.Param("itemID"), 10, 64)
	if err != nil || itemID < 1 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Action item tidak ditemukan"})
		return
	}

	var payload updateMeetingActionItemPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data action item tidak valid"})
		return
	}
	status := strings.TrimSpace(payload.Status)
	if !models.IsMeetingMinuteItemStatus(status) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Status action item tidak valid"})
		return
	}
	remarks := optionalMeetingTextWithoutLimit(payload.Remarks)
	if remarks != nil && utf8.RuneCountInString(*remarks) > 5000 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Catatan maksimal 5.000 karakter"})
		return
	}

	item, err := AppDeps.Meetings.UpdateActionItemStatus(c.Request.Context(), user, itemID, status, remarks)
	if err != nil {
		switch {
		case errors.Is(err, meetingminutes.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"message": "Action item tidak ditemukan"})
		case errors.Is(err, meetingminutes.ErrNoChange):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Tidak ada perubahan status atau catatan untuk disimpan"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memperbarui action item"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": actionItemResponse(item, kdkmp.DateString(kdkmp.BusinessDate()))})
}

func actionItemResponse(item *models.MeetingMinuteItem, today string) gin.H {
	histories := make([]gin.H, 0, len(item.StatusHistories))
	for index := range item.StatusHistories {
		history := &item.StatusHistories[index]
		changedByName := history.ChangedByName
		if history.ChangedByUser != nil && history.ChangedByUser.Name != "" {
			changedByName = history.ChangedByUser.Name
		}
		histories = append(histories, gin.H{
			"id":              history.ID,
			"from_status":     history.FromStatus,
			"to_status":       history.ToStatus,
			"note":            history.Note,
			"changed_by_name": changedByName,
			"created_at":      history.CreatedAt.Format(time.RFC3339),
		})
	}

	meeting := gin.H{}
	if item.MeetingMinute != nil {
		meeting = gin.H{
			"id":           item.MeetingMinute.ID,
			"title":        item.MeetingMinute.Title,
			"meeting_date": item.MeetingMinute.MeetingDate.Format("2006-01-02"),
		}
	}

	return gin.H{
		"id":               item.ID,
		"subject":          item.Subject,
		"action":           item.Action,
		"pic":              item.PIC,
		"date_start":       dateValue(item.DateStart),
		"date_finish":      dateValue(item.DateFinish),
		"status":           item.Status,
		"remarks":          item.Remarks,
		"is_overdue":       actionItemIsOverdue(item, today),
		"meeting_minute":   meeting,
		"status_histories": histories,
	}
}

func actionItemIsOverdue(item *models.MeetingMinuteItem, today string) bool {
	return item.DateFinish != nil &&
		item.DateFinish.Format("2006-01-02") < today &&
		(item.Status == models.MeetingMinuteItemStatusOpen || item.Status == models.MeetingMinuteItemStatusInProgress)
}
