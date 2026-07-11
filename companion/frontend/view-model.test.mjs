import test from "node:test"
import assert from "node:assert/strict"

import {
  accountListSignature,
  deriveView,
  deriveUpdaterView,
  heroAnnouncementSignature,
} from "./dist/view-model.mjs"

const now = Date.parse("2026-07-10T12:00:00Z")

function ready(overrides = {}) {
  return {
    initializing: false,
    current_step: "ready",
    running: true,
    recent_uploads: [],
    ...overrides,
  }
}

test("keeps actions hidden while startup checks run", () => {
  assert.equal(deriveView({ initializing: true, startup_phase: "Loading token" }, now).mode, "starting")
})

test("focuses setup on the first unresolved prerequisite", () => {
  for (const step of ["token", "wow", "addon", "saved_variables"]) {
    const view = deriveView({ initializing: false, current_step: step }, now)
    assert.equal(view.mode, "setup")
    assert.equal(view.setupStep, step)
  }
})

test("shows an in-flight upload before stale errors or successes", () => {
  const view = deriveView(
    ready({
      current_upload: { status: "uploading", row_count: 120 },
      last_error: "old error",
      last_upload: { uploaded_at: "2026-07-10T11:59:55Z" },
    }),
    now,
  )
  assert.equal(view.mode, "uploading")
})

test("distinguishes retryable and revoked-token failures", () => {
  const retry = deriveView(
    ready({
      upload_failure: { status: "failed", retryable: true, next_attempt_at: "2026-07-10T12:00:05Z" },
    }),
    now,
  )
  assert.equal(retry.mode, "retrying")

  const revoked = deriveView(
    ready({ upload_failure: { status: "failed", http_status: 401 } }),
    now,
  )
  assert.equal(revoked.mode, "failed")
  assert.match(revoked.summary, /token/i)

})

test("keeps a successful upload prominent briefly, then settles", () => {
  const upload = { status: "uploaded", uploaded_at: "2026-07-10T11:59:50Z" }
  assert.equal(deriveView(ready({ last_upload: upload }), now).mode, "uploaded")
  assert.equal(
    deriveView(ready({ last_upload: upload }), now + 31_000).mode,
    "waiting",
  )
})

test("shows detection briefly and otherwise settles into watching", () => {
  const detected = deriveView(
    ready({
      activity_kind: "archive",
      last_event_at: "2026-07-10T11:59:55Z",
      last_detected: { status: "detected", row_count: 200 },
    }),
    now,
  )
  assert.equal(detected.mode, "detected")

  const waiting = deriveView(ready(), now)
  assert.equal(waiting.mode, "waiting")
})

test("keeps setup actions visible while surfacing operational errors", () => {
  const view = deriveView(
    { initializing: false, current_step: "token", last_error: "Keychain is unavailable" },
    now,
  )
  assert.equal(view.mode, "setup")
  assert.equal(view.setupStep, "token")
  assert.equal(view.tone, "danger")
  assert.match(view.summary, /Keychain/)
})

test("keeps render signatures stable for unchanged polled snapshots", () => {
  const candidates = [{ account: "ACCOUNT", modified_at: "2026-07-10T12:00:00Z", path: "/scan.lua" }]
  assert.equal(
    accountListSignature(candidates, "/scan.lua"),
    accountListSignature(structuredClone(candidates), "/scan.lua"),
  )

  const view = deriveView(ready(), now)
  assert.equal(
    heroAnnouncementSignature(view),
    heroAnnouncementSignature(structuredClone(view)),
  )
})

test("keeps updater states calm and explicit", () => {
  assert.deepEqual(
    deriveUpdaterView({ status: "offline", enabled: true }).label,
    "Offline",
  )
  const ready = deriveUpdaterView({ status: "ready", available_version: "1.1.0", mandatory: false })
  assert.equal(ready.action, "Install and restart")
  assert.equal(ready.canDefer, true)

  const required = deriveUpdaterView({ status: "available", available_version: "1.2.0", mandatory: true })
  assert.equal(required.label, "Required")
  assert.equal(required.canDefer, false)
})

test("shows an unpromoted channel as current rather than a verification failure", () => {
  const updater = deriveUpdaterView({
    enabled: true,
    status: "current",
    current_version: "1.0.0-rc.1",
    message: "No release has been promoted to the beta channel yet",
  })
  assert.equal(updater.label, "Current")
  assert.match(updater.message, /beta channel/)
  assert.equal(updater.tone, "success")
})
