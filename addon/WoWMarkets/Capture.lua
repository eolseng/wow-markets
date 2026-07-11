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

local NEUTRAL_AUCTION_HOUSE_MAPS = {
  [1434] = true, -- Stranglethorn Vale / Booty Bay
  [1446] = true, -- Tanaris / Gadgetzan
  [1452] = true, -- Winterspring / Everlook
}

local NEUTRAL_AUCTION_HOUSE_ZONES = {
  ["Stranglethorn Vale"] = true,
  ["Tanaris"] = true,
  ["Winterspring"] = true,
}

local NEUTRAL_AUCTION_HOUSE_SUBZONES = {
  ["Booty Bay"] = true,
  ["Gadgetzan"] = true,
  ["Everlook"] = true,
}

local Capture = {
  active = nil,
  capturesThisSession = 0,
  needsHandoff = false,
  sessionScans = {},
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
  if type(entry) ~= "table" then
    entry = {}
  end
  local info = type(entry.auctionInfo) == "table" and entry.auctionInfo or {}

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

local function GetCaptureLocation()
  local zone = GetZoneText and GetZoneText() or ""
  local subzone = GetSubZoneText and GetSubZoneText() or ""
  local uiMapID = 0

  if C_Map and C_Map.GetBestMapForUnit then
    uiMapID = C_Map.GetBestMapForUnit("player") or 0
  end

  local isNeutral =
    NEUTRAL_AUCTION_HOUSE_MAPS[uiMapID] or
    NEUTRAL_AUCTION_HOUSE_ZONES[zone] or
    NEUTRAL_AUCTION_HOUSE_SUBZONES[subzone]

  return {
    zone = zone,
    subzone = subzone,
    uiMapID = uiMapID,
    auctionHouse = isNeutral and "neutral" or "faction",
  }
end

local function TrimQueue()
  local pendingScans = WOW_MARKETS_DB.pendingScans
  local maxPendingScans = WOW_MARKETS_DB.config.maxPendingScans

  local removedScan
  while #pendingScans >= maxPendingScans do
    removedScan = table.remove(pendingScans, 1)
  end
  return removedScan
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

  local removedScan = TrimQueue()
  table.insert(WOW_MARKETS_DB.pendingScans, scan)

  Capture.active = nil
  Capture.capturesThisSession = Capture.capturesThisSession + 1
  Capture.needsHandoff = true
  Capture.sessionScans[scan] = true

  local duration = string.format("%.1fs", scan.exportDurationMs / 1000)
  WoWMarkets.Print(
    "Scan ready: " .. WoWMarkets.FormatNumber(scan.exportedRowCount) ..
    " listings processed in " .. duration ..
    ". Type /reload or log out to send it to WoW Markets Companion."
  )

  if removedScan then
    if Capture.sessionScans[removedScan] then
      Capture.sessionScans[removedScan] = nil
      WoWMarkets.Print(
        "An earlier scan from this session was replaced before /reload.",
        "warning"
      )
    else
      WoWMarkets.Print("The oldest stored scan was rotated out to keep the latest scans.")
    end
  end

  if Capture.capturesThisSession == WOW_MARKETS_DB.config.maxPendingScans then
    WoWMarkets.Print(
      WoWMarkets.FormatNumber(Capture.capturesThisSession) ..
      " scans were captured this session. Type /reload before scanning again.",
      "warning"
    )
  end
end

local function ProcessBatch()
  local active = Capture.active
  if not active then
    return
  end

  local stopAt = math.min(active.nextRow + BATCH_SIZE - 1, active.sourceRowCount)
  for row = active.nextRow, stopAt do
    table.insert(
      active.scan.rows,
      CompactRow(active, row, active.rawFullScan[row])
    )
  end

  active.nextRow = stopAt + 1
  if active.nextRow > active.sourceRowCount then
    Finish()
    return
  end

  C_Timer.After(0, ProcessBatch)
end

function Capture:CompleteNow()
  local active = self.active
  if not active then
    return false
  end

  for row = active.nextRow, active.sourceRowCount do
    table.insert(
      active.scan.rows,
      CompactRow(active, row, active.rawFullScan[row])
    )
  end
  active.nextRow = active.sourceRowCount + 1
  Finish()
  return true
end

function Capture:Begin(rawFullScan)
  if self.active then
    WoWMarkets.Print(
      "A scan was ignored because another capture is still being prepared.",
      "warning"
    )
    return
  end

  if type(rawFullScan) ~= "table" then
    WoWMarkets.Print("Auctionator returned an invalid scan; nothing was captured.", "error")
    return
  end

  local sourceRowCount = #rawFullScan
  local getMetadata = C_AddOns and C_AddOns.GetAddOnMetadata or GetAddOnMetadata
  local scanner = GetScannerIdentity()
  if scanner.region == "unknown" then
    WoWMarkets.Print("The game region could not be determined; nothing was captured.", "error")
    return
  end
  local location = GetCaptureLocation()
  local startedAt = GetServerTime and GetServerTime() or time()

  self.active = {
    rawFullScan = rawFullScan,
    sourceRowCount = sourceRowCount,
    nextRow = 1,
    startedAtMs = debugprofilestop and debugprofilestop() or 0,
    itemLookup = {},
    scan = {
      formatVersion = WoWMarkets.ScanFormatVersion,
      status = "capturing",
      capturedAt = startedAt,
      exportStartedAt = startedAt,
      exportBatchSize = BATCH_SIZE,
      region = scanner.region,
      realm = GetRealmName() or "",
      faction = UnitFactionGroup("player") or "",
      auctionHouse = location.auctionHouse,
      captureZone = location.zone,
      captureSubzone = location.subzone,
      captureUiMapID = location.uiMapID,
      scannerCharacterName = scanner.name,
      scannerCharacterRealm = scanner.realm,
      scannerCharacterGUID = scanner.guid,
      scannerRegion = scanner.region,
      source = "Auctionator",
      sourceEvent = WoWMarkets.ScanCompleteEvent,
      sourceVersion = getMetadata("Auctionator", "Version") or "unknown",
      addonVersion = WoWMarkets.GetAddonVersion(),
      sourceRowCount = sourceRowCount,
      exportedRowCount = 0,
      itemCount = 0,
      truncated = false,
      itemFields = ITEM_FIELDS,
      items = {},
      rowFields = ROW_FIELDS,
      rows = {},
    },
  }

  WoWMarkets.Print(
    "Auctionator scan complete. Preparing " ..
    WoWMarkets.FormatNumber(sourceRowCount) ..
    " listings - please keep the game open."
  )

  if sourceRowCount == 0 then
    Finish()
    return
  end

  C_Timer.After(0, ProcessBatch)
end

function Capture:IsActive()
  return self.active ~= nil
end

function Capture:GetLocation()
  return GetCaptureLocation()
end

function Capture:GetStatus()
  if not self.active then
    local count = #WOW_MARKETS_DB.pendingScans
    local label = count == 1 and "scan" or "scans"
    if self.needsHandoff then
      return "Ready | " .. WoWMarkets.FormatNumber(count) .. " " .. label ..
        " stored | latest scan needs /reload"
    end
    return "Ready | " .. WoWMarkets.FormatNumber(count) .. " " .. label ..
      " stored | run an Auctionator full scan"
  end

  local processed = math.min(self.active.nextRow - 1, self.active.sourceRowCount)
  local percent = 100
  if self.active.sourceRowCount > 0 then
    percent = math.floor(processed / self.active.sourceRowCount * 100 + 0.5)
  end
  return "Preparing scan | " .. WoWMarkets.FormatNumber(processed) .. " / " ..
    WoWMarkets.FormatNumber(self.active.sourceRowCount) ..
    " listings (" .. percent .. "%)"
end

function Capture:ClearStoredScans()
  local count = #WOW_MARKETS_DB.pendingScans
  WOW_MARKETS_DB.pendingScans = {}
  self.needsHandoff = false
  self.capturesThisSession = 0
  self.sessionScans = {}
  return count
end

WoWMarkets.Capture = Capture
