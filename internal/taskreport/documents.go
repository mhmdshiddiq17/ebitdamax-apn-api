// Package taskreport menangani penyimpanan dokumen laporan task ke MinIO.
package taskreport

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/storage"
)

// DocumentService menyimpan dokumen task ke object storage.
type DocumentService struct {
	files *storage.Files
}

// NewDocumentService membuat service dokumen task.
func NewDocumentService(files *storage.Files) *DocumentService {
	return &DocumentService{files: files}
}

// StoreDocuments menyimpan daftar dokumen untuk fase start/finish.
func (s *DocumentService) StoreDocuments(
	ctx context.Context,
	reportUUID string,
	phase string,
	files []*multipart.FileHeader,
) ([]models.StoredDocument, error) {
	return s.store(ctx, files, fmt.Sprintf("task-reports/%s/%s/documents", reportUUID, phase))
}

// StoreAdditionalField menyimpan file untuk satu field tambahan.
func (s *DocumentService) StoreAdditionalField(
	ctx context.Context,
	reportUUID string,
	fieldUUID string,
	phase string,
	file *multipart.FileHeader,
) (models.StoredDocument, error) {
	documents, err := s.store(
		ctx,
		[]*multipart.FileHeader{file},
		fmt.Sprintf("task-reports/%s/%s/additional-fields/%s", reportUUID, phase, fieldUUID),
	)
	if err != nil {
		return models.StoredDocument{}, err
	}
	if len(documents) == 0 {
		return models.StoredDocument{}, fmt.Errorf("file field tambahan gagal disimpan")
	}
	return documents[0], nil
}

// RemoveAll menghapus dokumen (dipakai rollback saat transaksi gagal).
func (s *DocumentService) RemoveAll(ctx context.Context, documents []models.StoredDocument) {
	for _, document := range documents {
		if document.Path != "" {
			_ = s.files.Remove(ctx, document.Path)
		}
	}
}

func (s *DocumentService) store(
	ctx context.Context,
	files []*multipart.FileHeader,
	directory string,
) ([]models.StoredDocument, error) {
	stored := make([]models.StoredDocument, 0, len(files))

	for _, file := range files {
		extension := strings.ToLower(filepath.Ext(file.Filename))
		objectName := fmt.Sprintf("%s/%s%s", directory, uuid.NewString(), extension)

		contents, err := file.Open()
		if err != nil {
			s.RemoveAll(ctx, stored)
			return nil, err
		}

		contentType := file.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		err = s.files.Put(ctx, objectName, contents, file.Size, contentType)
		_ = contents.Close()
		if err != nil {
			s.RemoveAll(ctx, stored)
			return nil, err
		}

		stored = append(stored, models.StoredDocument{
			Disk:         "minio",
			Path:         objectName,
			OriginalName: filepath.Base(file.Filename),
			MimeType:     contentType,
			Size:         file.Size,
		})
	}

	return stored, nil
}
