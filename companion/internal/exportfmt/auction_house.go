package exportfmt

var neutralAuctionHouseMapIDs = map[int]struct{}{
	1434: {}, // Stranglethorn Vale / Booty Bay
	1446: {}, // Tanaris / Gadgetzan
	1452: {}, // Winterspring / Everlook
}

var neutralAuctionHouseZones = map[string]struct{}{
	"Stranglethorn Vale": {},
	"Tanaris":            {},
	"Winterspring":       {},
}

var neutralAuctionHouseSubzones = map[string]struct{}{
	"Booty Bay": {},
	"Gadgetzan": {},
	"Everlook":  {},
}

func classifyAuctionHouse(zone, subzone string, uiMapID int) string {
	if _, ok := neutralAuctionHouseMapIDs[uiMapID]; ok {
		return "neutral"
	}
	if _, ok := neutralAuctionHouseZones[zone]; ok {
		return "neutral"
	}
	if _, ok := neutralAuctionHouseSubzones[subzone]; ok {
		return "neutral"
	}
	return "faction"
}

func (scan Scan) Market() string {
	if scan.AuctionHouse == "neutral" {
		return "Neutral"
	}
	return scan.Faction
}
