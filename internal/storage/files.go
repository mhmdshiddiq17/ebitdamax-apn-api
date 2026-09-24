package storage

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
)

// Files membungkus operasi objek pada bucket MinIO aplikasi.
type Files struct {
	client *minio.Client
	bucket string
}

// NewFiles membuat wrapper penyimpanan objek.
func NewFiles(client *minio.Client, bucket string) *Files {
	return &Files{client: client, bucket: bucket}
}

// Put mengunggah objek.
func (f *Files) Put(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	_, err := f.client.PutObject(ctx, f.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// Stream membuka objek untuk dibaca (dipakai preview/download).
func (f *Files) Stream(ctx context.Context, objectName string) (*minio.Object, error) {
	object, err := f.client.GetObject(ctx, f.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	if _, err := object.Stat(); err != nil {
		_ = object.Close()
		return nil, err
	}

	return object, nil
}

// ObjectExists mengecek keberadaan objek.
func (f *Files) ObjectExists(ctx context.Context, objectName string) (bool, error) {
	_, err := f.client.StatObject(ctx, f.bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Remove menghapus objek (mengabaikan objek yang tidak ada).
func (f *Files) Remove(ctx context.Context, objectName string) error {
	return f.client.RemoveObject(ctx, f.bucket, objectName, minio.RemoveObjectOptions{})
}
