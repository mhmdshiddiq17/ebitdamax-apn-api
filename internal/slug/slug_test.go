package slug

import "testing"

func TestMake(t *testing.T) {
	cases := map[string]string{
		"Manager Wilayah":       "manager-wilayah",
		"  Kepala Toko / Gerai": "kepala-toko-gerai",
		"Admin  KDKMP":          "admin-kdkmp",
		"EBITDA_Max":            "ebitda-max",
		"---":                   "",
	}

	for input, expected := range cases {
		if got := Make(input); got != expected {
			t.Fatalf("Make(%q) = %q, expected %q", input, got, expected)
		}
	}
}

func TestMakeSeparator(t *testing.T) {
	if got := MakeSeparator("Total Biaya Operasional", "_"); got != "total_biaya_operasional" {
		t.Fatalf("MakeSeparator underscore = %q", got)
	}
}
