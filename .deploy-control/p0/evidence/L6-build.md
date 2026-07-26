# L6 — authoritative build (STANDBY / readiness)

- agent: Codex56#B · pane: w7:p4 · lane: L6 · task: L6-BUILD
- status: **STANDING BY** — the authoritative artifact build is deferred until **L4 wiring lands**. No build run yet (per instruction).
- repo HEAD: `a6d5098`

## Readiness preflight (verified; build NOT executed)

- go: `go version go1.26.1 linux/amd64` at `/home/ec2-user/goroot/go/bin/go`
- caches: will use lane-specific `/tmp` (`GOCACHE`/`GOTMPDIR`/`GOMODCACHE` under /tmp) — root disk is tight
- disk: root 4.3G (83% used); /tmp 3.1G (61% used) — sufficient headroom for a `go build ./...`
- working tree: intentionally dirty (`git status` = 464 entries); unrelated work preserved

## Precondition (BLOCKER)

- **Blocker:** authoritative build must not run until the L4 wiring is landed/integrated on the target tree
  (orq2). It has not been signaled as landed to this lane.
- **Owner:** manager (`w5:p1`) / L4 wiring owner.
- **Next action (on "L4 wiring landed" signal):** run the authoritative build with `/tmp` caches:
  `GOCACHE=/tmp/l6-gocache GOTMPDIR=/tmp/l6-gotmp GOMODCACHE=/tmp/l6-gomodcache /home/ec2-user/goroot/go/bin/go build ./...`
  from `multica-auth-work/server`, capture exact command + exit code + any failures by package, then record
  the artifact/build provenance here and check out with evidence.

## Non-claims

- No authoritative build executed; no artifact produced yet. No product code modified. No deploy/inference/
  secret/commit. `live_runs=false` respected.
