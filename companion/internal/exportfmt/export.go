package exportfmt

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/eolseng/wow-markets/companion/internal/luasv"
)

const (
	SchemaVersion = 1
	FormatVersion = 5
)

var ItemFields = []string{
	"itemId",
	"itemString",
	"name",
	"quality",
	"requiredLevel",
}

var RowFields = []string{
	"sourceRow",
	"itemRef",
	"stackCount",
	"minBid",
	"minIncrement",
	"buyout",
	"bidAmount",
	"saleStatus",
	"hasAllInfo",
}

type Database struct {
	SchemaVersion int
	Scans         []Scan
}

type Scan struct {
	FormatVersion    int    `json:"format_version"`
	Status           string `json:"status"`
	CapturedAt       int64  `json:"captured_at"`
	ExportStartedAt  int64  `json:"export_started_at"`
	ExportFinishedAt int64  `json:"export_finished_at"`
	ExportDurationMS int    `json:"export_duration_ms"`
	ExportBatchSize  int    `json:"export_batch_size"`
	Region           string `json:"region"`
	Realm            string `json:"realm"`
	Faction          string `json:"faction"`
	AuctionHouse     string `json:"auction_house"`
	CaptureZone      string `json:"capture_zone"`
	CaptureSubzone   string `json:"capture_subzone"`
	CaptureUIMapID   int    `json:"capture_ui_map_id"`
	ScannerName      string `json:"scanner_character_name"`
	ScannerRealm     string `json:"scanner_character_realm"`
	ScannerGUID      string `json:"scanner_character_guid"`
	ScannerRegion    string `json:"scanner_region"`
	Source           string `json:"source"`
	SourceEvent      string `json:"source_event"`
	SourceVersion    string `json:"source_version"`
	AddonVersion     string `json:"addon_version"`
	SourceRowCount   int    `json:"source_row_count"`
	ExportedRowCount int    `json:"exported_row_count"`
	ItemCount        int    `json:"item_count"`
	Truncated        bool   `json:"truncated"`
	Rows             []Row  `json:"rows"`
}

type Row struct {
	SourceRow     int    `json:"source_row"`
	ItemID        int    `json:"item_id"`
	ItemString    string `json:"item_string"`
	Name          string `json:"name"`
	StackCount    int    `json:"stack_count"`
	Quality       int    `json:"quality"`
	RequiredLevel int    `json:"required_level"`
	MinBid        int64  `json:"min_bid"`
	MinIncrement  int64  `json:"min_increment"`
	Buyout        int64  `json:"buyout"`
	BidAmount     int64  `json:"bid_amount"`
	SaleStatus    int    `json:"sale_status"`
	HasAllInfo    bool   `json:"has_all_info"`
}

type itemIdentity struct {
	ItemID        int
	ItemString    string
	Name          string
	Quality       int
	RequiredLevel int
}

func Decode(root *luasv.Table) (Database, error) {
	schemaVersion, err := intField(root, "schemaVersion")
	if err != nil {
		return Database{}, err
	}
	if schemaVersion != SchemaVersion {
		return Database{}, fmt.Errorf(
			"unsupported schemaVersion %d; expected %d",
			schemaVersion,
			SchemaVersion,
		)
	}

	rawScans, err := tableField(root, "pendingScans")
	if err != nil {
		return Database{}, err
	}
	scanValues, err := rawScans.Sequence()
	if err != nil {
		return Database{}, fmt.Errorf("pendingScans: %w", err)
	}

	database := Database{
		SchemaVersion: schemaVersion,
		Scans:         make([]Scan, 0, len(scanValues)),
	}
	for index, value := range scanValues {
		table, ok := value.(*luasv.Table)
		if !ok {
			return Database{}, fmt.Errorf("pendingScans[%d] is not a table", index+1)
		}
		scan, err := decodeScan(table)
		if err != nil {
			return Database{}, fmt.Errorf("pendingScans[%d]: %w", index+1, err)
		}
		database.Scans = append(database.Scans, scan)
	}

	return database, nil
}

func (scan Scan) Checksum() (string, error) {
	payload, err := scan.CanonicalJSON()
	if err != nil {
		return "", err
	}
	return ChecksumBytes(payload), nil
}

func ChecksumBytes(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (scan Scan) CanonicalJSON() ([]byte, error) {
	payload, err := json.Marshal(scan)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical scan: %w", err)
	}
	return payload, nil
}

func (scan Scan) Validate() error {
	if scan.FormatVersion != FormatVersion {
		return fmt.Errorf(
			"unsupported formatVersion %d; expected %d",
			scan.FormatVersion,
			FormatVersion,
		)
	}
	if scan.Status != "ready" {
		return fmt.Errorf("status is %q, expected %q", scan.Status, "ready")
	}
	if scan.CapturedAt <= 0 || scan.ExportStartedAt <= 0 || scan.ExportFinishedAt <= 0 {
		return fmt.Errorf("capture and export timestamps must be positive")
	}
	if scan.ExportFinishedAt < scan.ExportStartedAt {
		return fmt.Errorf("exportFinishedAt precedes exportStartedAt")
	}
	if scan.ExportDurationMS < 0 {
		return fmt.Errorf("exportDurationMs must not be negative")
	}
	if scan.ExportBatchSize <= 0 {
		return fmt.Errorf("exportBatchSize must be positive")
	}
	if !validRegion(scan.Region) {
		return fmt.Errorf("unsupported region %q", scan.Region)
	}
	if scan.Realm == "" || scan.ScannerRealm == "" {
		return fmt.Errorf("realm and scanner realm are required")
	}
	if scan.Faction != "Alliance" && scan.Faction != "Horde" {
		return fmt.Errorf("unsupported faction %q", scan.Faction)
	}
	if scan.AuctionHouse != "faction" && scan.AuctionHouse != "neutral" {
		return fmt.Errorf("unsupported auction house %q", scan.AuctionHouse)
	}
	if scan.CaptureZone == "" {
		return fmt.Errorf("capture zone is required")
	}
	if scan.CaptureUIMapID < 0 {
		return fmt.Errorf("capture UI map ID must not be negative")
	}
	expectedAuctionHouse := classifyAuctionHouse(
		scan.CaptureZone,
		scan.CaptureSubzone,
		scan.CaptureUIMapID,
	)
	if scan.AuctionHouse != expectedAuctionHouse {
		return fmt.Errorf(
			"auction house %q does not match capture location; expected %q",
			scan.AuctionHouse,
			expectedAuctionHouse,
		)
	}
	if scan.ScannerName == "" || scan.ScannerGUID == "" {
		return fmt.Errorf("scanner identity is incomplete")
	}
	if !validRegion(scan.ScannerRegion) {
		return fmt.Errorf("unsupported scanner region %q", scan.ScannerRegion)
	}
	if scan.ScannerRegion != scan.Region {
		return fmt.Errorf(
			"scanner region %q does not match market region %q",
			scan.ScannerRegion,
			scan.Region,
		)
	}
	if scan.Source == "" ||
		scan.SourceEvent == "" ||
		scan.SourceVersion == "" ||
		scan.AddonVersion == "" {
		return fmt.Errorf("source and addon metadata are required")
	}
	if scan.SourceRowCount < 0 ||
		scan.ExportedRowCount < 0 ||
		scan.ItemCount < 0 {
		return fmt.Errorf("row and item counts must not be negative")
	}
	if scan.SourceRowCount < scan.ExportedRowCount {
		return fmt.Errorf(
			"sourceRowCount %d is less than exportedRowCount %d",
			scan.SourceRowCount,
			scan.ExportedRowCount,
		)
	}
	if scan.ExportedRowCount != len(scan.Rows) {
		return fmt.Errorf(
			"exportedRowCount %d does not match %d rows",
			scan.ExportedRowCount,
			len(scan.Rows),
		)
	}
	if scan.Truncated != (scan.SourceRowCount != scan.ExportedRowCount) {
		return fmt.Errorf("truncated flag does not match declared row counts")
	}
	if scan.ExportedRowCount > 0 && scan.ItemCount == 0 {
		return fmt.Errorf("itemCount must be positive when rows are present")
	}

	sourceRows := make(map[int]struct{}, len(scan.Rows))
	for index, row := range scan.Rows {
		if _, err := validateRow(row); err != nil {
			return fmt.Errorf("rows[%d]: %w", index+1, err)
		}
		if row.SourceRow > scan.SourceRowCount {
			return fmt.Errorf(
				"rows[%d]: sourceRow %d exceeds sourceRowCount %d",
				index+1,
				row.SourceRow,
				scan.SourceRowCount,
			)
		}
		if _, exists := sourceRows[row.SourceRow]; exists {
			return fmt.Errorf("rows[%d]: duplicate sourceRow %d", index+1, row.SourceRow)
		}
		sourceRows[row.SourceRow] = struct{}{}
	}
	return nil
}

func (row Row) UnitBuyout() int64 {
	if row.Buyout <= 0 || row.StackCount <= 0 {
		return 0
	}
	count := int64(row.StackCount)
	return (row.Buyout + count - 1) / count
}

func decodeScan(table *luasv.Table) (Scan, error) {
	var scan Scan
	var err error

	if scan.FormatVersion, err = intField(table, "formatVersion"); err != nil {
		return Scan{}, err
	}
	if scan.FormatVersion != FormatVersion {
		return Scan{}, fmt.Errorf(
			"unsupported formatVersion %d; expected %d",
			scan.FormatVersion,
			FormatVersion,
		)
	}
	if scan.Status, err = stringField(table, "status"); err != nil {
		return Scan{}, err
	}
	if scan.CapturedAt, err = int64Field(table, "capturedAt"); err != nil {
		return Scan{}, err
	}
	if scan.ExportStartedAt, err = int64Field(table, "exportStartedAt"); err != nil {
		return Scan{}, err
	}
	if scan.ExportFinishedAt, err = int64Field(table, "exportFinishedAt"); err != nil {
		return Scan{}, err
	}
	if scan.ExportDurationMS, err = intField(table, "exportDurationMs"); err != nil {
		return Scan{}, err
	}
	if scan.ExportBatchSize, err = intField(table, "exportBatchSize"); err != nil {
		return Scan{}, err
	}
	if scan.Region, err = nonEmptyStringField(table, "region"); err != nil {
		return Scan{}, err
	}
	if scan.Realm, err = nonEmptyStringField(table, "realm"); err != nil {
		return Scan{}, err
	}
	if scan.Faction, err = nonEmptyStringField(table, "faction"); err != nil {
		return Scan{}, err
	}
	if scan.AuctionHouse, err = nonEmptyStringField(table, "auctionHouse"); err != nil {
		return Scan{}, err
	}
	if scan.CaptureZone, err = nonEmptyStringField(table, "captureZone"); err != nil {
		return Scan{}, err
	}
	if scan.CaptureSubzone, err = stringField(table, "captureSubzone"); err != nil {
		return Scan{}, err
	}
	if scan.CaptureUIMapID, err = intField(table, "captureUiMapID"); err != nil {
		return Scan{}, err
	}
	if scan.ScannerName, err = nonEmptyStringField(table, "scannerCharacterName"); err != nil {
		return Scan{}, err
	}
	if scan.ScannerRealm, err = nonEmptyStringField(table, "scannerCharacterRealm"); err != nil {
		return Scan{}, err
	}
	if scan.ScannerGUID, err = nonEmptyStringField(table, "scannerCharacterGUID"); err != nil {
		return Scan{}, err
	}
	if scan.ScannerRegion, err = nonEmptyStringField(table, "scannerRegion"); err != nil {
		return Scan{}, err
	}
	if scan.Source, err = nonEmptyStringField(table, "source"); err != nil {
		return Scan{}, err
	}
	if scan.SourceEvent, err = nonEmptyStringField(table, "sourceEvent"); err != nil {
		return Scan{}, err
	}
	if scan.SourceVersion, err = nonEmptyStringField(table, "sourceVersion"); err != nil {
		return Scan{}, err
	}
	if scan.AddonVersion, err = nonEmptyStringField(table, "addonVersion"); err != nil {
		return Scan{}, err
	}
	if scan.SourceRowCount, err = intField(table, "sourceRowCount"); err != nil {
		return Scan{}, err
	}
	if scan.ExportedRowCount, err = intField(table, "exportedRowCount"); err != nil {
		return Scan{}, err
	}
	if scan.ItemCount, err = intField(table, "itemCount"); err != nil {
		return Scan{}, err
	}
	if scan.Truncated, err = boolField(table, "truncated"); err != nil {
		return Scan{}, err
	}
	if err := validateFields(table, "itemFields", ItemFields); err != nil {
		return Scan{}, err
	}
	if err := validateFields(table, "rowFields", RowFields); err != nil {
		return Scan{}, err
	}

	items, err := decodeItemDictionary(table)
	if err != nil {
		return Scan{}, err
	}
	if scan.ItemCount != len(items) {
		return Scan{}, fmt.Errorf(
			"itemCount %d does not match %d dictionary items",
			scan.ItemCount,
			len(items),
		)
	}

	rawRows, err := tableField(table, "rows")
	if err != nil {
		return Scan{}, err
	}
	rowValues, err := rawRows.Sequence()
	if err != nil {
		return Scan{}, fmt.Errorf("rows: %w", err)
	}

	scan.Rows = make([]Row, 0, len(rowValues))
	for index, value := range rowValues {
		rowTable, ok := value.(*luasv.Table)
		if !ok {
			return Scan{}, fmt.Errorf("rows[%d] is not a table", index+1)
		}
		row, err := decodeRow(rowTable, items)
		if err != nil {
			return Scan{}, fmt.Errorf("rows[%d]: %w", index+1, err)
		}
		scan.Rows = append(scan.Rows, row)
	}

	if err := scan.Validate(); err != nil {
		return Scan{}, err
	}
	return scan, nil
}

func validateFields(table *luasv.Table, name string, expectedFields []string) error {
	rawFields, err := tableField(table, name)
	if err != nil {
		return err
	}
	values, err := rawFields.Sequence()
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if len(values) != len(expectedFields) {
		return fmt.Errorf(
			"%s has %d entries; expected %d",
			name,
			len(values),
			len(expectedFields),
		)
	}
	for index, expected := range expectedFields {
		actual, ok := values[index].(string)
		if !ok || actual != expected {
			return fmt.Errorf(
				"%s[%d] is %q; expected %q",
				name,
				index+1,
				actual,
				expected,
			)
		}
	}
	return nil
}

func decodeItemDictionary(table *luasv.Table) ([]itemIdentity, error) {
	rawItems, err := tableField(table, "items")
	if err != nil {
		return nil, err
	}
	itemValues, err := rawItems.Sequence()
	if err != nil {
		return nil, fmt.Errorf("items: %w", err)
	}

	items := make([]itemIdentity, 0, len(itemValues))
	for index, value := range itemValues {
		itemTable, ok := value.(*luasv.Table)
		if !ok {
			return nil, fmt.Errorf("items[%d] is not a table", index+1)
		}
		values, err := itemTable.Sequence()
		if err != nil {
			return nil, fmt.Errorf("items[%d]: %w", index+1, err)
		}
		if len(values) != len(ItemFields) {
			return nil, fmt.Errorf(
				"items[%d] has %d fields; expected %d",
				index+1,
				len(values),
				len(ItemFields),
			)
		}

		var item itemIdentity
		if item.ItemID, err = valueInt(values[0], "itemId"); err != nil {
			return nil, fmt.Errorf("items[%d]: %w", index+1, err)
		}
		if item.ItemString, err = valueString(values[1], "itemString"); err != nil {
			return nil, fmt.Errorf("items[%d]: %w", index+1, err)
		}
		if item.Name, err = valueString(values[2], "name"); err != nil {
			return nil, fmt.Errorf("items[%d]: %w", index+1, err)
		}
		if item.Quality, err = valueInt(values[3], "quality"); err != nil {
			return nil, fmt.Errorf("items[%d]: %w", index+1, err)
		}
		if item.RequiredLevel, err = valueInt(values[4], "requiredLevel"); err != nil {
			return nil, fmt.Errorf("items[%d]: %w", index+1, err)
		}
		if item.ItemID < 1 {
			return nil, fmt.Errorf("items[%d]: itemId must be positive", index+1)
		}
		items = append(items, item)
	}
	return items, nil
}

func decodeRow(table *luasv.Table, items []itemIdentity) (Row, error) {
	values, err := table.Sequence()
	if err != nil {
		return Row{}, err
	}
	if len(values) != len(RowFields) {
		return Row{}, fmt.Errorf("has %d fields; expected %d", len(values), len(RowFields))
	}

	var row Row
	if row.SourceRow, err = valueInt(values[0], "sourceRow"); err != nil {
		return Row{}, err
	}
	itemReference, err := valueInt(values[1], "itemRef")
	if err != nil {
		return Row{}, err
	}
	if itemReference < 1 || itemReference > len(items) {
		return Row{}, fmt.Errorf(
			"itemRef %d is outside the item dictionary",
			itemReference,
		)
	}
	item := items[itemReference-1]
	row.ItemID = item.ItemID
	row.ItemString = item.ItemString
	row.Name = item.Name
	row.Quality = item.Quality
	row.RequiredLevel = item.RequiredLevel

	if row.StackCount, err = valueInt(values[2], "stackCount"); err != nil {
		return Row{}, err
	}
	if row.MinBid, err = valueInt64(values[3], "minBid"); err != nil {
		return Row{}, err
	}
	if row.MinIncrement, err = valueInt64(values[4], "minIncrement"); err != nil {
		return Row{}, err
	}
	if row.Buyout, err = valueInt64(values[5], "buyout"); err != nil {
		return Row{}, err
	}
	if row.BidAmount, err = valueInt64(values[6], "bidAmount"); err != nil {
		return Row{}, err
	}
	if row.SaleStatus, err = valueInt(values[7], "saleStatus"); err != nil {
		return Row{}, err
	}
	hasAllInfo, err := valueInt(values[8], "hasAllInfo")
	if err != nil {
		return Row{}, err
	}
	if hasAllInfo != 0 && hasAllInfo != 1 {
		return Row{}, fmt.Errorf("hasAllInfo must be 0 or 1")
	}
	row.HasAllInfo = hasAllInfo == 1
	return validateRow(row)
}

func validateRow(row Row) (Row, error) {
	if row.SourceRow < 1 {
		return Row{}, fmt.Errorf("sourceRow must be positive")
	}
	if row.ItemID < 1 {
		return Row{}, fmt.Errorf("itemId must be positive")
	}
	if row.StackCount < 1 {
		return Row{}, fmt.Errorf("stackCount must be positive")
	}
	if row.MinBid < 0 || row.MinIncrement < 0 || row.Buyout < 0 || row.BidAmount < 0 {
		return Row{}, fmt.Errorf("money fields must not be negative")
	}
	return row, nil
}

func validRegion(region string) bool {
	switch region {
	case "us", "eu", "kr", "tw", "cn":
		return true
	default:
		return false
	}
}

func tableField(table *luasv.Table, name string) (*luasv.Table, error) {
	value, ok := table.Field(name)
	if !ok {
		return nil, fmt.Errorf("missing %s", name)
	}
	result, ok := value.(*luasv.Table)
	if !ok {
		return nil, fmt.Errorf("%s is not a table", name)
	}
	return result, nil
}

func stringField(table *luasv.Table, name string) (string, error) {
	value, ok := table.Field(name)
	if !ok {
		return "", fmt.Errorf("missing %s", name)
	}
	return valueString(value, name)
}

func nonEmptyStringField(table *luasv.Table, name string) (string, error) {
	value, err := stringField(table, name)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", name)
	}
	return value, nil
}

func intField(table *luasv.Table, name string) (int, error) {
	value, ok := table.Field(name)
	if !ok {
		return 0, fmt.Errorf("missing %s", name)
	}
	return valueInt(value, name)
}

func int64Field(table *luasv.Table, name string) (int64, error) {
	value, ok := table.Field(name)
	if !ok {
		return 0, fmt.Errorf("missing %s", name)
	}
	return valueInt64(value, name)
}

func boolField(table *luasv.Table, name string) (bool, error) {
	value, ok := table.Field(name)
	if !ok {
		return false, fmt.Errorf("missing %s", name)
	}
	result, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s is not a boolean", name)
	}
	return result, nil
}

func valueString(value luasv.Value, name string) (string, error) {
	result, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s is not a string", name)
	}
	return result, nil
}

func valueInt(value luasv.Value, name string) (int, error) {
	result, err := valueInt64(value, name)
	if err != nil {
		return 0, err
	}
	converted := int(result)
	if int64(converted) != result {
		return 0, fmt.Errorf("%s is outside the platform integer range", name)
	}
	return converted, nil
}

func valueInt64(value luasv.Value, name string) (int64, error) {
	result, ok := value.(int64)
	if !ok {
		return 0, fmt.Errorf("%s is not an integer", name)
	}
	return result, nil
}
