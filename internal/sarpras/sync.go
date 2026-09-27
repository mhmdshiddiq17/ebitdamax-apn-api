package sarpras

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"agrinaspangan/ebitda-api/internal/models"
)

// SyncOptions mengatur jalannya sinkronisasi titik sarpras.
type SyncOptions struct {
	DryRun   bool
	MaxPages int
	PageSize int
	// Progress dipanggil setiap halaman selesai diproses (opsional).
	Progress func(page, upserted int)
}

// SyncSummary merangkum hasil satu kali sinkronisasi.
type SyncSummary struct {
	Pages        int
	Upserted     int
	SkippedNoGeo int
	StaleDeleted int64
}

// SyncService menyinkronkan titik sarpras dari portal ke database.
type SyncService struct {
	db     *gorm.DB
	client *Client
	now    func() time.Time
}

// NewSyncService membuat service sinkronisasi titik sarpras.
func NewSyncService(db *gorm.DB, client *Client) *SyncService {
	return &SyncService{db: db, client: client, now: time.Now}
}

// Sync mengambil seluruh halaman portal lalu melakukan upsert per (nik, lat, lng)
// dan menghapus titik yang tidak muncul lagi pada sync ini.
func (s *SyncService) Sync(ctx context.Context, options SyncOptions) (SyncSummary, error) {
	summary := SyncSummary{}
	if s.client == nil || s.client.token == "" {
		return summary, errors.New("PORTAL_PEMBANGUNAN_SARPRAS_TOKEN belum di-set")
	}

	pageSize := options.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}

	// Truncate ke detik: kolom `synced_at` bertipe timestamp(0); tanpa ini
	// baris yang baru di-upsert akan dianggap stale dan ikut terhapus.
	runAt := s.now().Truncate(time.Second)
	page := 1

	for {
		portalPage, err := s.client.FetchPage(ctx, page, pageSize)
		if err != nil {
			return summary, err
		}
		summary.Pages++

		points := make([]models.KoperasiSarprasStatusPoint, 0, len(portalPage.Rows))
		for _, row := range portalPage.Rows {
			point, ok := MapRow(row, runAt)
			if !ok {
				summary.SkippedNoGeo++
				continue
			}
			points = append(points, point)
		}
		summary.Upserted += len(points)

		if !options.DryRun && len(points) > 0 {
			if err := s.upsert(ctx, points); err != nil {
				return summary, err
			}
		}

		if options.Progress != nil {
			options.Progress(page, summary.Upserted)
		}

		if !portalPage.Meta.HasMore {
			break
		}
		if options.MaxPages > 0 && page >= options.MaxPages {
			break
		}
		page++
	}

	if !options.DryRun {
		deleted, err := s.deleteStale(ctx, runAt)
		if err != nil {
			return summary, err
		}
		summary.StaleDeleted = deleted
	}

	return summary, nil
}

func (s *SyncService) upsert(ctx context.Context, points []models.KoperasiSarprasStatusPoint) error {
	err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "nik"}, {Name: "lat"}, {Name: "lng"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"nama_koperasi",
				"provinsi",
				"kota_kabupaten",
				"kecamatan",
				"desa",
				"kodim",
				"validation_status",
				"progress_percentage",
				"batch",
				"completed_sarpras_count",
				"sarpras_less_than_6",
				"sarpras_primary_lengkap",
				"sarpras_secondary_lengkap",
				"sarpras_lengkap",
				"synced_at",
				"updated_at",
			}),
		}).
		CreateInBatches(&points, 500).Error
	if err != nil {
		return fmt.Errorf("menyimpan titik sarpras: %w", err)
	}
	return nil
}

func (s *SyncService) deleteStale(ctx context.Context, runAt time.Time) (int64, error) {
	result := s.db.WithContext(ctx).
		Where("synced_at IS NULL OR synced_at < ?", runAt).
		Delete(&models.KoperasiSarprasStatusPoint{})
	if result.Error != nil {
		return 0, fmt.Errorf("menghapus titik sarpras lama: %w", result.Error)
	}
	return result.RowsAffected, nil
}
