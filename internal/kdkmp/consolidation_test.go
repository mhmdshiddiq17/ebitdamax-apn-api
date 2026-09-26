package kdkmp

import (
	"testing"

	"agrinaspangan/ebitda-api/internal/models"
)

func strPtr(value string) *string {
	return &value
}

func testRecord(planRevenue *string, actualRevenue *string) models.EbitdamaxKdkmp {
	return models.EbitdamaxKdkmp{PlanRevenue: planRevenue, ActualRevenue: actualRevenue}
}

func TestConsolidateEntriesNational(t *testing.T) {
	entries := []models.SdmKdkmpEntry{
		{
			ID: 1, Provinsi: strPtr("JAWA BARAT"), KotaKabupaten: strPtr("KOTA BANDUNG"),
			DailyEbitdaRecords: []models.EbitdamaxKdkmp{
				testRecord(strPtr("1000000"), strPtr("900000")),
				testRecord(strPtr("500000"), strPtr("750000")),
			},
		},
		{
			ID: 2, Provinsi: strPtr("JAWA TIMUR"), KotaKabupaten: strPtr("KOTA SURABAYA"),
			DailyEbitdaRecords: []models.EbitdamaxKdkmp{
				testRecord(strPtr("2000000"), nil),
			},
		},
	}

	rows := ConsolidateEntries(entries, ConsolidationLevelNational)
	if len(rows) != 1 {
		t.Fatalf("expected 1 baris nasional, got %d", len(rows))
	}

	row := rows[0]
	if row.Key != "national" || row.Label != "Indonesia" {
		t.Fatalf("baris nasional salah: %+v", row)
	}
	if row.TotalKdkmp != 2 || row.CompleteKdkmp != 3 {
		t.Fatalf("hitungan KDKMP salah: total=%d complete=%d", row.TotalKdkmp, row.CompleteKdkmp)
	}
	if row.PlanRevenue == nil || *row.PlanRevenue != 3500000 {
		t.Fatalf("plan revenue salah: %#v", row.PlanRevenue)
	}
	if row.ActualRevenue == nil || *row.ActualRevenue != 1650000 {
		t.Fatalf("actual revenue salah: %#v", row.ActualRevenue)
	}
	if row.Gap == nil || *row.Gap != -1850000 {
		t.Fatalf("gap salah: %#v", row.Gap)
	}
}

func TestConsolidateEntriesProvinceGroupingAndSort(t *testing.T) {
	entries := []models.SdmKdkmpEntry{
		{
			ID: 1, Provinsi: strPtr("JAWA BARAT"), KotaKabupaten: strPtr("KOTA BANDUNG"),
			DailyEbitdaRecords: []models.EbitdamaxKdkmp{testRecord(strPtr("100"), strPtr("150"))},
		},
		{
			ID: 2, Provinsi: strPtr("JAWA TIMUR"), KotaKabupaten: strPtr("KOTA SURABAYA"),
			DailyEbitdaRecords: nil,
		},
		{
			ID: 3, Provinsi: strPtr("JAWA BARAT"), KotaKabupaten: strPtr("KOTA BEKASI"),
			DailyEbitdaRecords: nil,
		},
	}

	rows := ConsolidateEntries(entries, ConsolidationLevelProvince)
	if len(rows) != 2 {
		t.Fatalf("expected 2 provinsi, got %d", len(rows))
	}
	if rows[0].Label != "JAWA BARAT" || rows[1].Label != "JAWA TIMUR" {
		t.Fatalf("urutan provinsi salah: %q, %q", rows[0].Label, rows[1].Label)
	}

	jawaBarat := rows[0]
	if jawaBarat.TotalKdkmp != 2 || jawaBarat.CompleteKdkmp != 1 {
		t.Fatalf("grup JAWA BARAT salah: total=%d complete=%d", jawaBarat.TotalKdkmp, jawaBarat.CompleteKdkmp)
	}
	if jawaBarat.KotaKabupaten == nil || *jawaBarat.KotaKabupaten != "KOTA BANDUNG" {
		t.Fatalf("metadata baris pertama harus dari entry pertama: %#v", jawaBarat.KotaKabupaten)
	}
	if jawaBarat.Gap == nil || *jawaBarat.Gap != 50 {
		t.Fatalf("gap JAWA BARAT salah: %#v", jawaBarat.Gap)
	}

	jawaTimur := rows[1]
	if jawaTimur.PlanRevenue != nil || jawaTimur.ActualRevenue != nil || jawaTimur.Gap != nil {
		t.Fatalf("tanpa record revenue harus null: %+v", jawaTimur)
	}
}

func TestConsolidateEntriesNullRevenueKeepsGapNull(t *testing.T) {
	entries := []models.SdmKdkmpEntry{
		{
			ID: 1, Provinsi: strPtr("BALI"),
			DailyEbitdaRecords: []models.EbitdamaxKdkmp{
				testRecord(strPtr("1000"), nil),
			},
		},
	}

	rows := ConsolidateEntries(entries, ConsolidationLevelProvince)
	if rows[0].PlanRevenue == nil || *rows[0].PlanRevenue != 1000 {
		t.Fatalf("plan revenue salah: %#v", rows[0].PlanRevenue)
	}
	if rows[0].ActualRevenue != nil {
		t.Fatalf("actual revenue harus null: %#v", rows[0].ActualRevenue)
	}
	if rows[0].Gap != nil {
		t.Fatalf("gap harus null bila salah satu revenue null: %#v", rows[0].Gap)
	}
}

func TestNaturalLess(t *testing.T) {
	cases := []struct {
		left     string
		right    string
		expected bool
	}{
		{"KECAMATAN 2", "KECAMATAN 10", true},
		{"KECAMATAN 10", "KECAMATAN 2", false},
		{"desa a", "Desa B", true},
		{"SUKAMAJU", "SUKAMAJU", false},
		{"01 DESA", "1 DESA", false},
	}

	for _, test := range cases {
		if got := naturalLess(test.left, test.right); got != test.expected {
			t.Fatalf("naturalLess(%q, %q) = %t, expected %t", test.left, test.right, got, test.expected)
		}
	}
}
