const state = {
  busy: false,
  snapshot: null,
  view: "overview",
}

const $ = (selector) => document.querySelector(selector)

const elements = {
  accountActions: $("#account-actions"),
  accountEmail: $("#account-email"),
  accountLoginButton: $("#account-login-button"),
  accountLoginEmail: $("#account-login-email"),
  accountLoginForm: $("#account-login-form"),
  accountLoginPassword: $("#account-login-password"),
  accountName: $("#account-name"),
  accountPageStatus: $("#account-page-status"),
  accountStatus: $("#account-status"),
  accountStepDot: $("#account-step-dot"),
  accountStepText: $("#account-step-text"),
  accountSummary: $("#account-summary"),
  archivedCount: $("#archived-count"),
  candidateCount: $("#candidate-count"),
  candidateList: $("#candidate-list"),
  discoverButton: $("#discover-button"),
  enrollButton: $("#enroll-button"),
  enrollForm: $("#enroll-form"),
  enrolledName: $("#enrolled-name"),
  failedCount: $("#failed-count"),
  flowCard: $("#flow-card"),
  flowKicker: $("#flow-kicker"),
  flowSummary: $("#flow-summary"),
  flowTitle: $("#flow-title"),
  installationActions: $("#installation-actions"),
  installationBlocked: $("#installation-blocked"),
  installationEnrollButton: $("#installation-enroll-button"),
  installationEnrollForm: $("#installation-enroll-form"),
  installationName: $("#installation-name"),
  installationPageName: $("#installation-page-name"),
  installationPageStatus: $("#installation-page-status"),
  installationStatus: $("#enrollment-status"),
  installationStepDot: $("#installation-step-dot"),
  installationStepText: $("#installation-step-text"),
  installationSummary: $("#installation-summary"),
  lastArchive: $("#last-archive"),
  lastError: $("#last-error"),
  lastEvent: $("#last-event"),
  lastUpload: $("#last-upload"),
  loginButton: $("#login-button"),
  loginEmail: $("#login-email"),
  loginForm: $("#login-form"),
  loginPassword: $("#login-password"),
  queuedCount: $("#queued-count"),
  readyFlow: $("#ready-flow"),
  readySummary: $("#ready-summary"),
  removeEnrollmentButton: $("#remove-enrollment-button"),
  runtimeDot: $("#runtime-dot"),
  runtimeLabel: $("#runtime-label"),
  scanDetail: $("#scan-detail"),
  scanDiscoverButton: $("#scan-discover-button"),
  scanFlow: $("#scan-flow"),
  scanHelp: $("#scan-help"),
  scanPageStatus: $("#scan-page-status"),
  scanSelectWowFolderButton: $("#scan-select-wow-folder-button"),
  scanStatus: $("#scan-status"),
  scanStepDot: $("#scan-step-dot"),
  scanStepText: $("#scan-step-text"),
  scanSummary: $("#scan-summary"),
  selectWowFolderButton: $("#select-wow-folder-button"),
  selectedAccount: $("#selected-account"),
  signoutButton: $("#signout-button"),
  startButton: $("#start-button"),
  statusOrb: $("#status-orb"),
  stopButton: $("#stop-button"),
  tokenPrefix: $("#token-prefix"),
  uploadedCount: $("#uploaded-count"),
  uploadsPageFailed: $("#uploads-page-failed"),
  uploadsPageQueued: $("#uploads-page-queued"),
  uploadsPageUploaded: $("#uploads-page-uploaded"),
  viewSelect: $("#view-select"),
  watcherState: $("#watcher-state"),
  wowFolder: $("#wow-folder"),
}

function backend() {
  const app = window.go?.main?.App
  if (!app) throw new Error("Wails backend is not ready")
  return app
}

async function refresh() {
  if (state.busy) return
  try {
    render(await backend().Snapshot())
  } catch (error) {
    renderError(error)
  }
}

function render(snapshot) {
  state.snapshot = snapshot
  renderPages()
  renderRuntime(snapshot)
  renderFlow(snapshot)
  renderAccount(snapshot)
  renderInstallation(snapshot)
  renderScan(snapshot)
  renderUploads(snapshot)
}

function renderPages() {
  elements.viewSelect.value = state.view
  for (const page of document.querySelectorAll("[data-page]")) {
    page.hidden = page.dataset.page !== state.view
  }
}

function renderRuntime(snapshot) {
  const hasError = Boolean(snapshot.last_error)
  const ready = Boolean(snapshot.ready)
  elements.runtimeLabel.textContent = hasError
    ? "Needs attention"
    : ready && snapshot.running
      ? "Running"
      : ready
        ? "Ready"
        : "Setup needed"
  elements.runtimeDot.classList.toggle("running", ready && !hasError)
  elements.runtimeDot.classList.toggle("error", hasError)
}

function renderFlow(snapshot) {
  setStep(elements.accountStepDot, snapshot.logged_in)
  elements.accountStepText.textContent = snapshot.logged_in
    ? userLabel(snapshot)
    : "Sign in"

  setStep(elements.installationStepDot, snapshot.enrolled)
  elements.installationStepText.textContent = snapshot.enrolled
    ? snapshot.installation_name || "Enrolled"
    : "Enroll device"

  setStep(elements.scanStepDot, Boolean(snapshot.scan_file_path))
  elements.scanStepText.textContent = snapshot.scan_file_path
    ? scanLabel(snapshot)
    : "Find folder"

  elements.flowCard.className = `hero-card step-${snapshot.current_step || "login"}`
  elements.statusOrb.className = `status-orb step-${snapshot.current_step || "login"}`

  const copy = flowCopy(snapshot)
  elements.flowKicker.textContent = copy.kicker
  elements.flowTitle.textContent = copy.title
  elements.flowSummary.textContent = copy.summary

  elements.loginForm.hidden = snapshot.current_step !== "login"
  elements.enrollForm.hidden = snapshot.current_step !== "enrollment"
  elements.scanFlow.hidden = snapshot.current_step !== "scan"
  elements.readyFlow.hidden = snapshot.current_step !== "ready"

  setInputIfUntouched(elements.loginEmail, snapshot.email || "")
  setInputIfUntouched(elements.installationName, snapshot.installation_name || "")
  elements.loginButton.disabled = state.busy
  elements.enrollButton.disabled = state.busy || !snapshot.logged_in
  elements.discoverButton.disabled = state.busy
  elements.selectWowFolderButton.disabled = state.busy

  elements.uploadedCount.textContent = String(snapshot.uploaded_count || 0)
  elements.queuedCount.textContent = String(snapshot.queued_count || 0)
  elements.failedCount.textContent = String(snapshot.failed_count || 0)
  elements.readySummary.textContent = snapshot.running
    ? "The watcher is running in the background."
    : "Everything is configured. Start the watcher from Uploads if needed."
}

function renderAccount(snapshot) {
  const loggedIn = Boolean(snapshot.logged_in)
  elements.accountStatus.textContent = loggedIn ? "Signed in" : "Not signed in"
  elements.accountPageStatus.textContent = elements.accountStatus.textContent
  elements.accountStatus.classList.toggle("ok", loggedIn)
  elements.accountPageStatus.classList.toggle("ok", loggedIn)
  elements.accountSummary.hidden = !loggedIn
  elements.accountActions.hidden = !loggedIn
  elements.accountLoginForm.hidden = loggedIn
  elements.accountName.textContent = snapshot.user_display_name || "Signed in"
  elements.accountEmail.textContent = snapshot.email || ""
  setInputIfUntouched(elements.accountLoginEmail, snapshot.email || "")
  elements.accountLoginButton.disabled = state.busy
  elements.signoutButton.disabled = state.busy || !loggedIn
}

function renderInstallation(snapshot) {
  const loggedIn = Boolean(snapshot.logged_in)
  const enrolled = Boolean(snapshot.enrolled)
  elements.installationStatus.textContent = enrolled ? "Enrolled" : "Not enrolled"
  elements.installationPageStatus.textContent = elements.installationStatus.textContent
  elements.installationStatus.classList.toggle("ok", enrolled)
  elements.installationPageStatus.classList.toggle("ok", enrolled)
  elements.installationEnrollForm.hidden = !loggedIn || enrolled
  elements.installationSummary.hidden = !enrolled
  elements.installationActions.hidden = !enrolled
  elements.installationBlocked.hidden = loggedIn || enrolled
  elements.enrolledName.textContent = snapshot.installation_name || "This device"
  elements.tokenPrefix.textContent = snapshot.token_prefix
    ? `Token ${snapshot.token_prefix}`
    : "Token stored"
  setInputIfUntouched(elements.installationPageName, snapshot.installation_name || "")
  elements.installationEnrollButton.disabled = state.busy || !loggedIn || enrolled
  elements.removeEnrollmentButton.disabled = state.busy || !enrolled
}

function renderScan(snapshot) {
  const scanConfigured = Boolean(snapshot.scan_file_path)
  elements.scanStatus.textContent = scanConfigured ? "Detected" : "Not detected"
  elements.scanPageStatus.textContent = elements.scanStatus.textContent
  elements.scanStatus.classList.toggle("ok", scanConfigured)
  elements.scanPageStatus.classList.toggle("ok", scanConfigured)
  elements.scanSummary.textContent = scanConfigured
    ? `${snapshot.scan_file_count || 1} account scan file${(snapshot.scan_file_count || 1) === 1 ? "" : "s"} found`
    : "No scan files found yet"
  elements.scanDetail.textContent = scanConfigured
    ? scanLabel(snapshot)
    : "Select the World of Warcraft installation folder to scan all Anniversary accounts."
  elements.wowFolder.textContent = snapshot.wow_install_path || "Not selected"
  elements.selectedAccount.textContent = snapshot.selected_account || "None"
  elements.scanDiscoverButton.disabled = state.busy
  elements.scanSelectWowFolderButton.disabled = state.busy
  renderCandidates(snapshot.discoveries || [], snapshot.scan_file_path)
}

function renderUploads(snapshot) {
  elements.uploadsPageUploaded.textContent = String(snapshot.uploaded_count || 0)
  elements.archivedCount.textContent = String(snapshot.archived_count || 0)
  elements.uploadsPageQueued.textContent = String(snapshot.queued_count || 0)
  elements.uploadsPageFailed.textContent = String(snapshot.failed_count || 0)
  elements.watcherState.textContent = snapshot.running ? "Running" : "Stopped"
  elements.watcherState.classList.toggle("ok", Boolean(snapshot.running))
  elements.lastUpload.textContent = formatTime(snapshot.last_upload_at)
  elements.lastArchive.textContent = formatTime(snapshot.last_archive_at)
  elements.lastEvent.textContent = eventText(snapshot)
  elements.lastError.textContent = snapshot.last_error || "None"
  elements.startButton.disabled = state.busy || snapshot.running || !snapshot.configured
  elements.stopButton.disabled = state.busy || !snapshot.running
}

function renderCandidates(candidates, selectedPath) {
  elements.candidateList.replaceChildren()
  elements.candidateCount.textContent = `${candidates.length} found`
  if (candidates.length === 0) {
    const empty = document.createElement("p")
    empty.className = "muted"
    empty.textContent = "No Anniversary account SavedVariables files detected."
    elements.candidateList.append(empty)
    return
  }
  for (const candidate of candidates) {
    const button = document.createElement("button")
    button.type = "button"
    button.className = "candidate"
    button.dataset.action = "true"
    button.disabled = state.busy
    if (candidate.path === selectedPath) button.classList.add("selected")
    const account = candidate.account || "Unknown account"
    button.innerHTML = `<strong>${escapeHTML(account)}</strong><span>${escapeHTML(candidate.path)}</span><small>${formatTime(candidate.modified_at)}</small>`
    button.addEventListener("click", () => run(() => backend().SetScanFile(candidate.path)))
    elements.candidateList.append(button)
  }
}

function flowCopy(snapshot) {
  switch (snapshot.current_step) {
    case "ready":
      return {
        kicker: "Ready",
        title: snapshot.running ? "Uploading is active" : "Companion is ready",
        summary: snapshot.running
          ? "You can close the window. The menu bar icon will stay visible while uploads continue."
          : "Everything is configured. The watcher can run from the Uploads page.",
      }
    case "enrollment":
      return {
        kicker: "Step 2 of 3",
        title: "Enroll this installation",
        summary: "Create an upload token for this device. The token will be stored in the OS credential store.",
      }
    case "scan":
      return {
        kicker: "Step 3 of 3",
        title: "Find your WoW installation",
        summary: "The companion will scan the Anniversary account folders and select the newest WowMarketScan file.",
      }
    case "login":
    default:
      return {
        kicker: "Step 1 of 3",
        title: "Sign in",
        summary: "Use your Wow Market Scan account before enrolling this device.",
      }
  }
}

function setStep(dot, complete) {
  dot.classList.toggle("complete", Boolean(complete))
}

function setInputIfUntouched(input, value) {
  if (document.activeElement === input || input.dataset.touched === "true") return
  input.value = value
}

function clearTouched(input) {
  input.dataset.touched = "false"
}

function userLabel(snapshot) {
  if (snapshot.user_display_name) return snapshot.user_display_name
  if (snapshot.email) return snapshot.email
  return "Signed in"
}

function scanLabel(snapshot) {
  if (snapshot.selected_account) return snapshot.selected_account
  if (snapshot.scan_file_path) return basename(snapshot.scan_file_path)
  return "Detected"
}

function eventText(snapshot) {
  if (!snapshot.last_message && !snapshot.last_event_at) return "None"
  if (!snapshot.last_event_at) return snapshot.last_message
  return `${snapshot.last_message} (${formatTime(snapshot.last_event_at)})`
}

function formatTime(value) {
  if (!value) return "None"
  return new Date(value).toLocaleString()
}

function basename(path) {
  return String(path || "").split(/[\\/]/).filter(Boolean).pop() || path
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
}

function renderError(error) {
  elements.runtimeLabel.textContent = "Needs attention"
  elements.runtimeDot.classList.add("error")
  elements.flowKicker.textContent = "Needs attention"
  elements.flowTitle.textContent = "Action failed"
  elements.flowSummary.textContent = error?.message || String(error)
}

async function run(action) {
	state.busy = true
	setBusy(true)
	let succeeded = false
	let failure = null
	try {
		const snapshot = await action()
		render(snapshot)
		succeeded = true
	} catch (error) {
		failure = error
		renderError(error)
	} finally {
		state.busy = false
		if (state.snapshot) {
			render(state.snapshot)
		}
		if (!succeeded && failure) {
			renderError(failure)
		}
		setBusy(false)
	}
	return succeeded
}

function setBusy(busy) {
	document.body.classList.toggle("busy", busy)
	if (!busy) return
	for (const button of document.querySelectorAll("[data-action]")) {
		button.disabled = true
	}
}

function submitLogin(emailInput, passwordInput) {
  return run(() =>
    backend().Login({
      email: emailInput.value,
      password: passwordInput.value,
    }),
  ).then((succeeded) => {
    if (!succeeded) return
    passwordInput.value = ""
    clearTouched(passwordInput)
    clearTouched(emailInput)
  })
}

function submitEnrollment(nameInput) {
  return run(() =>
    backend().Enroll({
      installation_name: nameInput.value,
    }),
  ).then((succeeded) => {
    if (!succeeded) return
    clearTouched(nameInput)
  })
}

for (const input of document.querySelectorAll("input")) {
  input.dataset.touched = "false"
  input.addEventListener("input", () => {
    input.dataset.touched = "true"
  })
}

elements.viewSelect.addEventListener("change", () => {
  state.view = elements.viewSelect.value
  renderPages()
})

for (const button of document.querySelectorAll("[data-view-jump]")) {
  button.addEventListener("click", () => {
    state.view = button.dataset.viewJump
    renderPages()
  })
}

elements.loginForm.addEventListener("submit", (event) => {
  event.preventDefault()
  submitLogin(elements.loginEmail, elements.loginPassword)
})

elements.accountLoginForm.addEventListener("submit", (event) => {
  event.preventDefault()
  submitLogin(elements.accountLoginEmail, elements.accountLoginPassword)
})

elements.enrollForm.addEventListener("submit", (event) => {
  event.preventDefault()
  submitEnrollment(elements.installationName)
})

elements.installationEnrollForm.addEventListener("submit", (event) => {
  event.preventDefault()
  submitEnrollment(elements.installationPageName)
})

elements.signoutButton.addEventListener("click", () => run(() => backend().SignOut()))
elements.removeEnrollmentButton.addEventListener("click", () =>
  run(() => backend().RemoveEnrollment()),
)
elements.discoverButton.addEventListener("click", () =>
  run(() => backend().DiscoverScanFiles()),
)
elements.scanDiscoverButton.addEventListener("click", () =>
  run(() => backend().DiscoverScanFiles()),
)
elements.selectWowFolderButton.addEventListener("click", () =>
  run(() => backend().SelectWowFolder()),
)
elements.scanSelectWowFolderButton.addEventListener("click", () =>
  run(() => backend().SelectWowFolder()),
)
elements.startButton.addEventListener("click", () =>
  run(() => backend().StartWatcher()),
)
elements.stopButton.addEventListener("click", () =>
  run(() => backend().StopWatcher()),
)

window.addEventListener("DOMContentLoaded", () => {
  refresh()
  window.setInterval(refresh, 2000)
})
