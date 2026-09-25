// Package meetingminutes menangani Meeting Minutes milik Manager KDKMP.
package meetingminutes

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/storage"
)

var (
	ErrNotFound    = errors.New("meeting minute tidak ditemukan")
	ErrInvalidItem = errors.New("item tidak terdaftar pada meeting minute")
	ErrNoChange    = errors.New("tidak ada perubahan action item")
)

// Input adalah data meeting minute yang sudah tervalidasi oleh handler.
type Input struct {
	Title       string
	MeetingDate time.Time
	StartTime   *models.ClockTime
	EndTime     *models.ClockTime
	Location    *string
	Attendees   *string
	Items       []ItemInput
}

// ItemInput adalah data satu item meeting minute yang sudah tervalidasi.
type ItemInput struct {
	ID          *int64
	Subject     string
	Description *string
	Action      *string
	Objectives  *string
	DateStart   *time.Time
	DateFinish  *time.Time
	PIC         *string
	Status      string
	Remarks     *string
}

// ActionItemFilters membatasi daftar tindak lanjut milik satu manager.
type ActionItemFilters struct {
	Search  string
	Status  string
	Overdue bool
	Page    int
	PerPage int
	Today   string
}

// ActionItemSummary adalah ringkasan status tanpa dipengaruhi filter daftar.
type ActionItemSummary struct {
	Total      int64
	Open       int64
	InProgress int64
	Completed  int64
	Overdue    int64
}

// ActionItemPage adalah halaman hasil dan ringkasan Action Items.
type ActionItemPage struct {
	Items      []models.MeetingMinuteItem
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
	Summary    ActionItemSummary
}

// Service menyimpan state meeting dan objek lampiran tanpa interface spekulatif.
type Service struct {
	db    *gorm.DB
	files *storage.Files
}

// NewService membuat service Meeting Minutes.
func NewService(db *gorm.DB, files *storage.Files) *Service {
	return &Service{db: db, files: files}
}

// List mengembalikan meeting minute milik satu manager.
func (s *Service) List(ctx context.Context, userID int64, search string) ([]models.MeetingMinute, error) {
	query := s.db.WithContext(ctx).
		Where("created_by = ?", userID).
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order").Order("id") }).
		Preload("Attachments", func(db *gorm.DB) *gorm.DB { return db.Order("created_at DESC").Order("id DESC") }).
		Order("meeting_date DESC").
		Order("created_at DESC")

	if search = strings.TrimSpace(search); search != "" {
		pattern := "%" + search + "%"
		query = query.Where("title ILIKE ? OR location ILIKE ? OR attendees ILIKE ?", pattern, pattern, pattern)
	}

	var meetings []models.MeetingMinute
	if err := query.Find(&meetings).Error; err != nil {
		return nil, err
	}
	return meetings, nil
}

// FindOwned mengembalikan meeting beserta item/lampirannya bila dimiliki manager.
func (s *Service) FindOwned(ctx context.Context, userID, meetingID int64) (*models.MeetingMinute, error) {
	var meeting models.MeetingMinute
	err := s.db.WithContext(ctx).
		Where("id = ? AND created_by = ?", meetingID, userID).
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order").Order("id") }).
		Preload("Attachments", func(db *gorm.DB) *gorm.DB { return db.Order("created_at DESC").Order("id DESC") }).
		First(&meeting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &meeting, nil
}

// Create menyimpan meeting dan seluruh item dalam satu transaksi database.
func (s *Service) Create(ctx context.Context, user *models.User, input Input) (*models.MeetingMinute, error) {
	var meeting models.MeetingMinute
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		meeting = models.MeetingMinute{
			Title:       input.Title,
			MeetingDate: input.MeetingDate,
			StartTime:   input.StartTime,
			EndTime:     input.EndTime,
			Location:    input.Location,
			Attendees:   input.Attendees,
			CreatedBy:   &user.ID,
		}
		if err := tx.Create(&meeting).Error; err != nil {
			return err
		}
		return syncItems(tx, meeting.ID, input.Items, false, nil)
	})
	if err != nil {
		return nil, err
	}
	return s.FindOwned(ctx, user.ID, meeting.ID)
}

// Update mengganti metadata dan snapshot item meeting milik manager secara atomik.
func (s *Service) Update(ctx context.Context, user *models.User, meetingID int64, input Input) (*models.MeetingMinute, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var meeting models.MeetingMinute
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND created_by = ?", meetingID, user.ID).
			First(&meeting).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		updates := map[string]any{
			"title":        input.Title,
			"meeting_date": input.MeetingDate,
			"start_time":   input.StartTime,
			"end_time":     input.EndTime,
			"location":     input.Location,
			"attendees":    input.Attendees,
			"updated_by":   user.ID,
		}
		if err := tx.Model(&meeting).Updates(updates).Error; err != nil {
			return err
		}
		return syncItems(tx, meeting.ID, input.Items, true, user)
	})
	if err != nil {
		return nil, err
	}
	return s.FindOwned(ctx, user.ID, meetingID)
}

// ListActionItems mengembalikan Action Items dan riwayat status milik manager.
func (s *Service) ListActionItems(ctx context.Context, userID int64, filters ActionItemFilters) (ActionItemPage, error) {
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PerPage < 1 {
		filters.PerPage = 15
	}

	query := s.actionItemsQuery(ctx, userID)
	if search := strings.TrimSpace(filters.Search); search != "" {
		pattern := "%" + search + "%"
		query = query.Where("meeting_minute_items.subject ILIKE ? OR meeting_minute_items.action ILIKE ? OR meeting_minute_items.pic ILIKE ? OR meeting_minutes.title ILIKE ?", pattern, pattern, pattern, pattern)
	}
	if filters.Status != "" {
		query = query.Where("meeting_minute_items.status = ?", filters.Status)
	}
	if filters.Overdue {
		query = query.Where("meeting_minute_items.date_finish < ? AND meeting_minute_items.status IN ?", filters.Today, []string{
			models.MeetingMinuteItemStatusOpen,
			models.MeetingMinuteItemStatusInProgress,
		})
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return ActionItemPage{}, err
	}

	var items []models.MeetingMinuteItem
	err := query.
		Select("meeting_minute_items.*").
		Preload("MeetingMinute").
		Preload("StatusHistories", func(db *gorm.DB) *gorm.DB { return db.Order("created_at DESC").Order("id DESC") }).
		Preload("StatusHistories.ChangedByUser").
		Order("meeting_minute_items.date_finish IS NULL").
		Order("meeting_minute_items.date_finish").
		Order("meeting_minute_items.id DESC").
		Offset((filters.Page - 1) * filters.PerPage).
		Limit(filters.PerPage).
		Find(&items).Error
	if err != nil {
		return ActionItemPage{}, err
	}

	summary, err := s.actionItemSummary(ctx, userID, filters.Today)
	if err != nil {
		return ActionItemPage{}, err
	}
	return ActionItemPage{
		Items:      items,
		Page:       filters.Page,
		PerPage:    filters.PerPage,
		Total:      total,
		TotalPages: max(1, int((total+int64(filters.PerPage)-1)/int64(filters.PerPage))),
		Summary:    summary,
	}, nil
}

// UpdateActionItemStatus mengunci item, menyimpan history, dan mengubah status.
func (s *Service) UpdateActionItemStatus(ctx context.Context, user *models.User, itemID int64, status string, remarks *string) (*models.MeetingMinuteItem, error) {
	var item models.MeetingMinuteItem
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: clause.CurrentTable}}).
			Joins("JOIN meeting_minutes ON meeting_minutes.id = meeting_minute_items.meeting_minute_id").
			Where("meeting_minute_items.id = ? AND meeting_minutes.created_by = ?", itemID, user.ID).
			First(&item).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if item.Status == status && sameText(item.Remarks, remarks) {
			return ErrNoChange
		}
		if err := tx.Create(&models.MeetingMinuteItemStatusHistory{
			MeetingMinuteItemID: item.ID,
			FromStatus:          item.Status,
			ToStatus:            status,
			Note:                remarks,
			ChangedBy:           &user.ID,
			ChangedByName:       user.Name,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&item).Updates(map[string]any{"status": status, "remarks": remarks}).Error; err != nil {
			return err
		}
		item.Status = status
		item.Remarks = remarks
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Delete menghapus meeting beserta child database dan objek lampirannya.
func (s *Service) Delete(ctx context.Context, userID, meetingID int64) error {
	var attachments []models.MeetingMinuteAttachment
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var meeting models.MeetingMinute
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND created_by = ?", meetingID, userID).
			First(&meeting).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := tx.Where("meeting_minute_id = ?", meeting.ID).Find(&attachments).Error; err != nil {
			return err
		}
		return tx.Delete(&meeting).Error
	})
	if err != nil {
		return err
	}
	s.removeAttachments(ctx, attachments)
	return nil
}

// AddAttachments menyimpan objek lebih dahulu lalu mencatat metadata secara atomik.
func (s *Service) AddAttachments(ctx context.Context, userID, meetingID int64, files []*multipart.FileHeader) ([]models.MeetingMinuteAttachment, error) {
	if err := s.ensureOwned(ctx, userID, meetingID); err != nil {
		return nil, err
	}

	attachments, err := s.storeAttachments(ctx, meetingID, userID, files)
	if err != nil {
		return nil, err
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var meeting models.MeetingMinute
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND created_by = ?", meetingID, userID).
			First(&meeting).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		return tx.Create(&attachments).Error
	})
	if err != nil {
		s.removeAttachments(ctx, attachments)
		return nil, err
	}
	return attachments, nil
}

// DeleteAttachment menghapus metadata lampiran milik manager lalu objek MinIO.
func (s *Service) DeleteAttachment(ctx context.Context, userID, meetingID, attachmentID int64) error {
	var attachment models.MeetingMinuteAttachment
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var meeting models.MeetingMinute
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND created_by = ?", meetingID, userID).
			First(&meeting).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		err = tx.Where("id = ? AND meeting_minute_id = ?", attachmentID, meeting.ID).First(&attachment).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		return tx.Delete(&attachment).Error
	})
	if err != nil {
		return err
	}
	s.removeAttachments(ctx, []models.MeetingMinuteAttachment{attachment})
	return nil
}

// FindAttachmentOwned memeriksa kepemilikan parent dan keterikatan lampiran.
func (s *Service) FindAttachmentOwned(ctx context.Context, userID, meetingID, attachmentID int64) (*models.MeetingMinuteAttachment, error) {
	var attachment models.MeetingMinuteAttachment
	err := s.db.WithContext(ctx).
		Table("meeting_minute_attachments AS attachments").
		Select("attachments.*").
		Joins("JOIN meeting_minutes AS meetings ON meetings.id = attachments.meeting_minute_id").
		Where("attachments.id = ? AND attachments.meeting_minute_id = ? AND meetings.created_by = ?", attachmentID, meetingID, userID).
		First(&attachment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

func (s *Service) ensureOwned(ctx context.Context, userID, meetingID int64) error {
	var meetingIDValue int64
	err := s.db.WithContext(ctx).
		Model(&models.MeetingMinute{}).
		Select("id").
		Where("id = ? AND created_by = ?", meetingID, userID).
		Scan(&meetingIDValue).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || meetingIDValue == 0 {
		return ErrNotFound
	}
	return err
}

func syncItems(tx *gorm.DB, meetingID int64, items []ItemInput, allowExisting bool, actor *models.User) error {
	var existing []models.MeetingMinuteItem
	if err := tx.Where("meeting_minute_id = ?", meetingID).Find(&existing).Error; err != nil {
		return err
	}

	existingByID := make(map[int64]models.MeetingMinuteItem, len(existing))
	for _, item := range existing {
		existingByID[item.ID] = item
	}

	incomingIDs := make([]int64, 0, len(items))
	seen := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if item.ID == nil {
			continue
		}
		if !allowExisting {
			return ErrInvalidItem
		}
		if _, exists := existingByID[*item.ID]; !exists {
			return ErrInvalidItem
		}
		if _, duplicate := seen[*item.ID]; duplicate {
			return ErrInvalidItem
		}
		seen[*item.ID] = struct{}{}
		incomingIDs = append(incomingIDs, *item.ID)
	}

	deleteQuery := tx.Where("meeting_minute_id = ?", meetingID)
	if len(incomingIDs) > 0 {
		deleteQuery = deleteQuery.Where("id NOT IN ?", incomingIDs)
	}
	if err := deleteQuery.Delete(&models.MeetingMinuteItem{}).Error; err != nil {
		return err
	}

	for index, input := range items {
		item := meetingMinuteItem(meetingID, index, input)
		if input.ID == nil {
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
			continue
		}
		previous := existingByID[*input.ID]
		if actor != nil && previous.Status != item.Status {
			if err := tx.Create(&models.MeetingMinuteItemStatusHistory{
				MeetingMinuteItemID: previous.ID,
				FromStatus:          previous.Status,
				ToStatus:            item.Status,
				Note:                item.Remarks,
				ChangedBy:           &actor.ID,
				ChangedByName:       actor.Name,
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&models.MeetingMinuteItem{}).
			Where("id = ? AND meeting_minute_id = ?", *input.ID, meetingID).
			Updates(map[string]any{
				"subject":     item.Subject,
				"description": item.Description,
				"action":      item.Action,
				"objectives":  item.Objectives,
				"date_start":  item.DateStart,
				"date_finish": item.DateFinish,
				"pic":         item.PIC,
				"status":      item.Status,
				"remarks":     item.Remarks,
				"sort_order":  item.SortOrder,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) actionItemsQuery(ctx context.Context, userID int64) *gorm.DB {
	return s.db.WithContext(ctx).
		Model(&models.MeetingMinuteItem{}).
		Joins("JOIN meeting_minutes ON meeting_minutes.id = meeting_minute_items.meeting_minute_id").
		Where("meeting_minutes.created_by = ?", userID)
}

func (s *Service) actionItemSummary(ctx context.Context, userID int64, today string) (ActionItemSummary, error) {
	var summary ActionItemSummary
	err := s.actionItemsQuery(ctx, userID).
		Select(`
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN meeting_minute_items.status = 'open' THEN 1 ELSE 0 END), 0) AS open,
			COALESCE(SUM(CASE WHEN meeting_minute_items.status = 'in_progress' THEN 1 ELSE 0 END), 0) AS in_progress,
			COALESCE(SUM(CASE WHEN meeting_minute_items.status = 'completed' THEN 1 ELSE 0 END), 0) AS completed,
			COALESCE(SUM(CASE WHEN meeting_minute_items.date_finish < ? AND meeting_minute_items.status IN ('open', 'in_progress') THEN 1 ELSE 0 END), 0) AS overdue`, today).
		Scan(&summary).Error
	return summary, err
}

func sameText(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func meetingMinuteItem(meetingID int64, index int, input ItemInput) models.MeetingMinuteItem {
	status := input.Status
	if status == "" {
		status = models.MeetingMinuteItemStatusOpen
	}
	return models.MeetingMinuteItem{
		MeetingMinuteID: meetingID,
		Subject:         input.Subject,
		Description:     input.Description,
		Action:          input.Action,
		Objectives:      input.Objectives,
		DateStart:       input.DateStart,
		DateFinish:      input.DateFinish,
		PIC:             input.PIC,
		Status:          status,
		Remarks:         input.Remarks,
		SortOrder:       index,
	}
}

func (s *Service) storeAttachments(ctx context.Context, meetingID, userID int64, files []*multipart.FileHeader) ([]models.MeetingMinuteAttachment, error) {
	if s.files == nil {
		return nil, fmt.Errorf("storage lampiran belum tersedia")
	}

	attachments := make([]models.MeetingMinuteAttachment, 0, len(files))
	for _, file := range files {
		extension := strings.ToLower(filepath.Ext(file.Filename))
		objectName := fmt.Sprintf("meeting-minutes/%d/%s%s", meetingID, uuid.NewString(), extension)
		contents, err := file.Open()
		if err != nil {
			s.removeAttachments(ctx, attachments)
			return nil, err
		}

		contentType := mime.TypeByExtension(extension)
		if contentType == "" {
			contentType = file.Header.Get("Content-Type")
		}
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		err = s.files.Put(ctx, objectName, contents, file.Size, contentType)
		_ = contents.Close()
		if err != nil {
			s.removeAttachments(ctx, attachments)
			return nil, err
		}

		attachments = append(attachments, models.MeetingMinuteAttachment{
			MeetingMinuteID: meetingID,
			Disk:            "minio",
			Path:            objectName,
			OriginalName:    filepath.Base(file.Filename),
			MimeType:        &contentType,
			Size:            file.Size,
			UploadedBy:      &userID,
		})
	}
	return attachments, nil
}

func (s *Service) removeAttachments(ctx context.Context, attachments []models.MeetingMinuteAttachment) {
	if s.files == nil {
		return
	}
	for _, attachment := range attachments {
		if attachment.Path != "" {
			_ = s.files.Remove(ctx, attachment.Path)
		}
	}
}
