package server

import "testing"

func TestRequiredAnnouncementText(t *testing.T) {
	if _, err := requiredAnnouncementText("   ", "wajib", 255); err == nil {
		t.Fatal("pesan kosong harus ditolak")
	}
	value, err := requiredAnnouncementText(" Pengumuman ", "wajib", 255)
	if err != nil || value != "Pengumuman" {
		t.Fatalf("teks normal = %q, %v", value, err)
	}
}
