package server

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"agrinaspangan/ebitda-api/internal/kdkmp"
	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
)

// PreviewTaskReportDocumentHandler godoc
//
//	@Summary      Pratinjau dokumen laporan task
//	@Description  Menampilkan dokumen laporan task (fase start/finish) secara inline.
//	@Tags         Task Reports
//	@Produce      application/octet-stream
//	@Security     CookieAuth
//	@Param        id     path  int     true  "ID laporan"
//	@Param        phase  path  string  true  "start|finish"
//	@Param        index  path  int     true  "Indeks dokumen"
//	@Success      200  {file}  binary
//	@Failure      403  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Router       /api/v1/task-reports/{id}/documents/{phase}/{index}/preview [get]
func PreviewTaskReportDocumentHandler(c *gin.Context) {
	document, ok := taskReportDocument(c)
	if !ok {
		return
	}
	streamStoredDocument(c, *document, "inline")
}

// DownloadTaskReportDocumentHandler godoc
//
//	@Summary      Unduh dokumen laporan task
//	@Description  Mengunduh dokumen laporan task (fase start/finish).
//	@Tags         Task Reports
//	@Produce      application/octet-stream
//	@Security     CookieAuth
//	@Param        id     path  int     true  "ID laporan"
//	@Param        phase  path  string  true  "start|finish"
//	@Param        index  path  int     true  "Indeks dokumen"
//	@Success      200  {file}  binary
//	@Failure      403  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Router       /api/v1/task-reports/{id}/documents/{phase}/{index}/download [get]
func DownloadTaskReportDocumentHandler(c *gin.Context) {
	document, ok := taskReportDocument(c)
	if !ok {
		return
	}
	streamStoredDocument(c, *document, "attachment")
}

// PreviewTaskReportPhotoHandler godoc
//
//	@Summary      Pratinjau foto laporan task
//	@Description  Menampilkan foto mulai/selesai task secara inline.
//	@Tags         Task Reports
//	@Produce      image/jpeg
//	@Security     CookieAuth
//	@Param        id     path  int     true  "ID laporan"
//	@Param        phase  path  string  true  "start|finish"
//	@Success      200  {file}  binary
//	@Failure      403  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Router       /api/v1/task-reports/{id}/photos/{phase}/preview [get]
func PreviewTaskReportPhotoHandler(c *gin.Context) {
	document, ok := taskReportPhoto(c)
	if !ok {
		return
	}
	streamStoredDocument(c, *document, "inline")
}

// DownloadTaskReportPhotoHandler godoc
//
//	@Summary      Unduh foto laporan task
//	@Description  Mengunduh foto mulai/selesai task.
//	@Tags         Task Reports
//	@Produce      application/octet-stream
//	@Security     CookieAuth
//	@Param        id     path  int     true  "ID laporan"
//	@Param        phase  path  string  true  "start|finish"
//	@Success      200  {file}  binary
//	@Failure      403  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Router       /api/v1/task-reports/{id}/photos/{phase}/download [get]
func DownloadTaskReportPhotoHandler(c *gin.Context) {
	document, ok := taskReportPhoto(c)
	if !ok {
		return
	}
	streamStoredDocument(c, *document, "attachment")
}

// PreviewTaskReportAdditionalFieldHandler godoc
//
//	@Summary      Pratinjau file field tambahan
//	@Description  Menampilkan file yang diunggah pada field tambahan laporan task.
//	@Tags         Task Reports
//	@Produce      application/octet-stream
//	@Security     CookieAuth
//	@Param        id       path  int  true  "ID laporan"
//	@Param        valueId  path  int  true  "ID nilai field"
//	@Success      200  {file}  binary
//	@Failure      403  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Router       /api/v1/task-reports/{id}/additional-fields/{valueId}/preview [get]
func PreviewTaskReportAdditionalFieldHandler(c *gin.Context) {
	document, ok := taskReportAdditionalFieldDocument(c)
	if !ok {
		return
	}
	streamStoredDocument(c, *document, "inline")
}

// DownloadTaskReportAdditionalFieldHandler godoc
//
//	@Summary      Unduh file field tambahan
//	@Description  Mengunduh file yang diunggah pada field tambahan laporan task.
//	@Tags         Task Reports
//	@Produce      application/octet-stream
//	@Security     CookieAuth
//	@Param        id       path  int  true  "ID laporan"
//	@Param        valueId  path  int  true  "ID nilai field"
//	@Success      200  {file}  binary
//	@Failure      403  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Router       /api/v1/task-reports/{id}/additional-fields/{valueId}/download [get]
func DownloadTaskReportAdditionalFieldHandler(c *gin.Context) {
	document, ok := taskReportAdditionalFieldDocument(c)
	if !ok {
		return
	}
	streamStoredDocument(c, *document, "attachment")
}

func taskReportDocument(c *gin.Context) (*models.StoredDocument, bool) {
	report, ok := findTaskReportForView(c)
	if !ok {
		return nil, false
	}

	phase := c.Param("phase")
	documents, ok := reportDocumentsByPhase(report, phase)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"message": "Dokumen tidak ditemukan"})
		return nil, false
	}

	index, err := strconv.Atoi(c.Param("index"))
	if err != nil || index < 0 || index >= len(documents) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Dokumen tidak ditemukan"})
		return nil, false
	}

	return &documents[index], true
}

func taskReportPhoto(c *gin.Context) (*models.StoredDocument, bool) {
	report, ok := findTaskReportForView(c)
	if !ok {
		return nil, false
	}

	phase := c.Param("phase")
	var path *string
	switch phase {
	case "start":
		path = report.StartedPhoto
	case "finish":
		path = report.FinishedPhoto
	default:
		c.JSON(http.StatusNotFound, gin.H{"message": "Foto tidak ditemukan"})
		return nil, false
	}

	if path == nil || *path == "" {
		c.JSON(http.StatusNotFound, gin.H{"message": "Foto tidak ditemukan"})
		return nil, false
	}

	document := models.StoredDocument{
		Disk:         "minio",
		Path:         *path,
		OriginalName: filepath.Base(*path),
		MimeType:     mimeByExtension(*path),
	}

	return &document, true
}

func taskReportAdditionalFieldDocument(c *gin.Context) (*models.StoredDocument, bool) {
	report, ok := findTaskReportForView(c)
	if !ok {
		return nil, false
	}

	valueID, err := strconv.ParseInt(c.Param("valueId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "File tidak ditemukan"})
		return nil, false
	}

	var value models.TaskReportValue
	err = AppDeps.DB.WithContext(c.Request.Context()).
		Where("id = ? AND task_report_id = ?", valueID, report.ID).
		First(&value).Error
	if err != nil || value.Value == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "File tidak ditemukan"})
		return nil, false
	}

	var document models.StoredDocument
	if err := json.Unmarshal([]byte(*value.Value), &document); err != nil || document.Path == "" {
		c.JSON(http.StatusNotFound, gin.H{"message": "File tidak ditemukan"})
		return nil, false
	}

	return &document, true
}

func findTaskReportForView(c *gin.Context) (*models.TaskReport, bool) {
	reportID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Laporan tidak ditemukan"})
		return nil, false
	}

	var report models.TaskReport
	err = AppDeps.DB.WithContext(c.Request.Context()).First(&report, reportID).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Laporan tidak ditemukan"})
		return nil, false
	}

	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return nil, false
	}

	allowed, err := kdkmp.CanViewTaskReport(c.Request.Context(), AppDeps.DB, user, &report)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa akses laporan"})
		return nil, false
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"message": "Anda tidak memiliki akses"})
		return nil, false
	}

	return &report, true
}

func reportDocumentsByPhase(report *models.TaskReport, phase string) (models.StoredDocuments, bool) {
	switch phase {
	case "start":
		return report.StartedDocuments, true
	case "finish":
		return report.FinishedDocuments, true
	default:
		return nil, false
	}
}

func streamStoredDocument(c *gin.Context, document models.StoredDocument, disposition string) {
	object, err := AppDeps.Files.Stream(c.Request.Context(), document.Path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Berkas tidak ditemukan"})
		return
	}
	defer func() { _ = object.Close() }()

	info, err := object.Stat()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Berkas tidak ditemukan"})
		return
	}

	contentType := document.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	filename := document.OriginalName
	if filename == "" {
		filename = filepath.Base(document.Path)
	}

	c.DataFromReader(http.StatusOK, info.Size, contentType, object, map[string]string{
		"Content-Disposition":    fmt.Sprintf("%s; filename=%q", disposition, filename),
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "private, max-age=300",
	})
}

func mimeByExtension(path string) string {
	extension := strings.ToLower(filepath.Ext(path))
	if contentType := mime.TypeByExtension(extension); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}
