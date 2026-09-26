package kdkmp

import (
	"testing"

	"agrinaspangan/ebitda-api/internal/models"
)

func TestComputeMetrics(t *testing.T) {
	t.Run("menghitung durasi, completion, dan compliance", func(t *testing.T) {
		lower, upper := 20, 40
		assigned := []models.Task{
			{ID: 1, LowerThreshold: &lower, UpperThreshold: &upper},
			{ID: 2},
			{ID: 3},
		}
		within, over := 30, 50
		completed := []models.TaskReport{
			{TaskID: 1, DurationMinutes: &within},
			{TaskID: 2, DurationMinutes: &over},
		}

		metrics := computeMetrics(assigned, completed, 125000, 500000)

		if metrics.TotalDuration != "1 jam 20 menit" {
			t.Fatalf("total duration = %q", metrics.TotalDuration)
		}
		if metrics.CompletionRate != 66.67 {
			t.Fatalf("completion rate = %v", metrics.CompletionRate)
		}
		if metrics.TimeComplianceRate != 100 {
			t.Fatalf("time compliance = %v", metrics.TimeComplianceRate)
		}
		if metrics.ActualCost != "125000" || metrics.ActualRevenue != "500000" {
			t.Fatalf("cost/revenue = %q / %q", metrics.ActualCost, metrics.ActualRevenue)
		}
	})

	t.Run("durasi di luar ambang batas tidak dihitung patuh", func(t *testing.T) {
		lower, upper := 20, 40
		assigned := []models.Task{{ID: 1, LowerThreshold: &lower, UpperThreshold: &upper}}
		short := 10
		completed := []models.TaskReport{{TaskID: 1, DurationMinutes: &short}}

		metrics := computeMetrics(assigned, completed, 0, 0)

		if metrics.TimeComplianceRate != 0 {
			t.Fatalf("time compliance = %v", metrics.TimeComplianceRate)
		}
		if metrics.CompletionRate != 100 {
			t.Fatalf("completion rate = %v", metrics.CompletionRate)
		}
	})

	t.Run("tanpa task dan laporan mengembalikan nilai nol", func(t *testing.T) {
		metrics := computeMetrics(nil, nil, 0, 0)

		if metrics.TotalDuration != "0 menit" || metrics.ActualCost != "0" || metrics.ActualRevenue != "0" {
			t.Fatalf("metrics = %+v", metrics)
		}
		if metrics.CompletionRate != 0 || metrics.TimeComplianceRate != 0 {
			t.Fatalf("rates = %v / %v", metrics.CompletionRate, metrics.TimeComplianceRate)
		}
	})
}
