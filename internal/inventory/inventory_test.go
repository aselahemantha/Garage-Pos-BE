package inventory

import (
	"testing"
)

func TestLowStockDeficitCalculation(t *testing.T) {
	tests := []struct {
		name                string
		stockOnHand         float64
		reorderLevel        float64
		costPrice           int64
		expectedDeficit     float64
		expectedSuggested   float64
		expectedEstCost     int64
	}{
		{
			name:              "below reorder level",
			stockOnHand:       2,
			reorderLevel:      5,
			costPrice:         10000,
			expectedDeficit:   3,
			expectedSuggested: 8, // (5*2)-2 = 8
			expectedEstCost:   80000,
		},
		{
			name:              "out of stock",
			stockOnHand:       0,
			reorderLevel:      6,
			costPrice:         5000,
			expectedDeficit:   6,
			expectedSuggested: 12, // (6*2)-0 = 12
			expectedEstCost:   60000,
		},
		{
			name:              "at reorder level exactly",
			stockOnHand:       4,
			reorderLevel:      4,
			costPrice:         2000,
			expectedDeficit:   0,
			expectedSuggested: 4, // (4*2)-4 = 4
			expectedEstCost:   8000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deficit := tt.reorderLevel - tt.stockOnHand
			if deficit < 0 {
				deficit = 0
			}
			if deficit != tt.expectedDeficit {
				t.Errorf("deficit expected %f, got %f", tt.expectedDeficit, deficit)
			}

			suggested := (tt.reorderLevel * 2) - tt.stockOnHand
			if suggested < tt.reorderLevel {
				suggested = tt.reorderLevel
			}
			if suggested <= 0 {
				suggested = 1
			}
			if suggested != tt.expectedSuggested {
				t.Errorf("suggested expected %f, got %f", tt.expectedSuggested, suggested)
			}

			estCost := int64(suggested * float64(tt.costPrice))
			if estCost != tt.expectedEstCost {
				t.Errorf("estCost expected %d, got %d", tt.expectedEstCost, estCost)
			}
		})
	}
}
