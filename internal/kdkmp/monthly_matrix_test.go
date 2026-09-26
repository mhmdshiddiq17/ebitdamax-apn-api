package kdkmp

import (
	"testing"
	"time"

	"agrinaspangan/ebitda-api/internal/models"
)

func TestMonthlyMatrixHelpers(t *testing.T) {
	t.Run("datesBetween inklusif", func(t *testing.T) {
		start := time.Date(2026, 2, 1, 0, 0, 0, 0, Location())
		end := time.Date(2026, 2, 5, 0, 0, 0, 0, Location())

		dates := datesBetween(start, end)

		if len(dates) != 5 {
			t.Fatalf("jumlah tanggal = %d", len(dates))
		}
		if DateString(dates[0]) != "2026-02-01" || DateString(dates[4]) != "2026-02-05" {
			t.Fatalf("rentang = %s s/d %s", DateString(dates[0]), DateString(dates[4]))
		}
	})

	t.Run("datesBetween terbalik kosong", func(t *testing.T) {
		start := time.Date(2026, 2, 5, 0, 0, 0, 0, Location())
		end := time.Date(2026, 2, 1, 0, 0, 0, 0, Location())

		if dates := datesBetween(start, end); len(dates) != 0 {
			t.Fatalf("dates = %v", dates)
		}
	})

	t.Run("fixed cost memakai default bila tidak lengkap", func(t *testing.T) {
		configured := []models.Task{
			{ID: 1, FixedCost: models.ConfiguredFixedCostBreakdown(models.CostBreakdown{"man": 1000, "machine": 2000})},
			{ID: 2, FixedCost: models.ConfiguredFixedCostBreakdown(models.CostBreakdown{"material": 500})},
		}
		if total := monthlyFixedCost(configured); total != 3500 {
			t.Fatalf("fixed cost terkonfigurasi = %v", total)
		}

		partial := []models.Task{
			{ID: 1, FixedCost: models.ConfiguredFixedCostBreakdown(models.CostBreakdown{"man": 1000})},
			{ID: 2, FixedCost: models.CostBreakdown{"man": 500}},
		}
		if total := monthlyFixedCost(partial); total != DefaultDailyFixedCost {
			t.Fatalf("fixed cost sebagian = %v", total)
		}

		if total := monthlyFixedCost(nil); total != DefaultDailyFixedCost {
			t.Fatalf("fixed cost kosong = %v", total)
		}
	})

	t.Run("task eksekusi = wajib atau dipilih", func(t *testing.T) {
		tasks := []models.Task{
			{ID: 1, IsMandatory: true},
			{ID: 2},
			{ID: 3},
		}

		if result := monthlyExecutionTasks(tasks, nil); len(result) != 1 || result[0].ID != 1 {
			t.Fatalf("tanpa pilihan = %+v", result)
		}
		if result := monthlyExecutionTasks(tasks, models.IntList{3}); len(result) != 2 || result[1].ID != 3 {
			t.Fatalf("dengan pilihan = %+v", result)
		}
	})

	t.Run("actual cost hanya bila ada durasi", func(t *testing.T) {
		plan, actual := monthlyCosts(1_000, 200, 50, 0)
		if plan != 1_200 || actual != 0 {
			t.Fatalf("tanpa durasi = %v / %v", plan, actual)
		}

		plan, actual = monthlyCosts(1_000, 200, 50, 10)
		if plan != 1_200 || actual != 1_250 {
			t.Fatalf("dengan durasi = %v / %v", plan, actual)
		}
	})
}
