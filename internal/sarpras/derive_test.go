package sarpras

import (
	"slices"
	"testing"
	"time"

	"agrinaspangan/ebitda-api/internal/models"
)

func TestSelectLatestByNIK(t *testing.T) {
	points := []models.KoperasiSarprasStatusPoint{
		{NIK: strPtr("NIK-A"), NamaKoperasi: strPtr("Terbaru A")},
		{NIK: strPtr("NIK-B"), NamaKoperasi: strPtr("B")},
		{NIK: strPtr("NIK-A"), NamaKoperasi: strPtr("Lama A")},
		{NIK: nil, NamaKoperasi: strPtr("Tanpa NIK")},
		{NIK: strPtr(""), NamaKoperasi: strPtr("NIK Kosong")},
	}

	selected := selectLatestByNIK(points)

	if len(selected) != 2 {
		t.Fatalf("jumlah terpilih = %d", len(selected))
	}
	if *selected[0].NamaKoperasi != "Terbaru A" || *selected[1].NamaKoperasi != "B" {
		t.Fatalf("urutan/pemilihan salah: %+v", selected)
	}
}

func TestCountEmptyNames(t *testing.T) {
	points := []models.KoperasiSarprasStatusPoint{
		{NamaKoperasi: strPtr("Ada")},
		{NamaKoperasi: strPtr("   ")},
		{NamaKoperasi: nil},
	}

	if count := countEmptyNames(points); count != 2 {
		t.Fatalf("nama kosong = %d", count)
	}
}

func TestDerivedUpdateColumnsProtectManualFields(t *testing.T) {
	protected := []string{"nik", "jumlah_karyawan", "catatan", "created_by", "updated_by", "created_at"}
	for _, column := range protected {
		if slices.Contains(derivedUpdateColumns, column) {
			t.Fatalf("kolom terlindungi %q tidak boleh ikut ditimpa", column)
		}
	}

	expected := []string{"nama_koperasi", "provinsi", "nama_kodim", "desa", "kecamatan", "kota_kabupaten", "batch", "updated_at"}
	for _, column := range expected {
		if !slices.Contains(derivedUpdateColumns, column) {
			t.Fatalf("kolom %q harus ikut diperbarui", column)
		}
	}
}

func TestDeriveRecordsMapsFields(t *testing.T) {
	points := []models.KoperasiSarprasStatusPoint{{
		NIK:           strPtr("NIK-A"),
		NamaKoperasi:  strPtr("Koperasi A"),
		Provinsi:      strPtr("Jawa Timur"),
		KotaKabupaten: strPtr("Kabupaten Sumenep"),
		Kecamatan:     strPtr("Gapura"),
		Desa:          strPtr("Andulang"),
		Kodim:         strPtr("Kodim 0827/Sumenep"),
		Batch:         strPtr("batch2"),
	}}
	timestamp := time.Date(2026, 9, 27, 6, 0, 0, 0, time.UTC)

	records := deriveRecords(points, timestamp)

	if len(records) != 1 {
		t.Fatalf("records = %d", len(records))
	}
	record := records[0]
	if record.NamaKodim == nil || *record.NamaKodim != "Kodim 0827/Sumenep" {
		t.Fatalf("nama_kodim = %v", record.NamaKodim)
	}
	if record.Batch == nil || *record.Batch != "batch2" {
		t.Fatalf("batch = %v", record.Batch)
	}
	if !record.CreatedAt.Equal(timestamp) || !record.UpdatedAt.Equal(timestamp) {
		t.Fatalf("timestamp = %v/%v", record.CreatedAt, record.UpdatedAt)
	}
}
