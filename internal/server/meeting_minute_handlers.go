package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"agrinaspangan/ebitda-api/internal/meetingminutes"
	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
)

type meetingMinutePayload struct {
	Title       string                     `json:"title"`
	MeetingDate string                     `json:"meeting_date"`
	StartTime   *string                    `json:"start_time"`
	EndTime     *string                    `json:"end_time"`
	Location    *string                    `json:"location"`
	Attendees   *string                    `json:"attendees"`
	Items       []meetingMinuteItemPayload `json:"items"`
}

type meetingMinuteItemPayload struct {
	ID          *int64  `json:"id"`
	Subject     string  `json:"subject"`
	Description *string `json:"description"`
	Action      *string `json:"action"`
	Objectives  *string `json:"objectives"`
	DateStart   *string `json:"date_start"`
	DateFinish  *string `json:"date_finish"`
	PIC         *string `json:"pic"`
	Status      *string `json:"status"`
	Remarks     *string `json:"remarks"`
}

// ListMeetingMinutesHandler godoc
//
//	@Summary	Daftar Meeting Minutes milik Manager KDKMP
//	@Tags		Meeting Minutes
//	@Produce	json
//	@Security	CookieAuth
//	@Param		search	query	string	false	"Cari judul, lokasi, atau peserta"
//	@Success	200	{object}	map[string]any
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Router		/api/v1/meeting-minutes [get]
func ListMeetingMinutesHandler(c *gin.Context) {
	user, ok := meetingMinuteUser(c)
	if !ok {
		return
	}
	search, ok := meetingSearch(c)
	if !ok {
		return
	}

	meetings, err := AppDeps.Meetings.List(c.Request.Context(), user.ID, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat meeting minutes"})
		return
	}

	data := make([]gin.H, 0, len(meetings))
	for index := range meetings {
		data = append(data, meetingMinuteResponse(&meetings[index]))
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// CreateMeetingMinuteHandler godoc
//
//	@Summary	Buat Meeting Minutes milik Manager KDKMP
//	@Tags		Meeting Minutes
//	@Accept		json
//	@Produce	json
//	@Security	CookieAuth
//	@Param		payload	body	meetingMinutePayload	true	"Meeting Minutes dan item"
//	@Success	201	{object}	map[string]any
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	422	{object}	map[string]string
//	@Router		/api/v1/meeting-minutes [post]
func CreateMeetingMinuteHandler(c *gin.Context) {
	user, ok := meetingMinuteUser(c)
	if !ok {
		return
	}

	var payload meetingMinutePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data meeting minutes tidak valid"})
		return
	}
	input, err := parseMeetingMinutePayload(payload, false)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": err.Error()})
		return
	}

	meeting, err := AppDeps.Meetings.Create(c.Request.Context(), user, input)
	if !respondMeetingMinuteError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": meetingMinuteResponse(meeting)})
}

// GetMeetingMinuteHandler godoc
//
//	@Summary	Detail Meeting Minutes milik Manager KDKMP
//	@Tags		Meeting Minutes
//	@Produce	json
//	@Security	CookieAuth
//	@Param		id	path	int	true	"ID Meeting Minutes"
//	@Success	200	{object}	map[string]any
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Router		/api/v1/meeting-minutes/{id} [get]
func GetMeetingMinuteHandler(c *gin.Context) {
	user, ok := meetingMinuteUser(c)
	if !ok {
		return
	}
	meetingID, ok := meetingMinuteID(c)
	if !ok {
		return
	}

	meeting, err := AppDeps.Meetings.FindOwned(c.Request.Context(), user.ID, meetingID)
	if !respondMeetingMinuteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": meetingMinuteResponse(meeting)})
}

// UpdateMeetingMinuteHandler godoc
//
//	@Summary	Perbarui Meeting Minutes milik Manager KDKMP
//	@Tags		Meeting Minutes
//	@Accept		json
//	@Produce	json
//	@Security	CookieAuth
//	@Param		id		path	int			true	"ID Meeting Minutes"
//	@Param		payload	body	meetingMinutePayload	true	"Meeting Minutes dan snapshot item"
//	@Success	200	{object}	map[string]any
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Failure	422	{object}	map[string]string
//	@Router		/api/v1/meeting-minutes/{id} [put]
func UpdateMeetingMinuteHandler(c *gin.Context) {
	user, ok := meetingMinuteUser(c)
	if !ok {
		return
	}
	meetingID, ok := meetingMinuteID(c)
	if !ok {
		return
	}

	var payload meetingMinutePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data meeting minutes tidak valid"})
		return
	}
	input, err := parseMeetingMinutePayload(payload, true)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": err.Error()})
		return
	}

	meeting, err := AppDeps.Meetings.Update(c.Request.Context(), user, meetingID, input)
	if !respondMeetingMinuteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": meetingMinuteResponse(meeting)})
}

// DeleteMeetingMinuteHandler godoc
//
//	@Summary	Hapus Meeting Minutes milik Manager KDKMP
//	@Tags		Meeting Minutes
//	@Produce	json
//	@Security	CookieAuth
//	@Param		id	path	int	true	"ID Meeting Minutes"
//	@Success	204
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Router		/api/v1/meeting-minutes/{id} [delete]
func DeleteMeetingMinuteHandler(c *gin.Context) {
	user, ok := meetingMinuteUser(c)
	if !ok {
		return
	}
	meetingID, ok := meetingMinuteID(c)
	if !ok {
		return
	}

	if !respondMeetingMinuteError(c, AppDeps.Meetings.Delete(c.Request.Context(), user.ID, meetingID)) {
		return
	}
	c.Status(http.StatusNoContent)
}

// AddMeetingMinuteAttachmentsHandler godoc
//
//	@Summary	Unggah lampiran Meeting Minutes ke MinIO
//	@Tags		Meeting Minutes
//	@Accept		multipart/form-data
//	@Produce	json
//	@Security	CookieAuth
//	@Param		id		path		int		true	"ID Meeting Minutes"
//	@Param		documents	formData	[]file	true	"Lampiran (maksimal 10 file, 10 MB per file)"
//	@Success	201	{object}	map[string]any
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Failure	422	{object}	map[string]string
//	@Router		/api/v1/meeting-minutes/{id}/attachments [post]
func AddMeetingMinuteAttachmentsHandler(c *gin.Context) {
	user, ok := meetingMinuteUser(c)
	if !ok {
		return
	}
	meetingID, ok := meetingMinuteID(c)
	if !ok {
		return
	}
	form, ok := parseTaskReportForm(c)
	if !ok {
		return
	}
	documents, ok := validateDocumentUploads(c, form.File["documents"])
	if !ok || len(documents) == 0 {
		if ok {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Minimal satu lampiran wajib diunggah"})
		}
		return
	}

	attachments, err := AppDeps.Meetings.AddAttachments(c.Request.Context(), user.ID, meetingID, documents)
	if !respondMeetingMinuteError(c, err) {
		return
	}

	data := make([]gin.H, 0, len(attachments))
	for index := range attachments {
		data = append(data, meetingMinuteAttachmentResponse(meetingID, &attachments[index]))
	}
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

// DeleteMeetingMinuteAttachmentHandler godoc
//
//	@Summary	Hapus lampiran Meeting Minutes
//	@Tags		Meeting Minutes
//	@Produce	json
//	@Security	CookieAuth
//	@Param		id			path	int	true	"ID Meeting Minutes"
//	@Param		attachmentID	path	int	true	"ID lampiran"
//	@Success	204
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Router		/api/v1/meeting-minutes/{id}/attachments/{attachmentID} [delete]
func DeleteMeetingMinuteAttachmentHandler(c *gin.Context) {
	user, ok := meetingMinuteUser(c)
	if !ok {
		return
	}
	meetingID, ok := meetingMinuteID(c)
	if !ok {
		return
	}
	attachmentID, ok := meetingMinuteAttachmentID(c)
	if !ok {
		return
	}

	if !respondMeetingMinuteError(c, AppDeps.Meetings.DeleteAttachment(c.Request.Context(), user.ID, meetingID, attachmentID)) {
		return
	}
	c.Status(http.StatusNoContent)
}

// PreviewMeetingMinuteAttachmentHandler godoc
//
//	@Summary	Pratinjau lampiran Meeting Minutes
//	@Tags		Meeting Minutes
//	@Produce	application/octet-stream
//	@Security	CookieAuth
//	@Param		id			path	int	true	"ID Meeting Minutes"
//	@Param		attachmentID	path	int	true	"ID lampiran"
//	@Success	200	{file}	binary
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Router		/api/v1/meeting-minutes/{id}/attachments/{attachmentID}/preview [get]
func PreviewMeetingMinuteAttachmentHandler(c *gin.Context) {
	attachment, ok := meetingMinuteAttachment(c)
	if !ok {
		return
	}
	streamStoredDocument(c, models.StoredDocument{
		Disk:         attachment.Disk,
		Path:         attachment.Path,
		OriginalName: attachment.OriginalName,
		MimeType:     nullableValue(attachment.MimeType),
		Size:         attachment.Size,
	}, "inline")
}

// DownloadMeetingMinuteAttachmentHandler godoc
//
//	@Summary	Unduh lampiran Meeting Minutes
//	@Tags		Meeting Minutes
//	@Produce	application/octet-stream
//	@Security	CookieAuth
//	@Param		id			path	int	true	"ID Meeting Minutes"
//	@Param		attachmentID	path	int	true	"ID lampiran"
//	@Success	200	{file}	binary
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Router		/api/v1/meeting-minutes/{id}/attachments/{attachmentID}/download [get]
func DownloadMeetingMinuteAttachmentHandler(c *gin.Context) {
	attachment, ok := meetingMinuteAttachment(c)
	if !ok {
		return
	}
	streamStoredDocument(c, models.StoredDocument{
		Disk:         attachment.Disk,
		Path:         attachment.Path,
		OriginalName: attachment.OriginalName,
		MimeType:     nullableValue(attachment.MimeType),
		Size:         attachment.Size,
	}, "attachment")
}

func meetingMinuteUser(c *gin.Context) (*models.User, bool) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return nil, false
	}
	return user, true
}

func meetingMinuteID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Meeting minutes tidak ditemukan"})
		return 0, false
	}
	return id, true
}

func meetingMinuteAttachmentID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("attachmentID"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Lampiran tidak ditemukan"})
		return 0, false
	}
	return id, true
}

func meetingMinuteAttachment(c *gin.Context) (*models.MeetingMinuteAttachment, bool) {
	user, ok := meetingMinuteUser(c)
	if !ok {
		return nil, false
	}
	meetingID, ok := meetingMinuteID(c)
	if !ok {
		return nil, false
	}
	attachmentID, ok := meetingMinuteAttachmentID(c)
	if !ok {
		return nil, false
	}
	attachment, err := AppDeps.Meetings.FindAttachmentOwned(c.Request.Context(), user.ID, meetingID, attachmentID)
	if !respondMeetingMinuteError(c, err) {
		return nil, false
	}
	return attachment, true
}

func parseMeetingMinutePayload(payload meetingMinutePayload, allowExistingItems bool) (meetingminutes.Input, error) {
	title := strings.TrimSpace(payload.Title)
	if title == "" {
		return meetingminutes.Input{}, errors.New("Judul meeting wajib diisi")
	}
	if utf8.RuneCountInString(title) > 255 {
		return meetingminutes.Input{}, errors.New("Judul meeting maksimal 255 karakter")
	}

	meetingDate, err := parseMeetingDate(payload.MeetingDate, "Tanggal meeting wajib menggunakan format YYYY-MM-DD")
	if err != nil {
		return meetingminutes.Input{}, err
	}
	startTime, err := parseMeetingClockTime(payload.StartTime)
	if err != nil {
		return meetingminutes.Input{}, errors.New("Jam mulai wajib menggunakan format HH:MM")
	}
	endTime, err := parseMeetingClockTime(payload.EndTime)
	if err != nil {
		return meetingminutes.Input{}, errors.New("Jam selesai wajib menggunakan format HH:MM")
	}
	location, err := optionalMeetingText(payload.Location, 255, "Lokasi meeting maksimal 255 karakter")
	if err != nil {
		return meetingminutes.Input{}, err
	}

	items := make([]meetingminutes.ItemInput, 0, len(payload.Items))
	for index, item := range payload.Items {
		parsed, err := parseMeetingMinuteItem(item, allowExistingItems)
		if err != nil {
			return meetingminutes.Input{}, fmt.Errorf("Item %d: %w", index+1, err)
		}
		items = append(items, parsed)
	}

	return meetingminutes.Input{
		Title:       title,
		MeetingDate: meetingDate,
		StartTime:   startTime,
		EndTime:     endTime,
		Location:    location,
		Attendees:   optionalMeetingTextWithoutLimit(payload.Attendees),
		Items:       items,
	}, nil
}

func parseMeetingMinuteItem(payload meetingMinuteItemPayload, allowExisting bool) (meetingminutes.ItemInput, error) {
	if payload.ID != nil && (*payload.ID < 1 || !allowExisting) {
		return meetingminutes.ItemInput{}, errors.New("ID item tidak valid")
	}
	subject := strings.TrimSpace(payload.Subject)
	if subject == "" {
		return meetingminutes.ItemInput{}, errors.New("subjek wajib diisi")
	}
	if utf8.RuneCountInString(subject) > 255 {
		return meetingminutes.ItemInput{}, errors.New("subjek maksimal 255 karakter")
	}
	pic, err := optionalMeetingText(payload.PIC, 255, "PIC maksimal 255 karakter")
	if err != nil {
		return meetingminutes.ItemInput{}, err
	}
	dateStart, err := optionalMeetingDate(payload.DateStart, "Tanggal mulai harus menggunakan format YYYY-MM-DD")
	if err != nil {
		return meetingminutes.ItemInput{}, err
	}
	dateFinish, err := optionalMeetingDate(payload.DateFinish, "Tanggal selesai harus menggunakan format YYYY-MM-DD")
	if err != nil {
		return meetingminutes.ItemInput{}, err
	}
	status := ""
	if payload.Status != nil {
		status = strings.TrimSpace(*payload.Status)
		if status != "" && !models.IsMeetingMinuteItemStatus(status) {
			return meetingminutes.ItemInput{}, errors.New("status tidak valid")
		}
	}

	return meetingminutes.ItemInput{
		ID:          payload.ID,
		Subject:     subject,
		Description: optionalMeetingTextWithoutLimit(payload.Description),
		Action:      optionalMeetingTextWithoutLimit(payload.Action),
		Objectives:  optionalMeetingTextWithoutLimit(payload.Objectives),
		DateStart:   dateStart,
		DateFinish:  dateFinish,
		PIC:         pic,
		Status:      status,
		Remarks:     optionalMeetingTextWithoutLimit(payload.Remarks),
	}, nil
}

func parseMeetingDate(value, message string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, errors.New(message)
	}
	return date, nil
}

func optionalMeetingDate(value *string, message string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	date, err := parseMeetingDate(*value, message)
	if err != nil {
		return nil, err
	}
	return &date, nil
}

func parseMeetingClockTime(value *string) (*models.ClockTime, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse("15:04", strings.TrimSpace(*value))
	if err != nil {
		return nil, err
	}
	return &models.ClockTime{Hour: parsed.Hour(), Minute: parsed.Minute(), Valid: true}, nil
}

func optionalMeetingText(value *string, limit int, message string) (*string, error) {
	trimmed := optionalMeetingTextWithoutLimit(value)
	if trimmed != nil && utf8.RuneCountInString(*trimmed) > limit {
		return nil, errors.New(message)
	}
	return trimmed, nil
}

func optionalMeetingTextWithoutLimit(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func meetingSearch(c *gin.Context) (string, bool) {
	search := strings.TrimSpace(c.Query("search"))
	if utf8.RuneCountInString(search) > 255 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Pencarian maksimal 255 karakter"})
		return "", false
	}
	return search, true
}

func meetingMinuteResponse(meeting *models.MeetingMinute) gin.H {
	items := make([]gin.H, 0, len(meeting.Items))
	for index := range meeting.Items {
		item := &meeting.Items[index]
		items = append(items, gin.H{
			"id":          item.ID,
			"subject":     item.Subject,
			"description": item.Description,
			"action":      item.Action,
			"objectives":  item.Objectives,
			"date_start":  dateValue(item.DateStart),
			"date_finish": dateValue(item.DateFinish),
			"pic":         item.PIC,
			"status":      item.Status,
			"remarks":     item.Remarks,
			"sort_order":  item.SortOrder,
		})
	}
	attachments := make([]gin.H, 0, len(meeting.Attachments))
	for index := range meeting.Attachments {
		attachments = append(attachments, meetingMinuteAttachmentResponse(meeting.ID, &meeting.Attachments[index]))
	}

	return gin.H{
		"id":           meeting.ID,
		"title":        meeting.Title,
		"meeting_date": meeting.MeetingDate.Format("2006-01-02"),
		"start_time":   meeting.StartTime.String(),
		"end_time":     meeting.EndTime.String(),
		"location":     meeting.Location,
		"attendees":    meeting.Attendees,
		"items":        items,
		"attachments":  attachments,
		"created_at":   meeting.CreatedAt.Format(time.RFC3339),
		"updated_at":   meeting.UpdatedAt.Format(time.RFC3339),
	}
}

func meetingMinuteAttachmentResponse(meetingID int64, attachment *models.MeetingMinuteAttachment) gin.H {
	path := fmt.Sprintf("/api/v1/meeting-minutes/%d/attachments/%d", meetingID, attachment.ID)
	return gin.H{
		"id":           attachment.ID,
		"name":         attachment.OriginalName,
		"mime_type":    attachment.MimeType,
		"size":         attachment.Size,
		"preview_url":  path + "/preview",
		"download_url": path + "/download",
	}
}

func dateValue(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format("2006-01-02")
	return &formatted
}

func nullableValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func respondMeetingMinuteError(c *gin.Context, err error) bool {
	if err == nil {
		return true
	}
	switch {
	case errors.Is(err, meetingminutes.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "Meeting minutes tidak ditemukan"})
	case errors.Is(err, meetingminutes.ErrInvalidItem):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Item tidak terdaftar pada meeting minutes ini"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan meeting minutes"})
	}
	return false
}
