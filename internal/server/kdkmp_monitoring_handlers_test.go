package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"agrinaspangan/ebitda-api/internal/kdkmp"
)

func monitoringTestContext(query string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("GET", "/api/v1/admin/kdkmp-dashboard?"+query, nil)
	return context, recorder
}

func TestParseMonitoringParams(t *testing.T) {
	businessDate := time.Date(2026, 9, 26, 0, 0, 0, 0, kdkmp.Location())

	t.Run("default mengikuti bulan berjalan", func(t *testing.T) {
		context, _ := monitoringTestContext("")

		params, ok := parseMonitoringParams(context, businessDate)

		if !ok {
			t.Fatalf("params ditolak")
		}
		if params.month != "2026-09" || params.status != "all" || params.level != kdkmp.ConsolidationLevelNational || params.page != 1 {
			t.Fatalf("params = %+v", params)
		}
	})

	t.Run("legacy date diambil bulan-nya", func(t *testing.T) {
		context, _ := monitoringTestContext("date=2026-08-15")

		params, ok := parseMonitoringParams(context, businessDate)

		if !ok || params.month != "2026-08" {
			t.Fatalf("month = %q, ok = %v", params.month, ok)
		}
	})

	t.Run("menolak bulan melewati bulan berjalan", func(t *testing.T) {
		context, recorder := monitoringTestContext("month=2026-10")

		if _, ok := parseMonitoringParams(context, businessDate); ok {
			t.Fatalf("bulan masa depan diterima")
		}
		if recorder.Code != 422 {
			t.Fatalf("status = %d", recorder.Code)
		}
	})

	t.Run("menolak status dan level tidak dikenal", func(t *testing.T) {
		context, recorder := monitoringTestContext("status=bogus")
		if _, ok := parseMonitoringParams(context, businessDate); ok || recorder.Code != 422 {
			t.Fatalf("status tidak dikenal diterima (%d)", recorder.Code)
		}

		context, recorder = monitoringTestContext("consolidation_level=galaxy")
		if _, ok := parseMonitoringParams(context, businessDate); ok || recorder.Code != 422 {
			t.Fatalf("level tidak dikenal diterima (%d)", recorder.Code)
		}
	})

	t.Run("menolak tanggal rincian melewati hari ini", func(t *testing.T) {
		context, recorder := monitoringTestContext("detail_date=2026-09-27")

		if _, ok := parseMonitoringParams(context, businessDate); ok {
			t.Fatalf("detail date masa depan diterima")
		}
		if recorder.Code != 422 {
			t.Fatalf("status = %d", recorder.Code)
		}
	})

	t.Run("memangkas filter wilayah dan menolak yang terlalu panjang", func(t *testing.T) {
		context, _ := monitoringTestContext("provinsi=%20Jawa%20Tengah%20&page=2")

		params, ok := parseMonitoringParams(context, businessDate)

		if !ok || params.region["provinsi"] != "Jawa Tengah" || params.page != 2 {
			t.Fatalf("params = %+v", params)
		}

		long := make([]byte, 256)
		for index := range long {
			long[index] = 'a'
		}
		context, recorder := monitoringTestContext("desa=" + string(long))
		if _, ok := parseMonitoringParams(context, businessDate); ok || recorder.Code != 422 {
			t.Fatalf("filter panjang diterima (%d)", recorder.Code)
		}
	})
}

func TestMonitoringHelpers(t *testing.T) {
	if monitoringTotalPages(0) != 1 || monitoringTotalPages(25) != 1 || monitoringTotalPages(26) != 2 {
		t.Fatalf("total pages salah")
	}
	if nullableRegionValue("") != nil || *nullableRegionValue("Jawa") != "Jawa" {
		t.Fatalf("nullable region value salah")
	}
}
