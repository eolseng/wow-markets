WowMarketScan = WowMarketScan or {}

local ADDON_NAME = ...
local SCAN_COMPLETE_EVENT = "get_all_scan_complete"

local DEFAULT_CONFIG = {
  maxExportRows = 100,
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
  if type(config.maxExportRows) ~= "number" or config.maxExportRows < 0 then
    config.maxExportRows = DEFAULT_CONFIG.maxExportRows
  end
  config.maxExportRows = math.floor(config.maxExportRows)

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
  WOW_MARKET_SCAN_DB = WOW_MARKET_SCAN_DB or {}
  WOW_MARKET_SCAN_DB.schemaVersion = 1
  WOW_MARKET_SCAN_DB.config = WOW_MARKET_SCAN_DB.config or {}
  WOW_MARKET_SCAN_DB.pendingScans = WOW_MARKET_SCAN_DB.pendingScans or {}

  ApplyDefaults(WOW_MARKET_SCAN_DB.config, DEFAULT_CONFIG)
  NormalizeConfig(WOW_MARKET_SCAN_DB.config)
  WOW_MARKET_SCAN_DB.config.captureOwner = nil
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
  local rowValue = string.match(normalized, "^rows%s+(.+)$")

  if normalized == "clear" then
    WOW_MARKET_SCAN_DB.pendingScans = {}
    Print("cleared pending scans.")
    return
  end

  if rowValue then
    if rowValue == "all" then
      WOW_MARKET_SCAN_DB.config.maxExportRows = 0
      Print("row limit set to all; full exports may create large SavedVariables files.")
      return
    end

    local numericValue = tonumber(rowValue)
    if numericValue and numericValue >= 1 then
      WOW_MARKET_SCAN_DB.config.maxExportRows = math.floor(numericValue)
      Print("row limit set to " .. WOW_MARKET_SCAN_DB.config.maxExportRows .. ".")
      return
    end

    Print("usage: /wms rows all or /wms rows <positive number>")
    return
  end

  if normalized == "status" or normalized == "" then
    local count = #WOW_MARKET_SCAN_DB.pendingScans
    local rowLimit = WOW_MARKET_SCAN_DB.config.maxExportRows
    local rowLimitText = rowLimit == 0 and "all" or tostring(rowLimit)
    Print(
      WowMarketScan.Capture:GetStatus() .. "; " ..
      count .. " pending scan(s); row limit " .. rowLimitText .. "."
    )
    return
  end

  Print("commands: /wms status, /wms clear, /wms rows all|<count>")
end

WowMarketScan.AddonName = ADDON_NAME
WowMarketScan.ScanCompleteEvent = SCAN_COMPLETE_EVENT
WowMarketScan.GetAddonVersion = GetAddonVersion
WowMarketScan.Print = Print
