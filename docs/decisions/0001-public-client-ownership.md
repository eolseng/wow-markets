# ADR 0001: Public client ownership

- Status: Accepted
- Date: 2026-07-10

## Context

The addon, companion, API, and web app originally shared the private
`eolseng/wow-markets` monorepo. That arrangement simplified early coordinated
changes but could not provide inspectable client source, public contribution,
or durable client releases without exposing service implementation and
operations.

The original repository was renamed `eolseng/wow-markets-service`. A filtered
history of `addon/` and `companion/` was published under the original
`eolseng/wow-markets` name so the established Go module path remains valid.

## Decision

This public repository is the source of truth for addon and companion source,
client-produced bytes, contract specifications and fixtures, client CI,
packaging, and releases. The private service repository owns the API and web
implementation, database, infrastructure, deployments, and operational docs.

The service consumes public contracts at a recorded public commit. Public CI
must never require a private checkout. Cross-repository changes deploy backward
compatible service support first, followed by companion and addon releases.

## Consequences

- Client contributors can clone, test, and build without private material.
- Contract changes require coordinated commits and pinned consumer tests.
- Repository-local changes can no longer make all components atomic, so the
  staged compatibility policy is mandatory.
- Historical monorepo decisions remain useful context but no longer describe
  current source ownership.
