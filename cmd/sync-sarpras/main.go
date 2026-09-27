// Command sync-sarpras menyinkronkan titik koperasi (sarpras/pembangunan) dari
// portal pembangunan ke tabel koperasi_sarpras_status_points.
//
// Jalankan: go run ./cmd/sync-sarpras [--dry-run] [--max-pages N] [--page-size N]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"agrinaspangan/ebitda-api/config"
	"agrinaspangan/ebitda-api/internal/database"
	"agrinaspangan/ebitda-api/internal/sarpras"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "ambil data portal tanpa menulis ke database")
	maxPages := flag.Int("max-pages", 0, "batasi jumlah halaman yang diproses (0 = semua)")
	pageSize := flag.Int("page-size", sarpras.DefaultPageSize, "jumlah baris per halaman")
	flag.Parse()

	config.LoadEnv()

	token := config.GetEnv("PORTAL_PEMBANGUNAN_SARPRAS_TOKEN", "")
	if token == "" {
		log.Fatal("PORTAL_PEMBANGUNAN_SARPRAS_TOKEN belum di-set")
	}
	baseURL := config.GetEnv("PORTAL_PEMBANGAN_BASE_URL", "https://portalkdkmp.id")

	db := database.Connect()
	service := sarpras.NewSyncService(db, sarpras.NewClient(baseURL, token))

	started := time.Now()
	summary, err := service.Sync(context.Background(), sarpras.SyncOptions{
		DryRun:   *dryRun,
		MaxPages: *maxPages,
		PageSize: *pageSize,
		Progress: func(page, upserted int) {
			fmt.Printf("halaman %d selesai (%d titik)\n", page, upserted)
		},
	})
	if err != nil {
		log.Fatalf("sync sarpras gagal: %v", err)
	}

	fmt.Printf(
		"sync sarpras selesai dalam %s (dry-run=%t): %d halaman, %d titik, %d tanpa koordinat, %d stale dihapus\n",
		time.Since(started).Round(time.Second), *dryRun, summary.Pages, summary.Upserted, summary.SkippedNoGeo, summary.StaleDeleted,
	)
}
