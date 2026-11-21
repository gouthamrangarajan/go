# Cull

A scheduled Go agent that cleans a Yahoo inbox: classify, unsubscribe, and move junk — with state that survives redeploys.

Built as a learning project. Read-only until you explicitly enable `-apply`.

## What it does

1. Connects to Yahoo IMAP over TLS using an app password.
2. Fetches **headers only** (`PEEK`) — nothing is marked as read.
3. Classifies each message:
   - **Tier 1 (rules, free):** keyword match + presence of `List-Unsubscribe`.
   - **Tier 2 (AI, ~$0.00002/msg):** sender + subject + has-unsubscribe flag → one of `keep`, `unsubscribe`, `junk`, `phishing`.
4. Plans an action, choosing the safest available door:
   - `unsubscribe-oneclick` — RFC 8058 POST, only when the sender declares it.
   - `move-to-junk` — `MOVE` into Yahoo's `Bulk` folder.
   - `skip` — anything without a safe path.
5. Executes only when `-apply` is set, capped by `-max-actions`.
6. Persists UIDs + sender outcomes to `state.json` so runs never repeat work.

## Classification policy

```
rules verdict  ──┐
                 ├─→ AI (if enabled) ─→ confidence ≥ 0.75 ? ─→ final
                 │                          │
                 │                          └─ no → fall back to rules
                 └─ phishing (rules) is a hard floor; AI cannot downgrade it to keep
```

If the AI call errors or returns low confidence, the rules verdict is used. There is no path where a classifier failure crashes the run or skips a message silently.

## Measured

| Metric | Value |
|---|---|
| Messages processed | 500 |
| Runtime (sequential AI calls) | 3 m 42 s (~444 ms/msg) |
| AI cost | ~$0.01 |
| False positives (audited in `Bulk`) | 0 |

Concurrency is **not** implemented yet — see [Not yet implemented](#not-yet-implemented).

## Project layout

```
cull/
├── cmd/cull/main.go        # wiring, flags, run loop
├── internal/
│   ├── config/config.go    # env loading + validation
│   ├── mail/
│   │   ├── client.go       # Open / RecentHeaders / MoveToJunk / Close
│   │   └── unsub.go        # parse List-Unsubscribe into mailto + URLs
│   ├── classify/
│   │   ├── classify.go     # rules tier
│   │   └── ai.go           # AI tier (decision API client)
│   ├── act/act.go          # Plan() + OneClick()
│   └── state/state.go      # Load / Save (atomic)
├── go.mod
├── .env.example
├── .gitignore
└── README.md
```

## Configuration

All config comes from environment variables (`.env` locally, Railway Variables in prod).

| Variable | Required | Description |
|---|---|---|
| `IMAP_HOST` | yes | `imap.mail.yahoo.com` |
| `IMAP_PORT` | yes | `993` |
| `IMAP_USER` | yes | Full Yahoo address |
| `IMAP_PASS` | yes | Yahoo **app password**, not account password |
| `CULL_STATE_PATH` | prod | `/data/state.json` on Railway (must be on a volume) |
| `CULL_APPLY` | prod | `true` to execute actions; anything else = dry run |
| `CULL_AI` | no | `true` enables AI classification |
| `CULL_AI_URL` | if AI | Decision API endpoint |
| `CULL_AI_KEY` | if AI | API key |
| `CULL_AI_MODEL` | if AI | Model ID |

`.env` is git-ignored. Use `.env.example` with placeholder values as the template.

## Usage

Dry run (default — prints planned actions, executes nothing):

```bash
go run ./cmd/cull -limit 50
```

Execute, capped:

```bash
go run ./cmd/cull -limit 50 -apply -max-actions 3
```

| Flag | Default | Meaning |
|---|---|---|
| `-limit` | 50 | Max messages to consider per run |
| `-apply` | `false` | Execute actions; otherwise dry run |
| `-max-actions` | 5 | Hard cap on executed actions per run |

Logs are JSON on stdout via `slog`.

## State

`state.json`:

```json
{
  "uidvalidity": 1,
  "last_uid": 362747,
  "sent": { "362741": true, "362743": true },
  "junked": { "362745": true }
}
```

| Field | Purpose |
|---|---|
| `uidvalidity` | Mailbox UID epoch. On mismatch, state is wiped and the agent re-bootstraps. |
| `last_uid` | Highest UID considered by a successful run. |
| `sent` | UIDs with a **successful** one-click unsubscribe. Failed attempts are retried. |
| `junked` | UIDs moved to `Bulk`. |

Saves are atomic (temp file + rename) so a crash mid-write can't corrupt state.

## Deployment (Railway)

1. Create a service, deploy the binary via Dockerfile.
2. **Mount a volume at `/data`** and set `CULL_STATE_PATH=/data/state.json`.
   Without the volume, state silently vanishes every run — the loud `read-only file system` / `no such file` errors are the *correct* behavior and should not be papered over.
3. Set all env vars above as Railway Variables.
4. Configure a cron schedule. The job must **exit** when done — build a binary, don't `go run`.
5. Keep `CULL_APPLY` unset for dry-run deploys; enable it deliberately.

Cron runs in UTC. Railway log retention is short — do not treat console logs as a durable audit trail once `-apply` is on a schedule.

## Safety model

- `PEEK` on every fetch: mail is never marked read as a side effect of inspection.
- **One-click only.** RFC 8058 requires the sender to declare `List-Unsubscribe-Post: List-Unsubscribe=One-Click`. Plain `<https://…>` links (trackers, login walls) and `mailto:` addresses (spoofable, confirm the address is live) are never auto-executed.
- URL validation before POST: `https` only, no raw IPs, no redirect following, no cookies, 15 s timeout.
- UID-based actions only. Sequence numbers shift when mail arrives mid-session; acting on them can hit the wrong message.
- Phishing from rules is a hard floor and cannot be downgraded to `keep`.
- `-max-actions` caps destructive work per run.

## Not yet implemented

- **Concurrency / parallelism.** Everything is sequential — IMAP fetch, AI classification (the dominant cost), and actions. Planned: bounded `errgroup` worker pool for AI calls only.
- **Sender-level memory.** Decisions are per-UID. Repeat senders cost a full AI call every time.
- **Unsubscribe verification.** No 14-day loop to detect senders that ignore one-click and auto-promote them to `move-to-junk`.
- **`mailto:` unsubscribes.** Deliberately manual.
- **A UI.** None. CLI + JSON logs only; a Datastar UI would come after the action layer stabilizes.

## Known gotchas

- **`Bulk` vs `Junk`** — on this account they are distinct mailboxes (`Bulk` holds real spam, `Junk` is an empty leftover). `Select` both and compare `UIDValidity` before assuming they alias.
- **`N:*` always returns at least one message** even when `N` is above the highest UID. `LastUID+1:*` needs the `uid <= LastUID` filter or every run reprocesses one message.
- **Advancing `LastUID` is conditional.** Don't advance on dry runs, and don't advance when `-max-actions` was hit — otherwise capped messages are never reconsidered.
- **Marking a mis-junked mail "not spam" does not undo its unsubscribe.** Re-subscribing is a separate act.
- **Inbox reported `10000` messages** (suspiciously round) — possibly a server cap. Relevant only if old mail is ever expected in the delta fetch.
- **AI overrides are logged**, not silent: watch `source: "ai"` plus `ai-low-conf` lines when tuning.

## Roadmap

1. Parallelize AI classification (`errgroup`, `SetLimit(10)`) → ~25 s for 500 messages.
2. Sender-level policy memory to skip AI on known senders.
3. Verification loop: did unsubscribed senders reappear after 14 days?
4. Adaptive policy per sender (auto-promote repeat offenders to `move-to-junk`).
5. Optional Datastar review UI for approvals.
