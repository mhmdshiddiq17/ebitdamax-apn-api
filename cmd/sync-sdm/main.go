// Command sync-sdm menurunkan titik sarpras (Sarpras Esensial 1 lengkap) ke
// data SDM KDKMP berdasarkan NIK. Default dry-run; tulis dengan --apply.
//
// Jalankan: go run ./cmd/sync-sdm [--apply]
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
	apply := flag.Bool("apply", false, "tulis hasil derivasi ke sdm_kdkmp_entries")
	flag.Parse()

	config.LoadEnv()
	db := database.Connect()

	started := time.Now()
	summary, err := sarpras.NewDeriveService(db).Derive(context.Background(), *apply)
	if err != nil {
		log.Fatalf("sync sdm gagal: %v", err)
	}

	fmt.Printf(
		"sync sdm selesai dalam %s (applied=%t): sumber unik %d, diperbarui %d, ditambahkan %d, total SDM setelah %d\n",
		time.Since(started).Round(time.Second), summary.Applied, summary.SourcePoints, summary.ToUpdate, summary.ToInsert, summary.TotalAfter,
	)
}
