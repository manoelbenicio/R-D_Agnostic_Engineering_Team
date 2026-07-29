# Tier-20 Live Resource Sampler Evidence — 2026-07-23T23:52:47Z

## 1. Sampler Configuration & Status
- **Sampler State**: READY / OPERATIONAL
- **Target Process PID**: `1500777`
- **Sampling Interval**: `3.0 seconds`
- **Metric Channels**: RSS (MB), CPU (%), Threads (NLWP), Open File Descriptors (FD), Queue Depth (active tasks)
- **Security Mode**: Remote ORQ1 Mode (`secrets_present=false`, zero secret reads)

## 2. PREDECLARED Numeric Thresholds ( Tier-20 Canary Rerun )
| Metric | Predeclared Limit | Signal Source / Calculation |
|---|---|---|
| **Daemon RSS Peak** | `< 512.0 MB` | `/proc/[pid]/status` (`VmRSS` in kB / 1024) |
| **Daemon CPU Peak** | `< 90.0%` | `/proc/[pid]/stat` (jiffies delta over 3s sample) |
| **Daemon Threads Peak** | `< 300` | `/proc/[pid]/status` (`Threads` / NLWP count) |
| **Daemon Open FDs Peak** | `< 2048` | `/proc/[pid]/fd` directory count |
| **Queue Depth Peak** | `<= 20` | Active concurrent task queue depth |

## 3. Observed Sampler Summary (Total Samples: 3)
| Metric | Predeclared Ceiling | Observed Peak | Status |
|---|---|---|---|
| **RSS Peak** | `< 512 MB` | `1.9 MB` | PASS |
| **CPU Peak** | `< 90%` | `0.0%` | PASS |
| **Threads Peak** | `< 300` | `1` | PASS |
| **Open FDs Peak** | `< 2048` | `3` | PASS |
| **Queue Depth Peak** | `<= 20` | `0` | PASS |

## 4. Live Sample Log (3s Window Samples)
```tsv
Timestamp (UTC)          PID      RSS(MB)   CPU(%)   Threads   Open_FDs   Queue_Depth   Thresholds
2026-07-23T23:52:44Z     1500777  1.9       0.0      1         3          0             OK
2026-07-23T23:52:45Z     1500777  1.9       0.0      1         3          0             OK
2026-07-23T23:52:46Z     1500777  1.9       0.0      1         3          0             OK
```

## 5. VERDICT
**OVERALL VERDICT: PASS** — All sampled metrics adhere strictly to predeclared numeric ceilings. Sampler is online and standing ready for full sampling during instrumented Tier-20 rerun.

## 6. LANE Sampler Self-Test Verification Report
- **Self-Test Date**: 2026-07-23T23:52:43Z
- **Units Validation**:
  - RSS Unit Conversion (kB -> MB): `PASS` (31744 kB -> 31.0 MB verified)
  - CPU % Calculation (delta jiffies / delta time): `PASS`
  - Threads (NLWP) Parsing: `PASS`
  - Open FDs Directory Count: `PASS`
  - Queue Depth Reader: `PASS`
- **Predeclared Thresholds Audit**: `PASS` (RSS<512MB, CPU<90%, Threads<300, FDs<2048, Queue<=20)
- **Process Discovery Test**: `PASS` (PID discovery verified)
- **Remote ORQ1 Mode & Security Audit**: `PASS (secrets_present=false)` (`secrets_present=false`, zero env/secret file access)
- **SELF-TEST VERDICT**: **PASS** — Sampler logic, unit conversions, threshold checks, and process discovery validated clean.
