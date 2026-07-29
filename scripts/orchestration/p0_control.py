#!/usr/bin/env python3
"""Main Brain P0 check-in/out and Herdr fleet monitor (stdlib only)."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import tempfile
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[2]
STATE = ROOT / ".deploy-control" / "p0"
CHECKINS = STATE / "checkins"
EVENTS = STATE / "events.jsonl"
MONITOR = STATE / "monitor.jsonl"
CONTROL = STATE / "control.json"
ACTIVE_STATUSES = {"IN_PROGRESS", "BLOCKED"}


def utc_now() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def parse_utc(value: str) -> datetime:
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def safe_component(value: str) -> str:
    cleaned = re.sub(r"[^A-Za-z0-9_.-]+", "-", value.strip()).strip("-.")
    if not cleaned:
        raise ValueError("identifier becomes empty after normalization")
    return cleaned[:96]


def ensure_state() -> None:
    CHECKINS.mkdir(parents=True, exist_ok=True)
    (STATE / "handoffs").mkdir(parents=True, exist_ok=True)
    (STATE / "evidence").mkdir(parents=True, exist_ok=True)


def load_json(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as handle:
        value = json.load(handle)
    if not isinstance(value, dict):
        raise ValueError(f"expected JSON object: {path}")
    return value


def atomic_write(path: Path, value: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, tmp_name = tempfile.mkstemp(prefix=f".{path.name}.", dir=path.parent, text=True)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as handle:
            json.dump(value, handle, ensure_ascii=False, indent=2, sort_keys=True)
            handle.write("\n")
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(tmp_name, path)
    finally:
        try:
            os.unlink(tmp_name)
        except FileNotFoundError:
            pass


def append_jsonl(path: Path, value: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(value, ensure_ascii=False, sort_keys=True) + "\n")
        handle.flush()
        os.fsync(handle.fileno())


def all_records() -> list[tuple[Path, dict[str, Any]]]:
    ensure_state()
    records: list[tuple[Path, dict[str, Any]]] = []
    for path in sorted(CHECKINS.glob("*.json")):
        # Human-readable six-hour receipts share this directory but are not
        # p0_control state records; only the tool-generated records are authoritative.
        if path.name.startswith(("CHECKIN__", "CHECKOUT__")):
            continue
        try:
            records.append((path, load_json(path)))
        except (OSError, ValueError, json.JSONDecodeError) as exc:
            raise RuntimeError(f"invalid check-in {path}: {exc}") from exc
    return records


def active_records() -> list[tuple[Path, dict[str, Any]]]:
    return [(path, rec) for path, rec in all_records() if rec.get("status") in ACTIVE_STATUSES]


def normalize_file(value: str) -> str:
    path = Path(value)
    if path.is_absolute():
        try:
            path = path.resolve().relative_to(ROOT.resolve())
        except ValueError as exc:
            raise ValueError(f"file lock must be inside repository: {value}") from exc
    normalized = Path(os.path.normpath(str(path)))
    if str(normalized).startswith("../") or str(normalized) == "..":
        raise ValueError(f"file lock escapes repository: {value}")
    return normalized.as_posix()


def find_record(agent: str, task: str) -> tuple[Path, dict[str, Any]]:
    candidates = [
        (path, rec)
        for path, rec in all_records()
        if rec.get("agent") == agent and rec.get("task") == task and rec.get("status") in ACTIVE_STATUSES
    ]
    if len(candidates) != 1:
        raise ValueError(f"expected one active record for agent={agent!r} task={task!r}; found {len(candidates)}")
    return candidates[0]


def transition_event(action: str, path: Path, record: dict[str, Any]) -> None:
    append_jsonl(
        EVENTS,
        {
            "at": utc_now(),
            "action": action,
            "record": str(path.relative_to(ROOT)),
            "agent": record.get("agent"),
            "lane": record.get("lane"),
            "task": record.get("task"),
            "status": record.get("status"),
            "progress": record.get("progress"),
        },
    )


def check_in(args: argparse.Namespace) -> int:
    ensure_state()
    files = sorted({normalize_file(value) for value in args.files})
    if not files:
        raise ValueError("at least one exact --files path is required; use a namespaced handoff path for read-only lanes")

    conflicts: list[str] = []
    requested = set(files)
    for path, rec in active_records():
        overlap = requested.intersection(rec.get("files_locked", []))
        if overlap:
            conflicts.append(f"{path.name}: {', '.join(sorted(overlap))}")
    if conflicts:
        raise ValueError("file-lock overlap with active assignment: " + "; ".join(conflicts))

    now = utc_now()
    stamp = now.replace("-", "").replace(":", "").replace("Z", "Z")
    name = f"{safe_component(args.agent)}__{safe_component(args.task)}__{stamp}.json"
    path = CHECKINS / name
    record = {
        "schema": "main-brain-p0-checkin.v1",
        "agent": args.agent,
        "pane_id": args.pane_id,
        "lane": args.lane,
        "task": args.task,
        "activity": args.activity,
        "status": "IN_PROGRESS",
        "progress": 0,
        "started_at": now,
        "last_heartbeat_at": now,
        "finished_at": None,
        "files_locked": files,
        "depends_on": args.depends_on or [],
        "blocker": None,
        "evidence": [],
        "validation": None,
        "summary": None,
    }
    atomic_write(path, record)
    transition_event("CHECK_IN", path, record)
    print(path.relative_to(ROOT))
    return 0


def heartbeat(args: argparse.Namespace) -> int:
    path, record = find_record(args.agent, args.task)
    if not 0 <= args.progress <= 99:
        raise ValueError("heartbeat progress must be between 0 and 99")
    record["last_heartbeat_at"] = utc_now()
    record["progress"] = args.progress
    record["activity"] = args.activity
    if args.resume:
        record["status"] = "IN_PROGRESS"
        record["blocker"] = None
    atomic_write(path, record)
    transition_event("HEARTBEAT", path, record)
    print(path.relative_to(ROOT))
    return 0


def block(args: argparse.Namespace) -> int:
    path, record = find_record(args.agent, args.task)
    record["status"] = "BLOCKED"
    record["blocker"] = args.blocker
    record["last_heartbeat_at"] = utc_now()
    atomic_write(path, record)
    transition_event("BLOCK", path, record)
    print(path.relative_to(ROOT))
    return 0


def check_out(args: argparse.Namespace) -> int:
    path, record = find_record(args.agent, args.task)
    evidence = [normalize_file(value) for value in args.evidence]
    missing = [value for value in evidence if not (ROOT / value).exists()]
    if missing:
        raise ValueError("evidence path does not exist: " + ", ".join(missing))
    now = utc_now()
    record.update(
        {
            "status": args.status,
            "progress": 100 if args.status == "DONE" else record.get("progress", 0),
            "activity": "checkout",
            "last_heartbeat_at": now,
            "finished_at": now,
            "blocker": None if args.status == "DONE" else record.get("blocker"),
            "evidence": evidence,
            "validation": args.validation,
            "summary": args.summary,
        }
    )
    atomic_write(path, record)
    transition_event("CHECK_OUT", path, record)
    print(path.relative_to(ROOT))
    return 0


def herdr_agents() -> list[dict[str, Any]]:
    if os.environ.get("HERDR_ENV") != "1":
        raise RuntimeError("HERDR_ENV is not 1; refusing to inspect panes from outside Herdr")
    proc = subprocess.run(
        ["herdr", "agent", "list"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
        timeout=30,
    )
    payload = json.loads(proc.stdout)
    agents = payload.get("result", {}).get("agents", [])
    if not isinstance(agents, list):
        raise RuntimeError("unexpected `herdr agent list` response")
    return [agent for agent in agents if isinstance(agent, dict)]


def lock_overlaps(records: list[tuple[Path, dict[str, Any]]]) -> list[dict[str, Any]]:
    owners: dict[str, list[str]] = {}
    for _, rec in records:
        key = f"{rec.get('agent')}:{rec.get('task')}"
        for locked in rec.get("files_locked", []):
            owners.setdefault(locked, []).append(key)
    return [
        {"file": locked, "assignments": values}
        for locked, values in sorted(owners.items())
        if len(values) > 1
    ]


def monitor_once() -> tuple[dict[str, Any], int]:
    ensure_state()
    now_dt = datetime.now(timezone.utc)
    now = now_dt.replace(microsecond=0).isoformat().replace("+00:00", "Z")
    records = active_records()
    findings: list[dict[str, Any]] = []

    try:
        agents = herdr_agents()
        by_pane = {agent.get("pane_id"): agent for agent in agents if agent.get("pane_id")}
    except Exception as exc:  # monitor must persist the failure itself
        agents = []
        by_pane = {}
        findings.append({"severity": "RED", "kind": "HERDR_UNAVAILABLE", "detail": str(exc)})

    for overlap in lock_overlaps(records):
        findings.append({"severity": "RED", "kind": "LOCK_OVERLAP", **overlap})

    active_summary: list[dict[str, Any]] = []
    for path, rec in records:
        pane_id = rec.get("pane_id")
        live = by_pane.get(pane_id)
        live_status = live.get("agent_status") if live else "missing"
        heartbeat = rec.get("last_heartbeat_at")
        try:
            age_seconds = int((now_dt - parse_utc(heartbeat)).total_seconds())
        except Exception:
            age_seconds = None

        item = {
            "record": str(path.relative_to(ROOT)),
            "agent": rec.get("agent"),
            "lane": rec.get("lane"),
            "task": rec.get("task"),
            "control_status": rec.get("status"),
            "herdr_status": live_status,
            "heartbeat_age_seconds": age_seconds,
            "progress": rec.get("progress"),
        }
        active_summary.append(item)

        control_status = rec.get("status")
        if live is None:
            findings.append({"severity": "RED", "kind": "PANE_MISSING", **item})
        if age_seconds is None:
            findings.append({"severity": "RED", "kind": "HEARTBEAT_INVALID", **item})
        elif control_status == "IN_PROGRESS" and age_seconds > 900:
            findings.append({"severity": "RED", "kind": "HEARTBEAT_STALE", **item})
        if control_status == "IN_PROGRESS" and live_status != "working":
            findings.append({"severity": "RED", "kind": "NOT_WORKING", **item})
        if control_status == "BLOCKED":
            blocker = rec.get("blocker")
            if not isinstance(blocker, str) or not blocker.strip():
                findings.append({"severity": "RED", "kind": "BLOCKER_MISSING", **item})
            else:
                findings.append(
                    {
                        "severity": "AMBER",
                        "kind": "BLOCKED",
                        **item,
                        "blocker": blocker,
                    }
                )

    control = load_json(CONTROL) if CONTROL.exists() else {"assignments": []}
    for assignment in control.get("assignments", []):
        if not isinstance(assignment, dict) or assignment.get("status") != "ACTIVE":
            continue
        matching = [
            rec
            for _, rec in records
            if rec.get("agent") == assignment.get("agent") and rec.get("task") == assignment.get("task")
        ]
        if not matching:
            findings.append({"severity": "RED", "kind": "ACTIVE_WITHOUT_CHECKIN", "assignment": assignment})

    severity = "GREEN"
    if any(item["severity"] == "RED" for item in findings):
        severity = "RED"
    elif findings:
        severity = "AMBER"

    snapshot = {
        "schema": "main-brain-p0-monitor.v1",
        "at": now,
        "severity": severity,
        "active_count": len(active_summary),
        "detected_agent_count": len(agents),
        "active": active_summary,
        "findings": findings,
        "unassigned_panes_ignored": True,
    }
    append_jsonl(MONITOR, snapshot)
    return snapshot, 1 if severity == "RED" else 0


def monitor(args: argparse.Namespace) -> int:
    if args.once:
        snapshot, status = monitor_once()
        print(json.dumps(snapshot, ensure_ascii=False, indent=2, sort_keys=True))
        return status

    iteration = 0
    while args.iterations == 0 or iteration < args.iterations:
        snapshot, _ = monitor_once()
        print(json.dumps(snapshot, ensure_ascii=False, sort_keys=True), flush=True)
        iteration += 1
        if args.iterations == 0 or iteration < args.iterations:
            time.sleep(args.interval)
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)

    start = sub.add_parser("check-in", help="create a lock-bearing active assignment record")
    start.add_argument("--agent", required=True)
    start.add_argument("--pane-id", required=True)
    start.add_argument("--lane", required=True)
    start.add_argument("--task", required=True)
    start.add_argument("--activity", required=True)
    start.add_argument("--files", nargs="+", required=True)
    start.add_argument("--depends-on", nargs="*")
    start.set_defaults(func=check_in)

    beat = sub.add_parser("heartbeat", help="refresh progress for one active assignment")
    beat.add_argument("--agent", required=True)
    beat.add_argument("--task", required=True)
    beat.add_argument("--progress", type=int, required=True)
    beat.add_argument("--activity", required=True)
    beat.add_argument("--resume", action="store_true")
    beat.set_defaults(func=heartbeat)

    blocked = sub.add_parser("block", help="record a concrete blocker")
    blocked.add_argument("--agent", required=True)
    blocked.add_argument("--task", required=True)
    blocked.add_argument("--blocker", required=True)
    blocked.set_defaults(func=block)

    finish = sub.add_parser("check-out", help="finish with evidence and focused validation")
    finish.add_argument("--agent", required=True)
    finish.add_argument("--task", required=True)
    finish.add_argument("--status", choices=("DONE", "FAILED"), default="DONE")
    finish.add_argument("--evidence", nargs="+", required=True)
    finish.add_argument("--validation", required=True)
    finish.add_argument("--summary", required=True)
    finish.set_defaults(func=check_out)

    watch = sub.add_parser("monitor", help="compare active disk assignments with live Herdr state")
    mode = watch.add_mutually_exclusive_group(required=True)
    mode.add_argument("--once", action="store_true")
    mode.add_argument("--interval", type=int, metavar="SECONDS")
    watch.add_argument("--iterations", type=int, default=0, help="0 means run until interrupted")
    watch.set_defaults(func=monitor)
    return parser


def main() -> int:
    try:
        args = build_parser().parse_args()
        if args.command == "monitor" and not args.once:
            if args.interval < 60:
                raise ValueError("monitor interval must be at least 60 seconds")
        return args.func(args)
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError, json.JSONDecodeError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
