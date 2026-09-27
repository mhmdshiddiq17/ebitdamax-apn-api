// Package sarpras menyinkronkan titik koperasi (sarpras/pembangunan) dari
// portal pembangunan dan menurunkan hasilnya ke data SDM KDKMP.
package sarpras

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"agrinaspangan/ebitda-api/internal/models"
)

// DefaultPageSize mengikuti aplikasi lama (500 baris per halaman).
const DefaultPageSize = 500

const requestTimeout = 60 * time.Second

// portalMaxAttempts membatasi percobaan ulang untuk error jaringan/timeout/5xx.
const portalMaxAttempts = 3

// statusError menandai respons portal non-2xx.
type statusError struct {
	Status int
	Body   string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("portal mengembalikan HTTP %d: %s", e.Status, e.Body)
}

// Client adalah klien HTTP portal pembangunan untuk endpoint status sarpras.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
	sleep   func(time.Duration)
}

// NewClient membuat klien portal. baseURL tanpa trailing slash.
// HTTP/1.1 dipakai (portal terbukti flaky pada HTTP/2: stream INTERNAL_ERROR).
func NewClient(baseURL, token string) *Client {
	transport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		ForceAttemptHTTP2: false,
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: requestTimeout, Transport: transport},
		sleep:   time.Sleep,
	}
}

// PortalMeta adalah metadata paginasi respons portal.
type PortalMeta struct {
	CurrentPage int  `json:"current_page"`
	PerPage     int  `json:"per_page"`
	Total       int  `json:"total"`
	TotalPages  int  `json:"total_pages"`
	HasMore     bool `json:"has_more"`
}

// PortalRow adalah satu baris respons portal (angka dikirim sebagai string).
type PortalRow struct {
	NIK                     *string `json:"nik_koperasi"`
	NamaKoperasi            *string `json:"koperasi_name"`
	Provinsi                *string `json:"province_name"`
	KotaKabupaten           *string `json:"city_name"`
	Kecamatan               *string `json:"district_name"`
	Desa                    *string `json:"village_name"`
	Kodim                   *string `json:"kodim_name"`
	Latitude                *string `json:"latitude"`
	Longitude               *string `json:"longitude"`
	ValidationStatus        *string `json:"validation_status"`
	ProgressPercentage      *string `json:"progress_percentage"`
	Batch                   *string `json:"batch"`
	CompletedSarprasCount   int     `json:"completed_sarpras_count"`
	SarprasLessThan6        bool    `json:"sarpras_less_than_6"`
	SarprasPrimaryLengkap   bool    `json:"sarpras_primary_lengkap"`
	SarprasSecondaryLengkap bool    `json:"sarpras_secondary_lengkap"`
	SarprasLengkap          bool    `json:"sarpras_lengkap"`
}

// PortalPage adalah satu halaman respons portal.
type PortalPage struct {
	Rows []PortalRow
	Meta PortalMeta
}

// FetchPage mengambil satu halaman titik sarpras dari portal dengan percobaan
// ulang untuk error jaringan/timeout/5xx (bukan error 4xx).
func (c *Client) FetchPage(ctx context.Context, page, perPage int) (PortalPage, error) {
	var lastErr error
	for attempt := 1; attempt <= portalMaxAttempts; attempt++ {
		result, err := c.fetchPage(ctx, page, perPage)
		if err == nil {
			return result, nil
		}
		lastErr = err

		if !retryablePortalError(err) || ctx.Err() != nil {
			return PortalPage{}, err
		}

		select {
		case <-ctx.Done():
			return PortalPage{}, ctx.Err()
		default:
			c.sleep(time.Duration(attempt) * 2 * time.Second)
		}
	}

	return PortalPage{}, fmt.Errorf("portal gagal setelah %d percobaan: %w", portalMaxAttempts, lastErr)
}

func retryablePortalError(err error) bool {
	var status *statusError
	if errors.As(err, &status) {
		return status.Status >= 500 || status.Status == http.StatusTooManyRequests
	}

	// Error jaringan/transport apa pun (timeout, koneksi putus, stream HTTP/2)
	// layak dicoba ulang; error 4xx di atas tidak diulang.
	return !errors.Is(err, context.Canceled)
}

func (c *Client) fetchPage(ctx context.Context, page, perPage int) (PortalPage, error) {
	if perPage <= 0 {
		perPage = DefaultPageSize
	}

	endpoint, err := url.Parse(c.baseURL + "/api/koperasi-sarpras-status")
	if err != nil {
		return PortalPage{}, fmt.Errorf("menyusun URL portal: %w", err)
	}
	query := endpoint.Query()
	query.Set("page", strconv.Itoa(page))
	query.Set("per_page", strconv.Itoa(perPage))
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return PortalPage{}, fmt.Errorf("membuat request portal: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Accept", "application/json")

	response, err := c.http.Do(request)
	if err != nil {
		return PortalPage{}, fmt.Errorf("memanggil portal: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return PortalPage{}, &statusError{Status: response.StatusCode, Body: strings.TrimSpace(string(body))}
	}

	var payload struct {
		Data []PortalRow `json:"data"`
		Meta PortalMeta  `json:"meta"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return PortalPage{}, fmt.Errorf("membaca respons portal: %w", err)
	}

	return PortalPage{Rows: payload.Data, Meta: payload.Meta}, nil
}

// MapRow memetakan baris portal menjadi titik sarpras. Baris tanpa koordinat
// valid dilewati (false), mengikuti aplikasi lama.
func MapRow(row PortalRow, syncedAt time.Time) (models.KoperasiSarprasStatusPoint, bool) {
	lat, latOK := parsePortalFloat(row.Latitude)
	lng, lngOK := parsePortalFloat(row.Longitude)
	if !latOK || !lngOK {
		return models.KoperasiSarprasStatusPoint{}, false
	}

	progress, _ := parsePortalFloat(row.ProgressPercentage)

	return models.KoperasiSarprasStatusPoint{
		NIK:                     row.NIK,
		NamaKoperasi:            row.NamaKoperasi,
		Provinsi:                row.Provinsi,
		KotaKabupaten:           row.KotaKabupaten,
		Kecamatan:               row.Kecamatan,
		Desa:                    row.Desa,
		Kodim:                   row.Kodim,
		Lat:                     math.Round(lat*1e6) / 1e6,
		Lng:                     math.Round(lng*1e6) / 1e6,
		ValidationStatus:        row.ValidationStatus,
		ProgressPercentage:      math.Round(progress*10) / 10,
		Batch:                   row.Batch,
		CompletedSarprasCount:   row.CompletedSarprasCount,
		SarprasLessThan6:        row.SarprasLessThan6,
		SarprasPrimaryLengkap:   row.SarprasPrimaryLengkap,
		SarprasSecondaryLengkap: row.SarprasSecondaryLengkap,
		SarprasLengkap:          row.SarprasLengkap,
		SyncedAt:                &syncedAt,
	}, true
}

func parsePortalFloat(value *string) (float64, bool) {
	if value == nil {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(*value), 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}
