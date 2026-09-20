> **RESOLVED-BY-ROUTING — docs-health 9th pass (2026-09-20):** research complete — the §6 verdict is final
> for v4.x (not scheduled; revisit conditions in §6's last paragraph). No open tasks.

# scheduling/ on systemd-timers — feasibility analysis (2026-09-18)

**Status:** RESEARCH — exploratory, no decision taken. Motivated by the v5
scheduling/ re-imagination discussion (2026-09-18 session).

**Question:** Could `scheduling/` use systemd timers as a backend on
supported platforms? PRO/CONTRA.

---

## 1. Context (why this question came up)

The v5 assessment of `scheduling/` concluded: **converge, don't re-imagine.**

- Storage semantics should converge onto `queue/` + `claiming/` (a timer is
  "enqueue + `NotBefore` + dedup key + fire-once"); one claim/retry/DLQ stack,
  not two.
- The `Scheduler` shrinks to poll + dispatch-once; retry/rejection/DLQ belongs
  to `commandlifecycle` (ADR-0117 / T17 partitioning).
- `scheduling/` has **zero metaengine integration today** — deliberately:
  metaengine has no claim/lease primitive, and `claiming/` exists precisely
  because multi-key conditional claims (`FOR UPDATE SKIP LOCKED`-style) don't
  fit the engine contract.
- Today's only production `TimerStore` is `scheduling/sqlstore`
  (SQLite/Postgres/MySQL + lease-based claiming). Durability currently
  **requires a database**.

That last point is the gap a systemd backend would fill: durable timers on
hosts that run no SQL at all.

## 2. How systemd timers work (primer)

systemd timers are declarative unit files (`.timer`) that tell PID 1 — the
scheduler itself, no cron daemon — when to activate a matching unit (usually a
`.service`).

1. **Declaration:** `OnCalendar=` (cron-like calendar expressions:
   `Mon..Fri 09:00`, `*-12-25 00:00`) or monotonic forms (`OnBootSec=`,
   `OnUnitActiveSec=`).
2. **No polling:** systemd computes next-elapse for all timers and arms kernel
   **timerfds** (`timerfd_create`); the kernel wakes PID 1 exactly when
   something is due. Visible via `systemctl list-timers`.
3. **Coalescing + jitter:** `AccuracySec=` (default 1min) lets nearby timers
   fire together (power saving); `RandomizedDelaySec=` spreads load.
4. **Durability is opt-in:** `Persistent=true` writes a last-trigger **stamp
   file** to `/var/lib/systemd/timers/` on each fire; on boot, a trigger missed
   while down fires once immediately (catch-up). Without it, downtime skips
   occurrences.
5. **No payload, no retry:** a timer's only job is activating a unit. Failure
   handling lives in the _service_ (`Restart=`, `OnFailure=`) — the timer never
   retries.
6. **Single-scheduler assumption:** PID 1 is the only claimer on the host;
   atomic claiming is unnecessary by construction.
7. **The interesting inversion:** systemd persists only the **last-fire
   stamp** — the schedule itself is config, not rows. Dynamic API-driven
   timers (our case) need the full-row store; static recurring deadlines
   would fit a declarative schedule + tiny stamp with zero per-fire row churn
   (a candidate v5 shape alongside fire-once timers).

### Mapping to go-cqrs-lite `scheduling/`

| systemd                                   | go-cqrs-lite `scheduling/`                                |
| ----------------------------------------- | --------------------------------------------------------- |
| PID 1 = sole scheduler                    | base `Scheduler` (or `ClaimingTimerStore` multi-inst.)    |
| stamp file + boot catch-up                | durable `TimerStore`; restart replays `Due(now)`          |
| `AccuracySec` / jitter                    | `WithPollInterval` + jitter constant                      |
| timer fires once; `Restart=` owns failure | Scheduler dispatches once; retry/DLQ → `commandlifecycle` |

## 3. Feasibility mechanics

Mechanically yes, on Linux with systemd (host or user session):

- `Schedule` → `systemd-run --unit=cqrs-<timerID> --on-active=<duration>` or
  D-Bus `StartTransientUnit`; unit name = `TimerID` gives accidental
  idempotency (name collisions fail the second `Schedule`).
- The unit execs a helper that **re-enters the application process**
  (HTTP/Unix socket) and dispatches the command.
- `Cancel` / `MarkFired` → `StopUnit` / let the transient unit elapse.

But: systemd is a **trigger, not a store** — it fires, it does not expose a
queryable due-queue. That distinction drives the entire assessment.

## 4. PRO

| Win                                   | Why it matters                                                                                                                                         |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Durable timers with **zero database** | PID 1 owns the schedule; survives app crashes by construction. The only durable backend on a host with no SQL — edge/laptop/NixOS-service deployments. |
| No poll loop                          | timerfd-armed by the kernel; kills the 1s `Due()` scan goroutine.                                                                                      |
| `WakeSystem=true`                     | fires from suspend — genuinely unique; no SQL store can do this.                                                                                       |
| `Persistent=true` stamp               | missed-fire catch-up on reboot for free.                                                                                                               |
| Calendar recurrence                   | `OnCalendar=` gives recurring schedules the fire-once `Timer` model lacks.                                                                             |
| Operator visibility                   | `systemctl list-timers` — fits the "operators reconcile at deployment" paradigm.                                                                       |
| Per-host dedup lock                   | unit namespace collisions make double-`Schedule` structurally impossible on one host.                                                                  |

## 5. CONTRA

| Problem                                                                                                                                                                                                           | Severity             |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------- |
| `Due(now)` **cannot be implemented** — no due-queue; a shadow table defeats the purpose                                                                                                                           | 🔴 contract-breaking |
| Dispatch becomes **IPC re-entry** — the `DispatchFunc` closure is gone; needs helper binary + socket + auth; failure happens outside the library where `commandlifecycle` retry/rejection semantics cannot see it | 🔴                   |
| Transient units live in `/run` (**lost on reboot**); durable units mean writing `/etc/systemd/system` → **root or polkit**; app-scheduled timers become root-scheduled exec config = attack surface               | 🔴                   |
| Payload `P` must ride env vars/credentials/paths — size + secrecy limits                                                                                                                                          | 🟠                   |
| Platform matrix: no systemd in Docker/K8s pods, nothing on macOS/Windows — fragments the "every backend everywhere" universality story                                                                            | 🟠                   |
| Multi-host: PID 1 is per-host — no cross-instance claiming; the problem `claiming/` solved reappears at fleet scale                                                                                               | 🟠                   |
| `godbus` dependency + build tags + a booted-systemd VM/nspawn leg for CI (repo has the infra, but it is weight)                                                                                                   | 🟡                   |
| ADR-0136: schedule state lives outside the engine world — Reset+replay means mutating unit files, awkward                                                                                                         | 🟡                   |

## 6. Verdict

**As a `TimerStore` implementation: no.** The `Due()`-pull contract is a lie
for a push-based trigger, and the IPC re-entry path moves failure handling
outside the library.

**As a push-style sibling: a genuinely good niche backend.** Recommended
shape, if ever built:

- New contract, not a `TimerStore` fit:
  `type TimerTrigger interface { Firings() <-chan Fired[P] }` (illustrative).
- Dep-isolated module `scheduling/systemd` (own `go.mod`, `//go:build linux`,
  runtime-probed via `sd_booted()` — never auto-selected).
- Positioning: **edge-deployment play** (monitor365-style collectors, NixOS
  services, laptops) where no database runs — orthogonal to the server-side
  SQL+claiming path, not a replacement.
- Natural companion: if v5 grows the declarative "static recurring schedule +
  last-fire stamp" shape (§2.7), systemd is its natural executor on hosts
  that have it.

Not scheduled. Revisit if (a) an edge/no-database consumer materializes, or
(b) the declarative recurring-schedule shape lands in v5 planning.
