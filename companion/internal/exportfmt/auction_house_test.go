package exportfmt

import "testing"

func TestClassifyAuctionHouse(t *testing.T) {
	tests := []struct {
		name     string
		zone     string
		subzone  string
		uiMapID  int
		expected string
	}{
		{
			name:     "neutral map ID",
			zone:     "Localized zone name",
			subzone:  "Localized town name",
			uiMapID:  1446,
			expected: "neutral",
		},
		{
			name:     "neutral zone fallback",
			zone:     "Winterspring",
			subzone:  "",
			uiMapID:  0,
			expected: "neutral",
		},
		{
			name:     "neutral subzone fallback",
			zone:     "Unknown",
			subzone:  "Booty Bay",
			uiMapID:  0,
			expected: "neutral",
		},
		{
			name:     "faction auction house",
			zone:     "Stormwind City",
			subzone:  "Trade District",
			uiMapID:  1453,
			expected: "faction",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := classifyAuctionHouse(test.zone, test.subzone, test.uiMapID)
			if actual != test.expected {
				t.Fatalf("classifyAuctionHouse() = %q, want %q", actual, test.expected)
			}
		})
	}
}

func TestScanMarket(t *testing.T) {
	tests := []struct {
		faction      string
		auctionHouse string
		expected     string
	}{
		{faction: "Alliance", auctionHouse: "faction", expected: "Alliance"},
		{faction: "Horde", auctionHouse: "faction", expected: "Horde"},
		{faction: "Alliance", auctionHouse: "neutral", expected: "Neutral"},
		{faction: "Horde", auctionHouse: "neutral", expected: "Neutral"},
	}

	for _, test := range tests {
		scan := Scan{
			Faction:      test.faction,
			AuctionHouse: test.auctionHouse,
		}
		if actual := scan.Market(); actual != test.expected {
			t.Fatalf("Market() = %q, want %q", actual, test.expected)
		}
	}
}
