local addonRoot = assert(arg[1], "addon root argument is required")

local messages = {}
local timerCallbacks = {}
local registeredListener
local addonFrame
local profileTime = 1000

local function AssertEqual(actual, expected, context)
  if actual ~= expected then
    error((context or "values differ") .. ": expected " .. tostring(expected) ..
      ", got " .. tostring(actual), 2)
  end
end

local function AssertContains(value, expected, context)
  if not string.find(value or "", expected, 1, true) then
    error((context or "text differs") .. ": expected to find " .. expected ..
      " in " .. tostring(value), 2)
  end
end

local function LastMessage()
  return messages[#messages]
end

local function RunNextTimer()
  local callback = table.remove(timerCallbacks, 1)
  assert(callback, "expected a pending timer")
  profileTime = profileTime + 500
  callback()
end

local function RunAllTimers()
  while #timerCallbacks > 0 do
    RunNextTimer()
  end
end

DEFAULT_CHAT_FRAME = {
  AddMessage = function(_, message)
    table.insert(messages, message)
  end,
}

C_AddOns = {
  GetAddOnMetadata = function(addonName, field)
    if addonName == "Auctionator" and field == "Version" then
      return "test-auctionator"
    end
    if addonName == "WoWMarkets" and field == "Version" then
      return "test-addon"
    end
  end,
}

Auctionator = {
  EventBus = {
    Register = function(_, listener)
      registeredListener = listener
    end,
  },
}

C_Timer = {
  After = function(_, callback)
    table.insert(timerCallbacks, callback)
  end,
}

C_Map = {
  GetBestMapForUnit = function()
    return 1434
  end,
}

function CreateFrame()
  addonFrame = {
    RegisterEvent = function() end,
    SetScript = function(self, _, callback)
      self.callback = callback
    end,
  }
  return addonFrame
end

function UnitFullName()
  return "Testchar", "Test Realm"
end

function UnitName()
  return "Testchar"
end

function GetRealmName()
  return "Test Realm"
end

function GetCurrentRegion()
  return 3
end

function UnitGUID()
  return "Player-0000-00000001"
end

function UnitFactionGroup()
  return "Alliance"
end

function GetZoneText()
  return "Stranglethorn Vale"
end

function GetSubZoneText()
  return "Booty Bay"
end

function GetServerTime()
  return 1781344800
end

function debugprofilestop()
  return profileTime
end

function time()
  return 1781344800
end

SlashCmdList = {}

local coreChunk = assert(loadfile(addonRoot .. "/WoWMarkets/Core.lua"))
coreChunk("WoWMarkets")
assert(loadfile(addonRoot .. "/WoWMarkets/Capture.lua"))()

addonFrame.callback(nil, "ADDON_LOADED", "WoWMarkets")
AssertContains(LastMessage(), "WoW Markets:", "display name")
AssertContains(LastMessage(), "Ready.", "first-run message")
AssertEqual(SLASH_WOWMARKETS1, "/wm", "primary slash command")
AssertEqual(SLASH_WOWMARKETS2, "/wms", "legacy slash command")
AssertEqual(WoWMarkets.FormatNumber(1234567), "1,234,567", "listing count formatting")
assert(registeredListener, "Auctionator listener was not registered")

SlashCmdList.WOWMARKETS(" status ")
AssertContains(LastMessage(), "Ready | 0 scans stored", "trimmed status command")

SlashCmdList.WOWMARKETS("location")
AssertContains(LastMessage(), "Neutral AH - Booty Bay, Stranglethorn Vale", "location message")

local rawScan = {}
for index = 1, 501 do
  rawScan[index] = {
    auctionInfo = {
      [1] = "Test Item",
      [3] = 20,
      [4] = 1,
      [6] = 0,
      [8] = 100,
      [9] = 5,
      [10] = 200,
      [11] = 0,
      [16] = 0,
      [17] = 1000 + index,
      [18] = true,
    },
    itemLink = "|Hitem:" .. (1000 + index) .. ":0:0:0:0:0:0:0|h[Test Item]|h",
  }
end

registeredListener:ReceiveEvent("get_all_scan_complete", rawScan)
AssertContains(LastMessage(), "Preparing 501 listings", "capture start message")
RunNextTimer()
SlashCmdList.WOWMARKETS("status")
AssertContains(LastMessage(), "250 / 501 listings (50%)", "capture progress status")
RunAllTimers()
AssertContains(LastMessage(), "Type /reload or log out", "capture handoff message")
AssertEqual(#WOW_MARKETS_DB.pendingScans, 1, "stored scan count")

SlashCmdList.WOWMARKETS("status")
AssertContains(LastMessage(), "latest scan needs /reload", "ready status")

SlashCmdList.WOWMARKETS("clear")
AssertEqual(#WOW_MARKETS_DB.pendingScans, 1, "clear requires confirmation")
AssertContains(LastMessage(), "/wm clear confirm", "clear confirmation message")
SlashCmdList.WOWMARKETS("clear confirm")
AssertEqual(#WOW_MARKETS_DB.pendingScans, 0, "confirmed clear")

WOW_MARKETS_DB.config.maxPendingScans = 2
for _ = 1, 3 do
  registeredListener:ReceiveEvent("get_all_scan_complete", { rawScan[1] })
  RunAllTimers()
end
AssertEqual(#WOW_MARKETS_DB.pendingScans, 2, "queue limit")
AssertContains(LastMessage(), "earlier scan from this session", "same-session eviction warning")

SlashCmdList.WOWMARKETS("unknown")
AssertContains(LastMessage(), "/wm status or /wm location", "concise command help")

WOW_MARKETS_DB = {
  schemaVersion = 1,
  config = "invalid",
  pendingScans = "invalid",
}
addonFrame.callback(nil, "ADDON_LOADED", "WoWMarkets")
AssertEqual(type(WOW_MARKETS_DB.config), "table", "config normalization")
AssertEqual(type(WOW_MARKETS_DB.pendingScans), "table", "queue normalization")

print("addon tests passed")
