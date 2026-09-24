// Package slug membuat slug URL-friendly dan menjaga keunikannya di database.
package slug

import (
	"fmt"
	"strings"
	"unicode"

	"gorm.io/gorm"
)

// Make mengubah teks menjadi slug (huruf kecil, pemisah "-").
func Make(value string) string {
	return MakeSeparator(value, "-")
}

// MakeSeparator mengubah teks menjadi slug dengan pemisah kustom.
func MakeSeparator(value string, separator string) string {
	var builder strings.Builder
	lastSeparator := false

	for _, r := range strings.ToLower(value) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastSeparator = false
		case !lastSeparator:
			builder.WriteString(separator)
			lastSeparator = true
		}
	}

	return strings.Trim(builder.String(), separator)
}

// Unique mencari slug yang belum dipakai pada table; bila bentrok menambahkan
// suffix angka (-2, -3, ...). excludeID dipakai saat update (0 = tidak ada).
func Unique(db *gorm.DB, table string, base string, excludeID int64) (string, error) {
	return UniqueColumn(db, table, "slug", base, excludeID)
}

// UniqueColumn seperti Unique tetapi untuk kolom bebas (mis. username).
func UniqueColumn(db *gorm.DB, table string, column string, base string, excludeID int64) (string, error) {
	if base == "" {
		base = "item"
	}

	candidate := base
	for i := 2; ; i++ {
		var count int64
		query := db.Table(table).Where(column+" = ?", candidate)
		if excludeID > 0 {
			query = query.Where("id <> ?", excludeID)
		}
		if err := query.Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}
