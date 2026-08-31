package storage

import (
	"context"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"agrinaspangan/ebitda-api/config"
)

// Connect membuat client MinIO dan memastikan bucket utama ada.
func Connect() *minio.Client {
	client, err := minio.New(config.GetEnv("MINIO_ENDPOINT", "127.0.0.1:9000"), &minio.Options{
		Creds:  credentials.NewStaticV4(config.GetEnv("MINIO_ACCESS_KEY", "minioadmin"), config.GetEnv("MINIO_SECRET_KEY", "minioadmin"), ""),
		Secure: config.GetEnv("MINIO_USE_SSL", "false") == "true",
	})
	if err != nil {
		log.Fatalf("failed to create minio client: %v", err)
	}

	ctx := context.Background()
	bucket := config.GetEnv("MINIO_BUCKET", "ebitdamax")

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		log.Fatalf("failed to check minio bucket: %v", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			log.Fatalf("failed to create minio bucket: %v", err)
		}
		log.Printf("minio bucket %q created", bucket)
	}

	log.Println("minio connected")
	return client
}
