WowMarketScan = WowMarketScan or {}

local ADDON_NAME = ...
local SCAN_COMPLETE_EVENT = "get_all_scan_complete"
local DATABASE_SCHEMA_VERSION = 1
local SCAN_FORMAT_VERSION = 4

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

local function Print(message)
  DEFAULT_CHAT_FRAME:AddMessage("|cff33ff99WoW Market Scan:|r " .. message)
end

local function GetAddonVersion()
  local getMetadata = C_AddOns and C_AddOns.GetAddOnMetadata or GetAddOnMetadata
  return getMetadata(ADDON_NAME, "Version") or "unknown"
end

local function InitializeDatabase()
  if type(WOW_MARKET_SCAN_DB) ~= "table" or
      WOW_MARKET_SCAN_DB.schemaVersion ~= DATABASE_SCHEMA_VERSION then
    WOW_MARKET_SCAN_DB = {}
  end

  WOW_MARKET_SCAN_DB.schemaVersion = DATABASE_SCHEMA_VERSION
  WOW_MARKET_SCAN_DB.config = WOW_MARKET_SCAN_DB.config or {}
  WOW_MARKET_SCAN_DB.pendingScans = WOW_MARKET_SCAN_DB.pendingScans or {}

  ApplyDefaults(WOW_MARKET_SCAN_DB.config, DEFAULT_CONFIG)
  NormalizeConfig(WOW_MARKET_SCAN_DB.config)
  WOW_MARKET_SCAN_DB.config.maxExportRows = nil

  for index = #WOW_MARKET_SCAN_DB.pendingScans, 1, -1 do
    local scan = WOW_MARKET_SCAN_DB.pendingScans[index]
    if type(scan) ~= "table" or scan.formatVersion ~= SCAN_FORMAT_VERSION then
      table.remove(WOW_MARKET_SCAN_DB.pendingScans, index)
    end
  end
end

local Listener = {}

function Listener:ReceiveEvent(eventName, rawFullScan)
  if eventName ~= SCAN_COMPLETE_EVENT then
    return
  end

  WowMarketScan.Capture:Begin(rawFullScan)
end

local Frame = CreateFrame("Frame")
Frame:RegisterEvent("ADDON_LOADED")
Frame:SetScript("OnEvent", function(_, event, loadedAddon)
  if event ~= "ADDON_LOADED" or loadedAddon ~= ADDON_NAME then
    return
  end

  InitializeDatabase()

  if not Auctionator or not Auctionator.EventBus then
    Print("Auctionator event bus is unavailable; capture is disabled.")
    return
  end

  Auctionator.EventBus:Register(Listener, { SCAN_COMPLETE_EVENT })
  Print("ready; waiting for an Auctionator full scan.")
end)

SLASH_WOWMARKETSCAN1 = "/wms"
SlashCmdList.WOWMARKETSCAN = function(command)
  local normalized = string.lower(command or "")

  if normalized == "clear" then
    WOW_MARKET_SCAN_DB.pendingScans = {}
    Print("cleared pending scans.")
    return
  end

  if normalized == "status" or normalized == "" then
    local count = #WOW_MARKET_SCAN_DB.pendingScans
    Print(
      WowMarketScan.Capture:GetStatus() .. "; " ..
      count .. " pending scan(s); complete scan export enabled."
    )
    return
  end

  if normalized == "location" then
    local location = WowMarketScan.Capture:GetLocation()
    Print(
      "location: " .. location.zone ..
      " / " .. location.subzone ..
      " (map " .. location.uiMapID .. "); AH " ..
      location.auctionHouse .. "."
    )
    return
  end

  Print("commands: /wms status, /wms location, /wms clear")
end

WowMarketScan.AddonName = ADDON_NAME
WowMarketScan.ScanCompleteEvent = SCAN_COMPLETE_EVENT
WowMarketScan.ScanFormatVersion = SCAN_FORMAT_VERSION
WowMarketScan.GetAddonVersion = GetAddonVersion
WowMarketScan.Print = Print
