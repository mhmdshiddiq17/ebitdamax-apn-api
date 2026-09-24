// Package kdkmp berisi logika domain KDKMP yang dipakai lintas modul
// (tanggal bisnis, pemilihan task, alokasi personel).
package kdkmp

import (
	"time"

	"agrinaspangan/ebitda-api/config"
)

// Location mengembalikan timezone bisnis KDKMP (default Asia/Jakarta).
func Location() *time.Location {
	name := config.GetEnv("KDKMP_BUSINESS_TIMEZONE", "Asia/Jakarta")
	location, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return location
}

// BusinessDate mengembalikan awal hari tanggal bisnis saat ini.
func BusinessDate() time.Time {
	now := time.Now().In(Location())
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, Location())
}

// DateString memformat tanggal bisnis (YYYY-MM-DD).
func DateString(date time.Time) string {
	return date.Format("2006-01-02")
}

// DayRange mengembalikan rentang awal dan akhir hari dalam UTC
// (dipakai untuk query kolom timestamp).
func DayRange(date time.Time) (time.Time, time.Time) {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, Location())
	end := start.Add(24*time.Hour - time.Nanosecond)
	return start.UTC(), end.UTC()
}
