local ROW_FIELDS = {
  "sourceRow",
  "itemId",
  "itemString",
  "name",
  "stackCount",
  "quality",
  "requiredLevel",
  "minBid",
  "minIncrement",
  "buyout",
  "bidAmount",
  "saleStatus",
  "hasAllInfo",
}

local AUCTION_INFO = {
  name = 1,
  stackCount = 3,
  quality = 4,
  requiredLevel = 6,
  minBid = 8,
  minIncrement = 9,
  buyout = 10,
  bidAmount = 11,
  saleStatus = 16,
  itemId = 17,
  hasAllInfo = 18,
}

local BATCH_SIZE = 250
local REGION_CODES = {
  [1] = "us",
  [2] = "kr",
  [3] = "eu",
  [4] = "tw",
  [5] = "cn",
}

local Capture = {
  active = nil,
}

local function NumberOrZero(value)
  if type(value) == "number" then
    return value
  end
  return 0
end

local function StringOrEmpty(value)
  if type(value) == "string" then
    return value
  end
  return ""
end

local function BooleanNumber(value)
  return value and 1 or 0
end

local function ExtractItemString(itemLink)
  if type(itemLink) ~= "string" then
    return ""
  end

  return string.match(itemLink, "|H(item:[^|]+)|h") or ""
end

local function CompactRow(sourceRow, entry)
  entry = entry or {}
  local info = entry.auctionInfo or {}

  return {
    sourceRow,
    NumberOrZero(info[AUCTION_INFO.itemId]),
    ExtractItemString(entry.itemLink),
    StringOrEmpty(info[AUCTION_INFO.name]),
    NumberOrZero(info[AUCTION_INFO.stackCount]),
    NumberOrZero(info[AUCTION_INFO.quality]),
    NumberOrZero(info[AUCTION_INFO.requiredLevel]),
    NumberOrZero(info[AUCTION_INFO.minBid]),
    NumberOrZero(info[AUCTION_INFO.minIncrement]),
    NumberOrZero(info[AUCTION_INFO.buyout]),
    NumberOrZero(info[AUCTION_INFO.bidAmount]),
    NumberOrZero(info[AUCTION_INFO.saleStatus]),
    BooleanNumber(info[AUCTION_INFO.hasAllInfo]),
  }
end

local function GetScannerIdentity()
  local characterName
  local characterRealm

  if UnitFullName then
    characterName, characterRealm = UnitFullName("player")
  end
  characterName = characterName or UnitName("player") or ""
  characterRealm = characterRealm or GetRealmName() or ""

  local regionID = GetCurrentRegion and GetCurrentRegion() or 0
  return {
    name = characterName,
    realm = characterRealm,
    guid = UnitGUID("player") or "",
    region = REGION_CODES[regionID] or "unknown",
  }
end

local function TrimQueue()
  local pendingScans = WOW_MARKET_SCAN_DB.pendingScans
  local maxPendingScans = WOW_MARKET_SCAN_DB.config.maxPendingScans

  while #pendingScans >= maxPendingScans do
    table.remove(pendingScans, 1)
  end
end

local function Finish()
  local active = Capture.active
  local scan = active.scan

  scan.status = "ready"
  scan.exportedRowCount = #scan.rows

  TrimQueue()
  table.insert(WOW_MARKET_SCAN_DB.pendingScans, scan)

  Capture.active = nil
  WowMarketScan.Print(
    "captured " .. scan.exportedRowCount .. " of " ..
    scan.sourceRowCount .. " rows."
  )
end

local function ProcessBatch()
  local active = Capture.active
  if not active then
    return
  end

  local stopAt = math.min(active.nextRow + BATCH_SIZE - 1, active.exportLimit)
  for row = active.nextRow, stopAt do
    table.insert(
      active.scan.rows,
      CompactRow(row, active.rawFullScan[row])
    )
  end

  active.nextRow = stopAt + 1
  if active.nextRow > active.exportLimit then
    Finish()
    return
  end

  C_Timer.After(0, ProcessBatch)
end

function Capture:Begin(rawFullScan)
  if self.active then
    WowMarketScan.Print("ignored a scan because capture is already active.")
    return
  end

  if type(rawFullScan) ~= "table" then
    WowMarketScan.Print("received an invalid Auctionator scan payload.")
    return
  end

  local config = WOW_MARKET_SCAN_DB.config
  local sourceRowCount = #rawFullScan
  local exportLimit = math.min(sourceRowCount, config.maxExportRows)
  local getMetadata = C_AddOns and C_AddOns.GetAddOnMetadata or GetAddOnMetadata
  local scanner = GetScannerIdentity()

  self.active = {
    rawFullScan = rawFullScan,
    exportLimit = exportLimit,
    nextRow = 1,
    scan = {
      formatVersion = 2,
      status = "capturing",
      capturedAt = GetServerTime and GetServerTime() or time(),
      realm = GetRealmName() or "",
      faction = UnitFactionGroup("player") or "",
      auctionHouse = "unknown",
      scannerCharacterName = scanner.name,
      scannerCharacterRealm = scanner.realm,
      scannerCharacterGUID = scanner.guid,
      scannerRegion = scanner.region,
      source = "Auctionator",
      sourceEvent = WowMarketScan.ScanCompleteEvent,
      sourceVersion = getMetadata("Auctionator", "Version") or "unknown",
      addonVersion = WowMarketScan.GetAddonVersion(),
      sourceRowCount = sourceRowCount,
      exportedRowCount = 0,
      truncated = exportLimit < sourceRowCount,
      rowFields = ROW_FIELDS,
      rows = {},
    },
  }

  if exportLimit == 0 then
    Finish()
    return
  end

  C_Timer.After(0, ProcessBatch)
end

function Capture:IsActive()
  return self.active ~= nil
end

WowMarketScan.Capture = Capture
