const recentActivityWindow = 30_000

export function accountListSignature(candidates = [], selectedPath = "") {
  return JSON.stringify([
    selectedPath,
    candidates.map((candidate) => [
      candidate.path,
      candidate.account,
      candidate.modified_at,
    ]),
  ])
}

export function heroAnnouncementSignature(view) {
  const scan = view.scan
  return JSON.stringify([
    view.mode,
    view.tone,
    view.eyebrow,
    view.title,
    view.summary,
    scan
      ? [
          scan.status,
          scan.realm,
          scan.market,
          scan.captured_at,
          scan.row_count,
          scan.item_count,
          scan.uploaded_at,
          scan.next_attempt_at,
          scan.error,
        ]
      : null,
  ])
}

export function deriveUpdaterView(updater = {}) {
  const version = updater.available_version ? `Version ${updater.available_version}` : ""
  switch (updater.status) {
    case "checking":
      return { label: "Checking", tone: "active", message: "Checking the signed update feed.", canCheck: false }
    case "downloading":
      return { label: "Downloading", tone: "active", message: version ? `${version} is downloading securely in the background.` : "Downloading securely in the background.", canCheck: false }
    case "available":
      return { label: updater.mandatory ? "Required" : "Available", tone: updater.mandatory ? "warning" : "active", message: `${version} is available.`, action: "Review update", canDefer: !updater.mandatory, canCheck: true }
    case "ready":
      return { label: updater.mandatory ? "Required" : "Ready", tone: updater.mandatory ? "warning" : "success", message: `${version} is verified and ready to install.`, action: "Install and restart", canDefer: !updater.mandatory, canCheck: true }
    case "deferred":
      return { label: "Later", tone: "warning", message: `${version} is deferred. Scans and uploads continue normally.`, action: updater.ready_to_install ? "Install and restart" : "Review update", canCheck: true }
    case "offline":
      return { label: "Offline", tone: "warning", message: "Updates could not be checked. Scans and uploads continue normally.", canCheck: true }
    case "error":
      return { label: "Check failed", tone: "danger", message: updater.message || "Update verification failed.", canCheck: true }
    case "disabled":
      return { label: "Development", tone: "", message: updater.message || "Updates are enabled in official builds.", canCheck: false }
    case "current":
    default:
      return { label: "Current", tone: "success", message: updater.message || `Version ${updater.current_version || "—"} is up to date.`, canCheck: Boolean(updater.enabled) }
  }
}

export function deriveView(snapshot, now = Date.now()) {
  if (!snapshot || snapshot.initializing) {
    return {
      mode: "starting",
      tone: "active",
      eyebrow: "Starting",
      title: "Checking your setup",
      summary: snapshot?.startup_phase || "Preparing WoW Markets Companion.",
    }
  }

  if (snapshot.current_step !== "ready") {
    const setup = setupView(snapshot.current_step)
    if (snapshot.last_error) {
      return {
        ...setup,
        tone: "danger",
        eyebrow: "Setup needs attention",
        summary: snapshot.last_error,
      }
    }
    return setup
  }

  const current = snapshot.current_upload
  if (current) {
    return {
      mode: "uploading",
      tone: "active",
      eyebrow: "Uploading",
      title: "Uploading your latest scan",
      summary: "Your Auctionator scan was detected and is being sent securely.",
      scan: current,
    }
  }

  const failure = snapshot.upload_failure
  if (failure) {
    if (failure.retryable) {
      return {
        mode: "retrying",
        tone: "warning",
        eyebrow: "Retry scheduled",
        title: "Upload interrupted",
        summary: "The companion will try again automatically.",
        scan: failure,
      }
    }
    return {
      mode: "failed",
      tone: "danger",
      eyebrow: "Needs attention",
      title: "Upload needs attention",
      summary:
        failure.http_status === 401 || failure.http_status === 403
          ? "This token is invalid or was revoked. Replace it in Settings."
          : failure.error || "The latest scan could not be uploaded.",
      scan: failure,
    }
  }

  if (snapshot.last_error) {
    return {
      mode: "failed",
      tone: "danger",
      eyebrow: "Needs attention",
      title: "Companion needs attention",
      summary: snapshot.last_error,
    }
  }

  const lastUpload = snapshot.last_upload
  if (
    lastUpload?.uploaded_at &&
    now - Date.parse(lastUpload.uploaded_at) >= 0 &&
    now - Date.parse(lastUpload.uploaded_at) <= recentActivityWindow
  ) {
    return {
      mode: "uploaded",
      tone: "success",
      eyebrow: "Complete",
      title: "Scan uploaded",
      summary: "Your market data is live. Watching for the next scan.",
      scan: lastUpload,
    }
  }

  if (
    ["archive", "queue"].includes(snapshot.activity_kind) &&
    isRecent(snapshot.last_event_at, now)
  ) {
    return {
      mode: "detected",
      tone: "active",
      eyebrow: "New scan detected",
      title: "Preparing your latest scan",
      summary: "The companion is validating and queueing it for upload.",
      scan: snapshot.last_detected,
    }
  }

  return {
    mode: "waiting",
    tone: "success",
    eyebrow: snapshot.running ? "Active" : "Ready",
    title: snapshot.running ? "Watching for scans" : "Ready to watch",
    summary: snapshot.running
      ? "Run a full Auctionator scan whenever you like. You can close this window."
      : "Your setup is complete. The watcher is starting.",
    scan: snapshot.last_upload || snapshot.last_detected,
  }
}

function setupView(step) {
  switch (step) {
    case "wow":
      return {
        mode: "setup",
        setupStep: "wow",
        tone: "active",
        eyebrow: "Setup · Step 2 of 4",
        title: "Find World of Warcraft",
        summary: "Choose your WoW folder so the companion can find Anniversary data.",
      }
    case "addon":
      return {
        mode: "setup",
        setupStep: "addon",
        tone: "warning",
        eyebrow: "Setup · Step 3 of 4",
        title: "WoW Markets addon not found",
        summary: "Install the addon for Anniversary, then check again.",
      }
    case "saved_variables":
      return {
        mode: "setup",
        setupStep: "saved_variables",
        tone: "active",
        eyebrow: "Setup · Step 4 of 4",
        title: "Waiting for scan data",
        summary: "Run an Auctionator full scan, then type /reload or log out.",
      }
    case "token":
    default:
      return {
        mode: "setup",
        setupStep: "token",
        tone: "active",
        eyebrow: "Setup · Step 1 of 4",
        title: "Connect WoW Markets",
        summary: "Create an installation token on the website, then paste it here.",
      }
  }
}

function isRecent(value, now) {
  if (!value) return false
  const elapsed = now - Date.parse(value)
  return elapsed >= 0 && elapsed <= recentActivityWindow
}
