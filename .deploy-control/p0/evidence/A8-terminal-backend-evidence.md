# A8-EVIDENCE — Terminal Backend Persistence Evidence

agent: Agy-P0-A8
lane: A8-EVIDENCE
task: P0-TERMINAL-BACKEND-EVIDENCE
pane: wB:p2
check-in: 2026-07-21T22:56:53Z
status: **BLOCKED**

## 1. Tool Preflight (Corrected)

| Tool | Path / Command | Observed Version | Status |
|---|---|---|---|
| git | `git` (on PATH) | 2.50.1 | ✅ PASS |
| python3 | `python3` (on PATH) | 3.9.25 | ✅ PASS |
| rg (ripgrep) | `rg` (on PATH) | 15.2.0 | ✅ PASS |
| Go | `/home/ec2-user/goroot/go/bin/go` | go1.26.1 linux/amd64 | ✅ PASS |
| gofmt | `/home/ec2-user/goroot/go/bin/gofmt` | (bundled with go1.26.1) | ✅ PASS |
| OpenSpec | `openspec` (on PATH) | 1.4.1 | ✅ PASS |
| Herdr pane | wB:p2, label `Agy-P0-A8` | agent_status: blocked | ✅ PASS |

## 2. Dependency & Blocker

**Concrete Dependency:** 
This task owns *only* the validation of backend terminal-result persistence and delivery evidence from already-reserved integrated route runs. A8-EVIDENCE must not initiate or duplicate a live run, perform source edits, conduct broad QA, or execute synthetically. 

**Current Blocker:**
Integrated build and authorized GLM/Opus48 run evidence do not exist yet. 

**Owner:**
W1, GLM/Opus48 live-run owners, and Principal Orchestrator.

**Next Action:**
When an authorized integrated run occurs, capture the backend persisted terminal result from that same run.
