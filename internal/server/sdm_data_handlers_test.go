package server

import "testing"

func TestValidateJumlahKaryawan(t *testing.T) {
	value := func(input int) *int { return &input }

	cases := []struct {
		name    string
		input   *int
		want    int
		message string
	}{
		{name: "kosong ditolak", input: nil, message: "Jumlah karyawan wajib diisi."},
		{name: "negatif ditolak", input: value(-1), message: "Jumlah karyawan tidak boleh negatif."},
		{name: "nol diterima", input: value(0), want: 0},
		{name: "positif diterima", input: value(7), want: 7},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, message := validateJumlahKaryawan(test.input)
			if message != test.message {
				t.Fatalf("message = %q, mau %q", message, test.message)
			}
			if message == "" && got != test.want {
				t.Fatalf("value = %d, mau %d", got, test.want)
			}
		})
	}
}

func TestSdmDataTotalPages(t *testing.T) {
	if sdmDataTotalPages(0) != 1 || sdmDataTotalPages(25) != 1 || sdmDataTotalPages(26) != 2 || sdmDataTotalPages(6456) != 259 {
		t.Fatalf("total pages salah")
	}
}
