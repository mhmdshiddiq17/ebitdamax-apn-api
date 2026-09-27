package sarpras

import (
	"context"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// syncLockKey adalah kunci advisory lock Postgres untuk sinkronisasi sarpras
// (pengganti `onOneServer` Laravel: hanya satu proses yang menjalankan sync).
const syncLockKey = 9121501

// StartScheduler menjalankan sinkronisasi titik sarpras secara berkala.
// Mengembalikan cron yang sudah berjalan; pemanggil dapat Stop saat shutdown.
func StartScheduler(db *gorm.DB, service *SyncService, interval string, logger *log.Logger) (*cron.Cron, error) {
	if interval == "" {
		interval = "@every 15m"
	}

	scheduler := cron.New(
		cron.WithLogger(cron.PrintfLogger(logger)),
		cron.WithChain(cron.SkipIfStillRunning(cron.PrintfLogger(logger))),
	)

	if _, err := scheduler.AddFunc(interval, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := RunWithLock(ctx, db, logger, func(ctx context.Context) error {
			summary, err := service.Sync(ctx, SyncOptions{})
			if err != nil {
				return err
			}
			logger.Printf("sync sarpras: %d halaman, %d titik, %d stale dihapus", summary.Pages, summary.Upserted, summary.StaleDeleted)
			return nil
		}); err != nil {
			logger.Printf("sync sarpras gagal: %v", err)
		}
	}); err != nil {
		return nil, err
	}

	scheduler.Start()
	return scheduler, nil
}

// RunWithLock menjalankan job hanya bila advisory lock Postgres didapat.
func RunWithLock(ctx context.Context, db *gorm.DB, logger *log.Logger, job func(context.Context) error) error {
	var locked bool
	if err := db.WithContext(ctx).Raw("SELECT pg_try_advisory_lock(?)", syncLockKey).Scan(&locked).Error; err != nil {
		return err
	}
	if !locked {
		if logger != nil {
			logger.Print("sync sarpras dilewati: proses lain sedang berjalan")
		}
		return nil
	}

	defer func() {
		_ = db.WithContext(context.Background()).Exec("SELECT pg_advisory_unlock(?)", syncLockKey).Error
	}()

	return job(ctx)
}
