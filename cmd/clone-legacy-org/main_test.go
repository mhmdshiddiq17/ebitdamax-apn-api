package main

import (
	"testing"

	"agrinaspangan/ebitda-api/internal/models"
)

func stringPtr(value string) *string { return &value }
func int64Ptr(value int64) *int64    { return &value }

func TestBuildPlanSkipsExistingAndDedupesUsername(t *testing.T) {
	state := &sourceState{
		Entries: []legacyEntry{
			{ID: 1, NIK: stringPtr("NIK-A"), NamaKoperasi: "Koperasi A"},
			{ID: 2, NIK: stringPtr("NIK-B"), NamaKoperasi: "Koperasi B"},
		},
		Managers: []models.User{
			{ID: 10, Email: "existing@agrinas.test", Username: stringPtr("manager-contoh"), SDMKdkmpEntryID: int64Ptr(1)},
			{ID: 11, Email: "baru@agrinas.test", Username: stringPtr("manager-contoh"), SDMKdkmpEntryID: int64Ptr(2)},
			{ID: 12, Email: "tanpa-entry@agrinas.test", Username: stringPtr("manager-lain")},
		},
		ProvinceRows: map[string]int{},
	}
	target := &targetState{
		EntryNIKs: map[string]bool{"NIK-A": true},
		UsersByEmail: map[string]models.User{
			"existing@agrinas.test": {ID: 7, Username: stringPtr("manager-contoh")},
		},
		Usernames: map[string]string{"manager-contoh": "existing@agrinas.test"},
	}

	plan := buildPlan(state, target, cloneOptions{})

	if len(plan.Entries) != 1 || plan.Entries[0].NIK == nil || *plan.Entries[0].NIK != "NIK-B" {
		t.Fatalf("entry rencana = %+v", plan.Entries)
	}
	if plan.EntriesSkipped != 1 {
		t.Fatalf("entry dilewati = %d", plan.EntriesSkipped)
	}
	if len(plan.Managers) != 3 {
		t.Fatalf("manager rencana = %d", len(plan.Managers))
	}
	if plan.Managers[0].ExistingID != 7 || plan.Managers[0].EntryNIK != "NIK-A" {
		t.Fatalf("manager existing = %+v", plan.Managers[0])
	}
	if plan.Managers[1].ExistingID != 0 || plan.Managers[1].Username != "manager-contoh-2" {
		t.Fatalf("manager baru = %+v", plan.Managers[1])
	}
	if plan.Managers[1].EntryNIK != "NIK-B" {
		t.Fatalf("entry manager baru = %q", plan.Managers[1].EntryNIK)
	}
	if plan.Managers[2].EntryNIK != "" {
		t.Fatalf("manager tanpa entry = %q", plan.Managers[2].EntryNIK)
	}
}

func TestResetTargetsPicksFirstWithEntry(t *testing.T) {
	managers := []managerPlan{
		{Legacy: models.User{Email: "a@x.test"}},
		{Legacy: models.User{Email: "b@x.test"}, EntryNIK: "NIK-B"},
		{Legacy: models.User{Email: "c@x.test"}, EntryNIK: "NIK-C"},
		{Legacy: models.User{Email: "d@x.test"}, EntryNIK: "NIK-D"},
	}

	targets := resetTargets(managers, 2)

	if len(targets) != 2 || targets[0].Legacy.Email != "b@x.test" || targets[1].Legacy.Email != "c@x.test" {
		t.Fatalf("target reset = %+v", targets)
	}
	if resetTargets(managers, 0) != nil {
		t.Fatal("count 0 harus kosong")
	}
}

func TestPlanRegionalManagersDeterministic(t *testing.T) {
	provinces := map[string]int{"Jawa Timur": 1018, "Jawa Tengah": 757, "Banten": 112}
	existing := map[string]bool{"manager-wilayah-jawa-tengah@agrinas.test": true}
	taken := map[string]string{"manager-wilayah-jawa-timur": "lain@agrinas.test"}

	plans := planRegionalManagers(provinces, 2, existing, taken)

	if len(plans) != 2 {
		t.Fatalf("rencana regional = %d", len(plans))
	}
	if plans[0].Provinsi != "Jawa Timur" || plans[0].Email != "manager-wilayah-jawa-timur@agrinas.test" {
		t.Fatalf("regional pertama = %+v", plans[0])
	}
	if plans[0].Username != "manager-wilayah-jawa-timur-2" {
		t.Fatalf("username dedupe = %q", plans[0].Username)
	}
	if plans[1].Provinsi != "Banten" {
		t.Fatalf("regional kedua = %+v", plans[1])
	}
}

func TestSlugifyAndUniqueUsername(t *testing.T) {
	if slug := slugify("Daerah Khusus Ibukota Jakarta"); slug != "daerah-khusus-ibukota-jakarta" {
		t.Fatalf("slugify = %q", slug)
	}
	if slug := slugify("DKI Jakarta"); slug != "dki-jakarta" {
		t.Fatalf("slugify = %q", slug)
	}

	taken := map[string]string{"manager": "a@x.test", "manager-2": "b@x.test"}
	if username := uniqueUsername("manager", taken); username != "manager-3" {
		t.Fatalf("uniqueUsername = %q", username)
	}
	if username := uniqueUsername("", map[string]string{}); username != "manager" {
		t.Fatalf("uniqueUsername fallback = %q", username)
	}
}

func TestEntryIDPointer(t *testing.T) {
	ids := map[string]int64{"NIK-A": 5}
	if pointer := entryIDPointer("", ids); pointer != nil {
		t.Fatal("NIK kosong harus nil")
	}
	if pointer := entryIDPointer("NIK-X", ids); pointer != nil {
		t.Fatal("NIK tidak dikenal harus nil")
	}
	if pointer := entryIDPointer("NIK-A", ids); pointer == nil || *pointer != 5 {
		t.Fatalf("pointer = %v", pointer)
	}
}

func TestPrepareEntryInsertsZeroesLegacyID(t *testing.T) {
	entries := []legacyEntry{{ID: 64, NamaKoperasi: "Koperasi Desa Kalipucang", NIK: stringPtr("kdkmp_kalipucang_grabag")}}

	prepared := prepareEntryInserts(entries)

	if len(prepared) != 1 || prepared[0].ID != 0 {
		t.Fatalf("ID harus dikosongkan: %+v", prepared)
	}
	if prepared[0].NamaKoperasi != "Koperasi Desa Kalipucang" || entries[0].ID != 64 {
		t.Fatalf("data sumber berubah: %+v", prepared)
	}
}
