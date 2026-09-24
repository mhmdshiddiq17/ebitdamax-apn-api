package slug

import "testing"

func TestMake(t *testing.T) {
	cases := map[string]string{
		"Manager Wilayah":       "manager-wilayah",
		"  Kepala Toko / Gerai": "kepala-toko-gerai",
		"Admin  KDKMP":          "admin-kdkmp",
		"EBITDA_Max":            "ebitda-max",
		"---":                   "item",
	}

	for input, expected := range cases {
		if got := Make(input); got != expected {
			t.Fatalf("Make(%q) = %q, expected %q", input, got, expected)
		}
	}
}
