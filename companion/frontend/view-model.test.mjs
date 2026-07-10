import test from "node:test"
import assert from "node:assert/strict"

import {
  accountListSignature,
  deriveView,
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
