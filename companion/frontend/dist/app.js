import {
  accountListSignature,
  deriveTokenRemovalView,
  deriveView,
  deriveUpdaterView,
  heroAnnouncementSignature,
} from "./view-model.mjs"

const state = {
  busy: false,
  accountsSignature: "",
  heroSignature: "",
  page: "overview",
  removeTokenConfirmationOpen: false,
  replaceTokenOpen: false,
  snapshot: null,
}

const $ = (selector) => document.querySelector(selector)
const $$ = (selector) => [...document.querySelectorAll(selector)]

const elements = {
  accountList: $("#account-list"),
  addonPath: $("#expected-addon-path"),
  appVersion: $("#app-version"),
  backgroundNote: $("#background-note"),
  dashboard: $("#dashboard"),
  failedCount: $("#failed-count"),
  heroCard: $("#hero-card"),
  heroEyebrow: $("#hero-eyebrow"),
  heroFacts: $("#hero-facts"),
  heroSummary: $("#hero-summary"),
  heroSymbol: $("#hero-symbol"),
  heroTitle: $("#hero-title"),
  homeUpdateButton: $("#home-update-button"),
  homeUpdateMessage: $("#home-update-message"),
  homeUpdateNotice: $("#home-update-notice"),
  homeUpdateTitle: $("#home-update-title"),
  launchAtLoginSection: $("#launch-at-login-section"),
  launchAtLoginToggle: $("#launch-at-login-toggle"),
  lastUploadContent: $("#last-upload-content"),
  lastUploadStatus: $("#last-upload-status"),
  notice: $("#notice"),
  overviewContent: $("#overview-content"),
  overviewPage: $("#overview-page"),
  recentList: $("#recent-list"),
  removeTokenButton: $("#remove-token-button"),
  removeTokenCancelButton: $("#remove-token-cancel-button"),
  removeTokenConfirmButton: $("#remove-token-confirm-button"),
  removeTokenConfirmation: $("#remove-token-confirmation"),
  replaceTokenButton: $("#replace-token-button"),
  runtimeDot: $("#runtime-dot"),
  runtimeLabel: $("#runtime-label"),
  settingsAccount: $("#settings-account"),
  settingsAddonState: $("#settings-addon-state"),
  settingsButton: $("#settings-button"),
  settingsDataDir: $("#settings-data-dir"),
  settingsPage: $("#settings-page"),
  settingsScanState: $("#settings-scan-state"),
  settingsTokenDetail: $("#settings-token-detail"),
  settingsTokenForm: $("#settings-token-form"),
  settingsTokenInput: $("#settings-token-input"),
  settingsTokenStatus: $("#settings-token-status"),
  settingsWatcherState: $("#settings-watcher-state"),
  settingsWowPath: $("#settings-wow-path"),
  settingsWowStatus: $("#settings-wow-status"),
  setupCard: $("#setup-card"),
  startingCard: $("#starting-card"),
  startingPhase: $("#starting-phase"),
  tokenForm: $("#token-form"),
  tokenInput: $("#token-input"),
  updateChannel: $("#update-channel"),
  updateCheckButton: $("#update-check-button"),
  updateDeferButton: $("#update-defer-button"),
  updateInstallButton: $("#update-install-button"),
  updateMessage: $("#update-message"),
  updateStatus: $("#update-status"),
  uploadedCount: $("#uploaded-count"),
  waitingCount: $("#waiting-count"),
}

function backend() {
  const app = window.go?.main?.App
  if (!app) throw new Error("WoW Markets Companion is still starting")
  return app
}

async function refresh() {
  if (state.busy) return
  try {
    render(await backend().Snapshot())
  } catch (error) {
    if (state.snapshot) showError(error)
  }
}

function render(snapshot) {
  state.snapshot = snapshot
  const view = deriveView(snapshot)
  renderPage()
  renderRuntime(snapshot, view)
  renderOverview(snapshot, view)
  renderSettings(snapshot)
}

function renderPage() {
  elements.overviewPage.hidden = state.page !== "overview"
  elements.settingsPage.hidden = state.page !== "settings"
  elements.settingsButton.disabled = !state.snapshot || state.snapshot.initializing
}

function renderRuntime(snapshot, view) {
  elements.runtimeDot.className = "runtime-dot"
  if (snapshot.initializing) {
    elements.runtimeLabel.textContent = "Starting"
    elements.runtimeDot.classList.add("loading")
    return
  }
  if (view.tone === "danger") {
    elements.runtimeLabel.textContent = "Needs attention"
    elements.runtimeDot.classList.add("error")
    return
  }
  if (view.mode === "retrying" || view.setupStep === "addon") {
    elements.runtimeLabel.textContent = view.mode === "retrying" ? "Retrying" : "Setup needed"
    elements.runtimeDot.classList.add("warning")
    return
  }
  if (snapshot.ready && snapshot.running) {
    elements.runtimeLabel.textContent = "Running"
    elements.runtimeDot.classList.add("active")
    return
  }
  elements.runtimeLabel.textContent = snapshot.ready ? "Starting watcher" : "Setup needed"
  elements.runtimeDot.classList.add("loading")
}

function renderOverview(snapshot, view) {
  const starting = view.mode === "starting"
  elements.startingCard.hidden = !starting
  elements.overviewContent.hidden = starting
  if (starting) {
    elements.startingPhase.textContent = view.summary
    return
  }

  const signature = heroAnnouncementSignature(view)
  if (signature !== state.heroSignature) {
    state.heroSignature = signature
    elements.heroCard.className = `hero-card tone-${view.tone}`
    elements.heroSymbol.className = `hero-symbol ${symbolClass(view)}`
    elements.heroEyebrow.textContent = view.eyebrow
    elements.heroTitle.textContent = view.title
    elements.heroSummary.textContent = view.summary
    renderFacts(elements.heroFacts, view.scan)
  }

  const setup = view.mode === "setup"
  elements.setupCard.hidden = !setup
  elements.dashboard.hidden = setup
  elements.backgroundNote.textContent = setup
    ? "Setup is saved as you go. You can return to it at any time."
    : "The companion keeps watching when you close this window."
  if (setup) renderSetup(snapshot, view.setupStep)
  else renderDashboard(snapshot)
}

function symbolClass(view) {
  if (view.tone === "danger") return "danger"
  if (view.tone === "warning") return "warning"
  if (["uploaded", "waiting"].includes(view.mode)) return "success"
  return "active"
}

function renderSetup(snapshot, currentStep) {
  const checks = [
    ["token", snapshot.token_stored, snapshot.token_prefix ? `Stored · ${snapshot.token_prefix}…` : "Required"],
    ["wow", snapshot.wow_detected, snapshot.wow_detected ? basename(snapshot.wow_install_path) : "Required"],
    ["addon", snapshot.addon_detected, snapshot.addon_detected ? "Installed" : "Required"],
    ["saved-variables", snapshot.saved_variables_detected, snapshot.saved_variables_detected ? `${snapshot.scan_file_count || 1} found` : "Required"],
  ]
  for (const [name, complete, detail] of checks) {
    const check = $(`#check-${name}`)
    check.classList.toggle("complete", complete)
    const normalized = name === "saved-variables" ? "saved_variables" : name
    check.classList.toggle("current", normalized === currentStep)
    check.querySelector("small").textContent = detail
  }
  for (const panel of $$('[data-setup-panel]')) {
    panel.hidden = panel.dataset.setupPanel !== currentStep
  }
  elements.addonPath.textContent = snapshot.addon_path || "Select your World of Warcraft folder first"
}

function renderDashboard(snapshot) {
  elements.uploadedCount.textContent = formatNumber(snapshot.uploaded_count)
  elements.waitingCount.textContent = formatNumber((snapshot.queued_count || 0) + (snapshot.uploading_count || 0))
  elements.failedCount.textContent = formatNumber(snapshot.failed_count)

  const lastUpload = snapshot.last_upload
  if (!lastUpload) {
    elements.lastUploadStatus.textContent = "Waiting"
    elements.lastUploadStatus.className = "status-badge active"
    elements.lastUploadContent.innerHTML = `<div class="empty-state"><strong>No uploads yet</strong><span>Run an Auctionator full scan, then /reload or log out.</span></div>`
  } else {
    elements.lastUploadStatus.textContent = uploadStatusLabel(lastUpload)
    elements.lastUploadStatus.className = `status-badge ${uploadTone(lastUpload)}`
    elements.lastUploadContent.innerHTML = summaryMarkup(lastUpload)
  }

  elements.recentList.replaceChildren()
  const recent = snapshot.recent_uploads || []
  if (recent.length === 0) {
    const empty = document.createElement("div")
    empty.className = "empty-state"
    empty.innerHTML = "<strong>Waiting for your first scan</strong><span>New uploads will appear here.</span>"
    elements.recentList.append(empty)
    return
  }
  for (const upload of recent.slice(0, 5)) {
    const row = document.createElement("div")
    row.className = "recent-row"
    const status = escapeHTML(upload.status || "pending")
    row.innerHTML = `
      <span class="recent-icon ${status}" aria-hidden="true"></span>
      <div class="recent-copy">
        <strong>${escapeHTML(scanLabel(upload))}</strong>
        <span>${escapeHTML(scanMetrics(upload))}</span>
      </div>
      <time class="recent-time" datetime="${escapeHTML(activityTime(upload))}">${escapeHTML(relativeTime(activityTime(upload)))}</time>`
    elements.recentList.append(row)
  }
}

function renderFacts(container, scan) {
  container.replaceChildren()
  if (!scan) {
    container.hidden = true
    return
  }
  const facts = [
    ["Market", marketLabel(scan)],
    ["Captured", formatDateTime(scan.captured_at)],
    ["Rows", formatNumber(scan.row_count)],
    ["Items", formatNumber(scan.item_count)],
  ].filter(([, value]) => value && value !== "—")
  for (const [label, value] of facts) {
    const item = document.createElement("span")
    const small = document.createElement("small")
    small.textContent = label
    item.append(small, document.createTextNode(value))
    container.append(item)
  }
  container.hidden = facts.length === 0
}

function renderSettings(snapshot) {
  const tokenStored = Boolean(snapshot.token_stored)
  if (!tokenStored) state.removeTokenConfirmationOpen = false
  const removal = deriveTokenRemovalView(tokenStored, state.removeTokenConfirmationOpen)
  elements.settingsTokenDetail.textContent = tokenStored
    ? `${snapshot.token_prefix || "wms1_"}… stored securely`
    : "No token stored"
  setBadge(elements.settingsTokenStatus, tokenStored ? "Stored" : "Missing", tokenStored ? "success" : "warning")
  elements.removeTokenButton.hidden = !removal.showTrigger
  elements.removeTokenButton.disabled = state.busy || !removal.canRemove
  elements.removeTokenConfirmation.hidden = !removal.showConfirmation
  elements.removeTokenCancelButton.disabled = state.busy
  elements.removeTokenConfirmButton.disabled = state.busy
  elements.replaceTokenButton.textContent = tokenStored ? "Replace" : "Add token"
  elements.replaceTokenButton.setAttribute("aria-expanded", String(state.replaceTokenOpen))
  elements.settingsTokenForm.hidden = !state.replaceTokenOpen

  elements.settingsWowPath.textContent = snapshot.wow_install_path || "Not detected"
  setBadge(elements.settingsWowStatus, snapshot.wow_detected ? "Detected" : "Missing", snapshot.wow_detected ? "success" : "warning")
  elements.settingsAddonState.textContent = snapshot.addon_detected ? "Installed" : "Not found"
  elements.settingsScanState.textContent = snapshot.saved_variables_detected
    ? `${snapshot.scan_file_count || 1} SavedVariables file${(snapshot.scan_file_count || 1) === 1 ? "" : "s"}`
    : "Waiting for scan data"
  elements.settingsAccount.textContent = snapshot.selected_account || "—"
  renderAccounts(snapshot.discoveries || [], snapshot.scan_file_path)

  renderUpdater(snapshot.updater || {})
  elements.launchAtLoginSection.hidden = !snapshot.launch_at_login_supported
  elements.launchAtLoginToggle.checked = Boolean(snapshot.launch_at_login)
  elements.launchAtLoginToggle.disabled = state.busy || !snapshot.launch_at_login_supported
  elements.appVersion.textContent = snapshot.version ? `v${snapshot.version}` : "Version unavailable"
  elements.settingsDataDir.textContent = snapshot.data_dir || "—"
  elements.settingsWatcherState.textContent = snapshot.running ? "Running in the background" : snapshot.ready ? "Starting" : "Waiting for setup"
}

function renderUpdater(updater) {
  const view = deriveUpdaterView(updater)
  const showHomeNotice = Boolean(view.notify && updater.available_version)
  elements.homeUpdateNotice.hidden = !showHomeNotice
  if (showHomeNotice) {
    elements.homeUpdateTitle.textContent = `Update ${updater.available_version} is available`
    elements.homeUpdateMessage.textContent = view.message
    elements.homeUpdateButton.hidden = !view.action
    elements.homeUpdateButton.disabled = state.busy || !view.action
    elements.homeUpdateButton.textContent = view.action || "Review update"
  }
  setBadge(elements.updateStatus, view.label, view.tone)
  elements.updateMessage.textContent = view.message
  elements.updateChannel.value = updater.channel || "stable"
  elements.updateChannel.disabled = state.busy || !updater.enabled
  elements.updateCheckButton.disabled = state.busy || !view.canCheck
  elements.updateInstallButton.hidden = !view.action
  elements.updateInstallButton.disabled = state.busy || !view.action
  elements.updateInstallButton.textContent = view.action || "Review update"
  elements.updateDeferButton.hidden = !view.canDefer
  elements.updateDeferButton.disabled = state.busy || !view.canDefer
}

function renderAccounts(candidates, selectedPath) {
  const signature = accountListSignature(candidates, selectedPath)
  if (signature === state.accountsSignature) return
  state.accountsSignature = signature
  elements.accountList.replaceChildren()
  elements.accountList.hidden = candidates.length < 2
  if (candidates.length < 2) return
  for (const candidate of candidates) {
    const button = document.createElement("button")
    button.type = "button"
    button.className = `account-option${candidate.path === selectedPath ? " selected" : ""}`
    button.dataset.action = "true"
    button.disabled = state.busy
    const label = document.createElement("span")
    label.textContent = candidate.account || "Unknown account"
    const detail = document.createElement("small")
    detail.textContent = candidate.path === selectedPath ? "Selected" : formatDateTime(candidate.modified_at)
    button.append(label, detail)
    button.addEventListener("click", () => run(() => backend().SetScanFile(candidate.path)))
    elements.accountList.append(button)
  }
}

function setBadge(element, text, tone) {
  element.textContent = text
  element.className = `status-badge ${tone}`
}

function summaryMarkup(scan) {
  return `<div class="scan-summary">
    <div class="scan-summary-main">
      <strong>${escapeHTML(scanLabel(scan))}</strong>
      <time datetime="${escapeHTML(scan.uploaded_at || scan.captured_at || "")}">${escapeHTML(relativeTime(scan.uploaded_at || scan.captured_at))}</time>
    </div>
    <div class="summary-metrics">
      <span><small>Captured </small>${escapeHTML(formatDateTime(scan.captured_at))}</span>
      <span><small>Rows </small>${escapeHTML(formatNumber(scan.row_count))}</span>
      <span><small>Items </small>${escapeHTML(formatNumber(scan.item_count))}</span>
      ${scan.scan_id ? `<span><small>Scan </small>#${escapeHTML(formatNumber(scan.scan_id))}</span>` : ""}
    </div>
  </div>`
}

function scanLabel(scan) {
  const character = [scan.scanner_name, scan.scanner_realm].filter(Boolean).join(" · ")
  return character || marketLabel(scan) || "Auctionator scan"
}

function marketLabel(scan) {
  return [scan.realm, scan.market].filter(Boolean).join(" · ")
}

function scanMetrics(scan) {
  const metrics = []
  if (scan.row_count) metrics.push(`${formatNumber(scan.row_count)} rows`)
  if (scan.item_count) metrics.push(`${formatNumber(scan.item_count)} items`)
  metrics.push(uploadStatusLabel(scan))
  return metrics.join(" · ")
}

function uploadStatusLabel(scan) {
  if (scan.status === "uploaded") return scan.server_status === "duplicate" ? "Already uploaded" : "Uploaded"
  if (scan.status === "uploading") return "Uploading"
  if (scan.status === "failed") return scan.retryable ? "Retrying" : "Failed"
  return "Waiting"
}

function uploadTone(scan) {
  if (scan.status === "uploaded") return "success"
  if (scan.status === "failed") return scan.retryable ? "warning" : "danger"
  return "active"
}

function activityTime(scan) {
  return scan.uploaded_at || scan.last_attempt_at || scan.queued_at || scan.captured_at || ""
}

function formatDateTime(value) {
  if (!value) return "—"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "—"
  return date.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" })
}

function relativeTime(value) {
  if (!value) return "—"
  const timestamp = Date.parse(value)
  if (Number.isNaN(timestamp)) return "—"
  const seconds = Math.round((timestamp - Date.now()) / 1000)
  const formatter = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" })
  if (Math.abs(seconds) < 60) return formatter.format(seconds, "second")
  const minutes = Math.round(seconds / 60)
  if (Math.abs(minutes) < 60) return formatter.format(minutes, "minute")
  const hours = Math.round(minutes / 60)
  if (Math.abs(hours) < 24) return formatter.format(hours, "hour")
  return formatter.format(Math.round(hours / 24), "day")
}

function formatNumber(value) {
  const number = Number(value || 0)
  return Number.isFinite(number) ? number.toLocaleString() : "0"
}

function basename(path) {
  return String(path || "").split(/[\\/]/).filter(Boolean).pop() || path
}

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;")
}

function showError(error) {
  elements.notice.textContent = errorMessage(error)
  elements.notice.hidden = false
}

function clearError() {
  elements.notice.hidden = true
  elements.notice.textContent = ""
}

function errorMessage(error) {
  if (typeof error === "string") return error
  return error?.message || String(error)
}

function setBusy(busy) {
  state.busy = busy
  for (const element of $$('[data-action]')) element.disabled = busy
  if (state.snapshot) renderSettings(state.snapshot)
}

async function run(action) {
  if (state.busy) return
  clearError()
  setBusy(true)
  try {
    const result = await action()
    if (result && typeof result === "object" && "initializing" in result) render(result)
    else await refresh()
  } catch (error) {
    showError(error)
    if (state.snapshot) render(state.snapshot)
  } finally {
    setBusy(false)
  }
}

function openPage(page) {
  state.page = page
  clearError()
  renderPage()
}

elements.homeButton = $("#home-button")
elements.homeButton.addEventListener("click", () => openPage("overview"))
elements.settingsButton.addEventListener("click", () => openPage("settings"))
$("#back-button").addEventListener("click", () => openPage("overview"))

elements.tokenForm.addEventListener("submit", (event) => {
  event.preventDefault()
  const token = elements.tokenInput.value
  run(async () => {
    const snapshot = await backend().SetInstallationToken({ token })
    elements.tokenForm.reset()
    return snapshot
  })
})

elements.settingsTokenForm.addEventListener("submit", (event) => {
  event.preventDefault()
  const token = elements.settingsTokenInput.value
  run(async () => {
    const snapshot = await backend().SetInstallationToken({ token })
    elements.settingsTokenForm.reset()
    state.replaceTokenOpen = false
    return snapshot
  })
})

elements.replaceTokenButton.addEventListener("click", () => {
  state.replaceTokenOpen = !state.replaceTokenOpen
  if (state.snapshot) renderSettings(state.snapshot)
  if (state.replaceTokenOpen) elements.settingsTokenInput.focus()
})

elements.removeTokenButton.addEventListener("click", () => {
  state.removeTokenConfirmationOpen = true
  if (state.snapshot) renderSettings(state.snapshot)
})

elements.removeTokenCancelButton.addEventListener("click", () => {
  state.removeTokenConfirmationOpen = false
  if (state.snapshot) renderSettings(state.snapshot)
})

elements.removeTokenConfirmButton.addEventListener("click", () => {
  run(async () => {
    const snapshot = await backend().RemoveInstallationToken()
    state.removeTokenConfirmationOpen = false
    return snapshot
  })
})

for (const selector of ["#get-token-button", "#settings-open-token-button"]) {
  $(selector).addEventListener("click", () => run(() => backend().OpenInstallationsPage()))
}
for (const selector of ["#auto-detect-button", "#settings-auto-detect-button"]) {
  $(selector).addEventListener("click", () => run(() => backend().AutoDetectWowFolder()))
}
for (const selector of ["#choose-wow-button", "#settings-choose-wow-button"]) {
  $(selector).addEventListener("click", () => run(() => backend().SelectWowFolder()))
}
for (const selector of ["#addon-check-button", "#scan-check-button", "#refresh-settings-button"]) {
  $(selector).addEventListener("click", () => run(() => backend().RefreshSetup()))
}

elements.launchAtLoginToggle.addEventListener("change", () => {
  const enabled = elements.launchAtLoginToggle.checked
  run(() => backend().SetLaunchAtLogin(enabled))
})

elements.updateChannel.addEventListener("change", () => {
  const channel = elements.updateChannel.value
  run(() => backend().SetUpdateChannel(channel))
})

elements.updateCheckButton.addEventListener("click", () => {
  run(() => backend().CheckForUpdates())
})

elements.updateInstallButton.addEventListener("click", () => {
  run(() => backend().InstallUpdate())
})

elements.homeUpdateButton.addEventListener("click", () => {
  run(() => backend().InstallUpdate())
})

elements.updateDeferButton.addEventListener("click", () => {
  run(() => backend().DeferUpdate())
})

window.addEventListener("DOMContentLoaded", () => {
  if (window.runtime?.EventsOn) {
    window.runtime.EventsOn("companion:snapshot", (snapshot) => render(snapshot))
  }
  void refresh()
  window.setInterval(refresh, 2500)
})
