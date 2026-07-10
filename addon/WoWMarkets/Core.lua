WoWMarkets = WoWMarkets or {}

local ADDON_NAME = ...
local DISPLAY_NAME = "WoW Markets"
local SCAN_COMPLETE_EVENT = "get_all_scan_complete"
local DATABASE_SCHEMA_VERSION = 1
local SCAN_FORMAT_VERSION = 5

local DEFAULT_CONFIG = {
  maxPendingScans = 3,
}

local function ApplyDefaults(target, defaults)
  for key, value in pairs(defaults) do
    if target[key] == nil then
      target[key] = value
    end
  end
end

local function NormalizeConfig(config)
  if type(config.maxPendingScans) ~= "number" or config.maxPendingScans < 1 then
    config.maxPendingScans = DEFAULT_CONFIG.maxPendingScans
  end
  config.maxPendingScans = math.floor(config.maxPendingScans)
end

local MESSAGE_PREFIXES = {
  error = "|cffff5555Error:|r ",
  warning = "|cffffcc00Warning:|r ",
}

local function Print(message, level)
  DEFAULT_CHAT_FRAME:AddMessage(
    "|cff33ff99" .. DISPLAY_NAME .. ":|r " ..
    (MESSAGE_PREFIXES[level] or "") .. message
  )
end

local function FormatNumber(value)
  local formatted = tostring(value or 0)
  while true do
    local updated, replacements = string.gsub(formatted, "^(-?%d+)(%d%d%d)", "%1,%2")
    formatted = updated
    if replacements == 0 then
      return formatted
    end
  end
end

local function GetAddonVersion()
  local getMetadata = C_AddOns and C_AddOns.GetAddOnMetadata or GetAddOnMetadata
  return getMetadata(ADDON_NAME, "Version") or "unknown"
end

local function InitializeDatabase()
  local hadDatabase = WOW_MARKETS_DB ~= nil
  local resetDatabase = type(WOW_MARKETS_DB) ~= "table" or
    WOW_MARKETS_DB.schemaVersion ~= DATABASE_SCHEMA_VERSION
  if resetDatabase then
    WOW_MARKETS_DB = {}
  end

  WOW_MARKETS_DB.schemaVersion = DATABASE_SCHEMA_VERSION
  if type(WOW_MARKETS_DB.config) ~= "table" then
    WOW_MARKETS_DB.config = {}
  end
  if type(WOW_MARKETS_DB.pendingScans) ~= "table" then
    WOW_MARKETS_DB.pendingScans = {}
  end

  ApplyDefaults(WOW_MARKETS_DB.config, DEFAULT_CONFIG)
  NormalizeConfig(WOW_MARKETS_DB.config)
  WOW_MARKETS_DB.config.maxExportRows = nil

  local removedScans = 0
  for index = #WOW_MARKETS_DB.pendingScans, 1, -1 do
    local scan = WOW_MARKETS_DB.pendingScans[index]
    if type(scan) ~= "table" or scan.formatVersion ~= SCAN_FORMAT_VERSION then
      table.remove(WOW_MARKETS_DB.pendingScans, index)
      removedScans = removedScans + 1
    end
  end

  return not hadDatabase, resetDatabase and hadDatabase, removedScans
end

local Listener = {}

function Listener:ReceiveEvent(eventName, rawFullScan)
  if eventName ~= SCAN_COMPLETE_EVENT then
    return
  end

  WoWMarkets.Capture:Begin(rawFullScan)
end

local Frame = CreateFrame("Frame")
Frame:RegisterEvent("ADDON_LOADED")
Frame:SetScript("OnEvent", function(_, event, loadedAddon)
  if event ~= "ADDON_LOADED" or loadedAddon ~= ADDON_NAME then
    return
  end

  local isFirstRun, resetDatabase, removedScans = InitializeDatabase()

  if resetDatabase then
    Print("Stored data used an unsupported format and was reset.", "warning")
  elseif removedScans > 0 then
    Print(
      FormatNumber(removedScans) .. " incompatible stored scan(s) were removed.",
      "warning"
    )
  end

  if not Auctionator or not Auctionator.EventBus then
    WoWMarkets.CaptureEnabled = false
    Print("Auctionator is unavailable; scan capture is disabled.", "error")
    return
  end

  WoWMarkets.CaptureEnabled = true
  Auctionator.EventBus:Register(Listener, { SCAN_COMPLETE_EVENT })
  if isFirstRun then
    Print("Ready. Run an Auctionator full scan when you are at the Auction House.")
  end
end)

SLASH_WOWMARKETS1 = "/wm"
SLASH_WOWMARKETS2 = "/wms"
SlashCmdList.WOWMARKETS = function(command)
  local normalized = string.lower(string.match(command or "", "^%s*(.-)%s*$"))

  if normalized == "clear" then
    local count = #WOW_MARKETS_DB.pendingScans
    if count == 0 then
      Print("There are no stored scans to clear.")
      return
    end
    Print(
      "This will remove " .. FormatNumber(count) ..
      " stored scan(s). Type /wm clear confirm to continue.",
      "warning"
    )
    return
  end

  if normalized == "clear confirm" then
    local count = WoWMarkets.Capture:ClearStoredScans()
    Print("Removed " .. FormatNumber(count) .. " stored scan(s).")
    return
  end

  if normalized == "status" or normalized == "" then
    if WoWMarkets.CaptureEnabled == false then
      Print("Capture is disabled because Auctionator is unavailable.", "error")
    else
      Print(WoWMarkets.Capture:GetStatus())
    end
    return
  end

  if normalized == "location" then
    local location = WoWMarkets.Capture:GetLocation()
    local place = location.subzone
    if place == "" then
      place = location.zone
    elseif location.zone ~= "" and location.zone ~= location.subzone then
      place = place .. ", " .. location.zone
    end
    if place == "" then
      place = "unknown location"
    end

    local auctionHouse = location.auctionHouse == "neutral" and "Neutral AH" or "Faction AH"
    local mapDetail = location.uiMapID > 0 and " (map " .. location.uiMapID .. ")" or ""
    Print(auctionHouse .. " - " .. place .. mapDetail .. ".")
    return
  end

  Print("Commands: /wm status or /wm location.")
end

WoWMarkets.AddonName = ADDON_NAME
WoWMarkets.DisplayName = DISPLAY_NAME
WoWMarkets.ScanCompleteEvent = SCAN_COMPLETE_EVENT
WoWMarkets.ScanFormatVersion = SCAN_FORMAT_VERSION
WoWMarkets.GetAddonVersion = GetAddonVersion
WoWMarkets.FormatNumber = FormatNumber
WoWMarkets.Print = Print
