local ROW_FIELDS = {
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

local ITEM_FIELDS = {
  "itemId",
  "itemString",
  "name",
  "quality",
  "requiredLevel",
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

local function GetItemReference(active, info, itemLink)
  local itemID = NumberOrZero(info[AUCTION_INFO.itemId])
  local itemString = ExtractItemString(itemLink)
  local name = StringOrEmpty(info[AUCTION_INFO.name])
  local key = itemString
  if key == "" then
    key = tostring(itemID) .. "\031" .. name
  end

  local existing = active.itemLookup[key]
  if existing then
    return existing
  end

  table.insert(active.scan.items, {
    itemID,
    itemString,
    name,
    NumberOrZero(info[AUCTION_INFO.quality]),
    NumberOrZero(info[AUCTION_INFO.requiredLevel]),
  })
  local reference = #active.scan.items
  active.itemLookup[key] = reference
  return reference
end

local function CompactRow(active, sourceRow, entry)
  entry = entry or {}
  local info = entry.auctionInfo or {}

  return {
    sourceRow,
    GetItemReference(active, info, entry.itemLink),
    NumberOrZero(info[AUCTION_INFO.stackCount]),
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
  scan.itemCount = #scan.items
  scan.exportFinishedAt = GetServerTime and GetServerTime() or time()
  if debugprofilestop then
    scan.exportDurationMs = math.floor(debugprofilestop() - active.startedAtMs + 0.5)
  else
    scan.exportDurationMs = 0
  end

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
      CompactRow(active, row, active.rawFullScan[row])
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
  local exportLimit = sourceRowCount
  if config.maxExportRows > 0 then
    exportLimit = math.min(sourceRowCount, config.maxExportRows)
  end
  local getMetadata = C_AddOns and C_AddOns.GetAddOnMetadata or GetAddOnMetadata
  local scanner = GetScannerIdentity()
  local startedAt = GetServerTime and GetServerTime() or time()

  self.active = {
    rawFullScan = rawFullScan,
    exportLimit = exportLimit,
    nextRow = 1,
    startedAtMs = debugprofilestop and debugprofilestop() or 0,
    itemLookup = {},
    scan = {
      formatVersion = 3,
      status = "capturing",
      capturedAt = startedAt,
      exportStartedAt = startedAt,
      exportBatchSize = BATCH_SIZE,
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
      itemCount = 0,
      truncated = exportLimit < sourceRowCount,
      itemFields = ITEM_FIELDS,
      items = {},
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

function Capture:GetStatus()
  if not self.active then
    return "idle"
  end

  return "capturing " ..
    math.min(self.active.nextRow - 1, self.active.exportLimit) ..
    "/" .. self.active.exportLimit
end

WowMarketScan.Capture = Capture
