package kdkmp

import (
	"testing"

	"agrinaspangan/ebitda-api/internal/models"
)

func TestLockedFiltersFromOptions(t *testing.T) {
	single := []RegionOption{
		{Provinsi: "JAWA BARAT", KotaKabupaten: "KOTA BANDUNG", Kecamatan: "COBLONG", Desa: "SADANG"},
		{Provinsi: "JAWA BARAT", KotaKabupaten: "KOTA BANDUNG", Kecamatan: "COBLONG", Desa: "DAGO"},
	}

	locks := lockedFiltersFromOptions(single)
	if locks.Provinsi == nil || *locks.Provinsi != "JAWA BARAT" {
		t.Fatalf("provinsi harus terkunci, got %#v", locks.Provinsi)
	}
	if locks.KotaKabupaten == nil || *locks.KotaKabupaten != "KOTA BANDUNG" {
		t.Fatalf("kota_kabupaten harus terkunci, got %#v", locks.KotaKabupaten)
	}
	if locks.Kecamatan == nil || *locks.Kecamatan != "COBLONG" {
		t.Fatalf("kecamatan harus terkunci, got %#v", locks.Kecamatan)
	}
	if locks.Desa != nil {
		t.Fatalf("desa punya >1 nilai, tidak boleh terkunci: %#v", *locks.Desa)
	}

	multi := []RegionOption{
		{Provinsi: "JAWA BARAT"},
		{Provinsi: "JAWA TIMUR"},
	}
	if locks := lockedFiltersFromOptions(multi); locks.Provinsi != nil {
		t.Fatalf("provinsi dengan 2 nilai tidak boleh terkunci")
	}

	empty := []RegionOption{{Provinsi: "", KotaKabupaten: ""}}
	if locks := lockedFiltersFromOptions(empty); locks.Provinsi != nil || locks.KotaKabupaten != nil {
		t.Fatal("nilai kosong harus diabaikan")
	}
}

func TestScopeLabel(t *testing.T) {
	cases := []struct {
		assignmentCount int64
		hasOwnEntry     bool
		expected        string
	}{
		{3, false, "3 cakupan wilayah"},
		{2, true, "2 cakupan wilayah"},
		{1, true, "KDKMP sendiri"},
		{1, false, "Wilayah penugasan"},
		{0, true, "KDKMP sendiri"},
		{0, false, "Wilayah penugasan"},
	}

	for _, test := range cases {
		if got := scopeLabel(test.assignmentCount, test.hasOwnEntry); got != test.expected {
			t.Fatalf("scopeLabel(%d, %t) = %q, expected %q", test.assignmentCount, test.hasOwnEntry, got, test.expected)
		}
	}
}

func TestAccessibleScopeConditions(t *testing.T) {
	ownEntry := int64(5)
	sql, args := accessibleScopeConditions(&ownEntry, nil)
	if sql != "sdm_kdkmp_entries.id = ?" || len(args) != 1 || args[0].(int64) != 5 {
		t.Fatalf("scope entry sendiri salah: %q %#v", sql, args)
	}

	provinsi := "JAWA BARAT"
	kota := "KOTA BANDUNG"
	kecamatan := "COBLONG"
	assignments := []models.UserRegionalAssignment{
		{ScopeLevel: models.RegionalScopeProvince, Provinsi: provinsi},
		{ScopeLevel: models.RegionalScopeRegency, Provinsi: provinsi, KotaKabupaten: &kota},
		{ScopeLevel: models.RegionalScopeDistrict, Provinsi: provinsi, KotaKabupaten: &kota, Kecamatan: &kecamatan},
	}

	sql, args = accessibleScopeConditions(nil, assignments)
	expectedSQL := "sdm_kdkmp_entries.id = ? OR (sdm_kdkmp_entries.provinsi = ?) OR (sdm_kdkmp_entries.provinsi = ? AND sdm_kdkmp_entries.kota_kabupaten = ?) OR (sdm_kdkmp_entries.provinsi = ? AND sdm_kdkmp_entries.kota_kabupaten = ? AND sdm_kdkmp_entries.kecamatan = ?)"
	if sql != expectedSQL {
		t.Fatalf("sql scope salah:\n%s", sql)
	}
	if len(args) != 7 {
		t.Fatalf("expected 7 args, got %d: %#v", len(args), args)
	}
	if args[0].(int64) != -1 {
		t.Fatalf("id placeholder harus -1 saat tanpa entry sendiri, got %#v", args[0])
	}
}

func TestRegionFilterConditions(t *testing.T) {
	sql, args := regionFilterConditions(nil)
	if sql != "" || len(args) != 0 {
		t.Fatalf("filter kosong harus menghasilkan sql kosong, got %q %#v", sql, args)
	}

	sql, args = regionFilterConditions(map[string]string{
		"provinsi":       "JAWA BARAT",
		"kota_kabupaten": "KOTA BANDUNG",
		"desa":           " ",
	})
	expected := "sdm_kdkmp_entries.provinsi = ? AND sdm_kdkmp_entries.kota_kabupaten = ?"
	if sql != expected {
		t.Fatalf("sql filter salah: %q", sql)
	}
	if len(args) != 2 || args[0] != "JAWA BARAT" || args[1] != "KOTA BANDUNG" {
		t.Fatalf("args filter salah: %#v", args)
	}
}
