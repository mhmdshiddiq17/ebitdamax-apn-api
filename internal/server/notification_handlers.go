package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
)

const announcementNotificationType = "announcement"

type announcementPayload struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

type notificationData struct {
	Title      string `json:"title"`
	Message    string `json:"message"`
	SenderName string `json:"sender_name"`
}

// ListNotificationsHandler godoc
//
//	@Summary	Daftar notifikasi Manager KDKMP
//	@Tags		Notifications
//	@Produce	json
//	@Security	CookieAuth
//	@Param		limit	query	int	false	"Maksimal 100, default 100"
//	@Success	200	{object}	map[string]any
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Router		/api/v1/notifications [get]
func ListNotificationsHandler(c *gin.Context) {
	user, ok := notificationUser(c)
	if !ok {
		return
	}
	limit := notificationLimit(c)
	query := ownedNotifications(c, user.ID)

	var rows []models.Notification
	if err := query.Order("created_at DESC").Order("id DESC").Limit(limit).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat notifikasi"})
		return
	}
	var unreadCount int64
	if err := ownedNotifications(c, user.ID).Where("read_at IS NULL").Count(&unreadCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat notifikasi"})
		return
	}

	data := make([]gin.H, 0, len(rows))
	for index := range rows {
		data = append(data, notificationResponse(&rows[index]))
	}
	c.JSON(http.StatusOK, gin.H{"data": data, "meta": gin.H{"unread_count": unreadCount}})
}

// MarkNotificationReadHandler godoc
//
//	@Summary	Tandai satu notifikasi Manager KDKMP sebagai dibaca
//	@Tags		Notifications
//	@Produce	json
//	@Security	CookieAuth
//	@Param		id	path	string	true	"ID notifikasi"
//	@Success	200	{object}	map[string]any
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Router		/api/v1/notifications/{id}/read [patch]
func MarkNotificationReadHandler(c *gin.Context) {
	user, ok := notificationUser(c)
	if !ok {
		return
	}
	var notification models.Notification
	err := ownedNotifications(c, user.ID).Where("id = ?", c.Param("id")).First(&notification).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Notifikasi tidak ditemukan"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat notifikasi"})
		return
	}
	if notification.ReadAt == nil {
		now := time.Now()
		notification.ReadAt = &now
		if err := AppDeps.DB.WithContext(c.Request.Context()).Model(&notification).Update("read_at", now).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memperbarui notifikasi"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": notificationResponse(&notification)})
}

// MarkAllNotificationsReadHandler godoc
//
//	@Summary	Tandai seluruh notifikasi Manager KDKMP sebagai dibaca
//	@Tags		Notifications
//	@Produce	json
//	@Security	CookieAuth
//	@Success	200	{object}	map[string]string
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Router		/api/v1/notifications/read-all [patch]
func MarkAllNotificationsReadHandler(c *gin.Context) {
	user, ok := notificationUser(c)
	if !ok {
		return
	}
	if err := ownedNotifications(c, user.ID).Where("read_at IS NULL").Update("read_at", time.Now()).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memperbarui notifikasi"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Seluruh notifikasi telah ditandai dibaca."})
}

// CreateAnnouncementHandler godoc
//
//	@Summary	Kirim pengumuman kepada seluruh Manager KDKMP
//	@Tags		Announcements
//	@Accept		json
//	@Produce	json
//	@Security	CookieAuth
//	@Param		payload	body	announcementPayload	true	"Judul dan pesan pengumuman"
//	@Success	201		{object}	map[string]any
//	@Failure	401		{object}	map[string]string
//	@Failure	403		{object}	map[string]string
//	@Failure	422		{object}	map[string]string
//	@Router		/api/v1/announcements [post]
func CreateAnnouncementHandler(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	if actor == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}
	var payload announcementPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data pengumuman tidak valid"})
		return
	}
	title, err := requiredAnnouncementText(payload.Title, "Judul pengumuman wajib diisi", 255)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": err.Error()})
		return
	}
	message, err := requiredAnnouncementText(payload.Message, "Pesan pengumuman wajib diisi", 5000)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": err.Error()})
		return
	}

	var recipients []models.User
	if err := AppDeps.DB.WithContext(c.Request.Context()).
		Model(&models.User{}).
		Joins("JOIN roles ON roles.id = users.role_id").
		Where("roles.domain = ? AND roles.slug = ?", models.RoleDomainKdkmp, models.RoleSlugManager).
		Find(&recipients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat penerima pengumuman"})
		return
	}
	if len(recipients) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Belum ada Manager KDKMP sebagai penerima pengumuman"})
		return
	}

	data, err := json.Marshal(notificationData{Title: title, Message: message, SenderName: actor.Name})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyiapkan pengumuman"})
		return
	}
	now := time.Now()
	notifications := make([]models.Notification, 0, len(recipients))
	for _, recipient := range recipients {
		notifications = append(notifications, models.Notification{ID: uuid.NewString(), Type: announcementNotificationType, NotifiableType: models.UserNotifiableType, NotifiableID: recipient.ID, Data: string(data), CreatedAt: now, UpdatedAt: now})
	}
	if err := AppDeps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		return tx.CreateInBatches(notifications, 100).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mengirim pengumuman"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Pengumuman berhasil dikirimkan.", "recipients_count": len(recipients)})
}

func notificationUser(c *gin.Context) (*models.User, bool) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return nil, false
	}
	return user, true
}

func ownedNotifications(c *gin.Context, userID int64) *gorm.DB {
	return AppDeps.DB.WithContext(c.Request.Context()).Model(&models.Notification{}).Where("notifiable_type = ? AND notifiable_id = ?", models.UserNotifiableType, userID)
}

func notificationLimit(c *gin.Context) int {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil || limit < 1 {
		return 100
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func requiredAnnouncementText(value, requiredMessage string, maximum int) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errors.New(requiredMessage)
	}
	if utf8.RuneCountInString(trimmed) > maximum {
		return "", errors.New("Teks melebihi batas karakter")
	}
	return trimmed, nil
}

func notificationResponse(notification *models.Notification) gin.H {
	data := notificationData{Title: "Pengumuman"}
	_ = json.Unmarshal([]byte(notification.Data), &data)
	if strings.TrimSpace(data.Title) == "" {
		data.Title = "Pengumuman"
	}
	return gin.H{"id": notification.ID, "title": data.Title, "message": data.Message, "sender_name": data.SenderName, "created_at": notification.CreatedAt, "read_at": notification.ReadAt}
}
