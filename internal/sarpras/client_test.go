package sarpras

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func strPtr(value string) *string { return &value }

func TestFetchPageSendsAuthAndParsesRows(t *testing.T) {
	var gotAuth, gotPage, gotPerPage string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPage = r.URL.Query().Get("page")
		gotPerPage = r.URL.Query().Get("per_page")

		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{
				"nik_koperasi":              "kdkmp_andulang_gapura",
				"koperasi_name":             "Koperasi Desa Andulang",
				"province_name":             "Jawa Timur",
				"city_name":                 "Kabupaten Sumenep",
				"district_name":             "Gapura",
				"village_name":              "Andulang",
				"kodim_name":                "Kodim 0827/Sumenep",
				"latitude":                  "-7.003553600000000",
				"longitude":                 "113.980125300000000",
				"validation_status":         "Terverifikasi",
				"progress_percentage":       "100.00",
				"batch":                     "batch2",
				"completed_sarpras_count":   6,
				"sarpras_less_than_6":       false,
				"sarpras_primary_lengkap":   true,
				"sarpras_secondary_lengkap": false,
				"sarpras_lengkap":           false,
			}},
			"meta": map[string]any{"current_page": 1, "per_page": 500, "total": 1, "total_pages": 1, "has_more": false},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL+"/", "token-rahasia")
	page, err := client.FetchPage(context.Background(), 2, 250)
	if err != nil {
		t.Fatalf("FetchPage error = %v", err)
	}
	if gotAuth != "Bearer token-rahasia" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotPage != "2" || gotPerPage != "250" {
		t.Fatalf("page/per_page = %q/%q", gotPage, gotPerPage)
	}
	if page.Meta.HasMore || len(page.Rows) != 1 {
		t.Fatalf("page = %+v", page.Meta)
	}

	syncedAt := time.Date(2026, 9, 27, 6, 0, 0, 0, time.UTC)
	point, ok := MapRow(page.Rows[0], syncedAt)
	if !ok {
		t.Fatal("baris valid dilewati")
	}
	if point.Lat != -7.003554 || point.Lng != 113.980125 {
		t.Fatalf("lat/lng = %v/%v", point.Lat, point.Lng)
	}
	if point.ProgressPercentage != 100 || !point.SarprasPrimaryLengkap {
		t.Fatalf("point = %+v", point)
	}
	if point.SyncedAt == nil || !point.SyncedAt.Equal(syncedAt) {
		t.Fatalf("synced_at = %v", point.SyncedAt)
	}
}

func TestMapRowSkipsMissingCoordinates(t *testing.T) {
	syncedAt := time.Now()

	cases := []PortalRow{
		{Latitude: nil, Longitude: strPtr("113.9")},
		{Latitude: strPtr("-7.0"), Longitude: nil},
		{Latitude: strPtr("bukan-angka"), Longitude: strPtr("113.9")},
	}

	for _, row := range cases {
		if _, ok := MapRow(row, syncedAt); ok {
			t.Fatalf("baris tanpa koordinat valid diterima: %+v", row)
		}
	}
}

func TestFetchPageErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "portal sedang sibuk", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	client.sleep = func(time.Duration) {}
	if _, err := client.FetchPage(context.Background(), 1, 10); err == nil {
		t.Fatal("HTTP 500 harus menghasilkan error")
	}
}

func TestFetchPageRetriesTransientErrors(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 3 {
			http.Error(w, "sibuk", http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{},
			"meta": map[string]any{"has_more": false},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	client.sleep = func(time.Duration) {}
	if _, err := client.FetchPage(context.Background(), 1, 10); err != nil {
		t.Fatalf("harus berhasil setelah retry: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("percobaan = %d", attempts)
	}
}

func TestFetchPageDoesNotRetryClientErrors(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		http.Error(w, "token salah", http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	if _, err := client.FetchPage(context.Background(), 1, 10); err == nil {
		t.Fatal("HTTP 401 harus menghasilkan error")
	}
	if attempts != 1 {
		t.Fatalf("percobaan = %d (4xx tidak boleh diulang)", attempts)
	}
}

func TestFetchPageRetriesTransportErrors(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("hijacker tidak tersedia")
			}
			connection, _, err := hijacker.Hijack()
			if err == nil {
				_ = connection.Close()
			}
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{},
			"meta": map[string]any{"has_more": false},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	client.sleep = func(time.Duration) {}
	if _, err := client.FetchPage(context.Background(), 1, 10); err != nil {
		t.Fatalf("harus berhasil setelah retry transport: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("percobaan = %d", attempts)
	}
}
