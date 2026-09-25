package server

import "testing"

func TestParseCustomerAnalysisPayload(t *testing.T) {
	valid := customerAnalysisPayload{
		FullName:         "Siti Pelanggan",
		OccupationRole:   "other",
		OccupationOther:  ptrString("Pekerja Kreatif"),
		Age:              30,
		Gender:           "female",
		InterviewPurpose: "Memahami kebutuhan pelanggan",
		Summary:          "Membutuhkan produk harian yang terjangkau.",
		Sentiment:        4,
	}
	analysis, err := parseCustomerAnalysisPayload(valid)
	if err != nil || analysis.OccupationOther == nil || *analysis.OccupationOther != "Pekerja Kreatif" {
		t.Fatalf("payload valid = %#v, %v", analysis, err)
	}

	valid.OccupationOther = nil
	if _, err := parseCustomerAnalysisPayload(valid); err == nil {
		t.Fatal("occupation other tanpa rincian harus ditolak")
	}

	valid.OccupationRole = "farmer"
	valid.OccupationOther = ptrString("tidak digunakan")
	valid.Age = 121
	if _, err := parseCustomerAnalysisPayload(valid); err == nil {
		t.Fatal("umur di luar rentang harus ditolak")
	}
}

func ptrString(value string) *string { return &value }
