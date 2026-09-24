package kdkmp

import (
	"math"
	"testing"

	"agrinaspangan/ebitda-api/internal/models"
)

func TestDashboardCalculations(t *testing.T) {
	t.Run("normalizes Indonesian money", func(t *testing.T) {
		value, ok := numericValue("Rp 3.000.000,50")
		if !ok || value != 3_000_000.5 {
			t.Fatalf("numericValue() = %v, %v", value, ok)
		}
	})

	t.Run("uses legacy fixed cost and weights", func(t *testing.T) {
		tasks := []models.Task{{ID: 1, TimeRequire: 30}, {ID: 2, TimeRequire: 30}}
		costs := plannedFixedCosts(tasks, DefaultDailyFixedCost, 60, false)
		if math.Abs(costs[1]+costs[2]-DefaultDailyFixedCost) > 0.001 {
			t.Fatalf("allocated fixed cost = %v", costs)
		}
		if score := CalculatePerformanceScoring("20000000", "20000000", 100, 100); score == nil || *score != "100%" {
			t.Fatalf("score = %v", score)
		}
		if margin := CalculateActualEbitdaMargin("20000000"); margin == nil || *margin != "53.82%" {
			t.Fatalf("margin = %v", margin)
		}
	})
}
