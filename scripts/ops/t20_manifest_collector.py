#!/usr/bin/env python3
"""T20 real-run manifest collector (scripts/ops; NOT product source).

Produces the provenance manifest the hardened Tier-20 verifier
(internal/daemon/observability/e2e/t20_hardened_verify_test.go) binds evidence to.
It ONLY reads real run inputs and FAILS CLOSED if anything is missing, malformed,
count-mismatched, stale, or unverifiable. It NEVER generates spans/IDs, never pads
to a count, and never infers a missing value.

Collected fields:
  - run_id, window_start, window_end            (from the launcher; not generated)
  - expected_tasks: exact N (default 20) task IDs from the launched batch record
  - exports: backend/daemon span files -> {inode, mtime_utc, size, sha256}
  - launched_procs: observed child PIDs tied to /proc starttime captured during the run
                    (re-verified against live /proc when the process still exists)
  - persisted: exact N terminal rows -> {row_id, status}

Live invocation is documented in the module docstring footer; a live run is NOT
performed here and no live PASS is claimed. `--self-test` runs unit tests over
temp fixtures only.
"""
import argparse
import datetime as _dt
import hashlib
import json
import os
import sys


class CollectError(Exception):
    """Raised on any fail-closed condition."""


def parse_rfc3339(value):
    v = (value or "").strip()
    if not v:
        raise CollectError("empty timestamp")
    if v.endswith("Z"):
        v = v[:-1] + "+00:00"
    try:
        dt = _dt.datetime.fromisoformat(v)
    except ValueError as exc:
        raise CollectError("invalid RFC3339 timestamp: %s" % value) from exc
    if dt.tzinfo is None:
        raise CollectError("timestamp must be timezone-aware: %s" % value)
    return dt.astimezone(_dt.timezone.utc)


def _load_json(path):
    if not os.path.isfile(path):
        raise CollectError("input file missing: %s" % path)
    try:
        with open(path, "r", encoding="utf-8") as fh:
            return json.load(fh)
    except (OSError, ValueError) as exc:
        raise CollectError("unreadable/invalid JSON: %s (%s)" % (path, exc)) from exc


def file_provenance(path, window_start, window_end, grace_seconds):
    """inode+mtime+sha256 of an export file; fail closed if missing/unreadable/stale."""
    if not os.path.isfile(path):
        raise CollectError("export file missing: %s" % path)
    try:
        st = os.stat(path)
        h = hashlib.sha256()
        with open(path, "rb") as fh:
            for chunk in iter(lambda: fh.read(65536), b""):
                h.update(chunk)
    except OSError as exc:
        raise CollectError("export file unreadable: %s (%s)" % (path, exc)) from exc
    mtime = _dt.datetime.fromtimestamp(st.st_mtime, tz=_dt.timezone.utc)
    lo = window_start - _dt.timedelta(seconds=grace_seconds)
    hi = window_end + _dt.timedelta(seconds=grace_seconds)
    if mtime < lo or mtime > hi:
        raise CollectError(
            "export file stale: %s mtime=%s outside run window [%s, %s]"
            % (path, mtime.isoformat(), lo.isoformat(), hi.isoformat())
        )
    return {
        "path": os.path.abspath(path),
        "inode": st.st_ino,
        "size": st.st_size,
        "mtime_utc": mtime.isoformat(),
        "sha256": h.hexdigest(),
    }


def load_task_ids(path, expected_count):
    """Exact `expected_count` unique non-empty task IDs from the launched batch."""
    data = _load_json(path)
    if not isinstance(data, list):
        raise CollectError("batch tasks must be a JSON list: %s" % path)
    ids = []
    seen = set()
    for item in data:
        if not isinstance(item, str) or not item.strip():
            raise CollectError("batch task id must be a non-empty string")
        tid = item.strip()
        if tid in seen:
            raise CollectError("duplicate task id in batch: %s" % tid)
        seen.add(tid)
        ids.append(tid)
    if len(ids) != expected_count:
        raise CollectError(
            "launched batch has %d task ids, expected exactly %d (no padding/inference)"
            % (len(ids), expected_count)
        )
    return ids


def read_proc_starttime(pid, proc_root="/proc"):
    """Return field-22 (starttime, clock ticks) of /proc/<pid>/stat, or raise.

    Parses robustly around a comm that may contain spaces/parens by splitting
    after the final ')'.
    """
    stat_path = os.path.join(proc_root, str(pid), "stat")
    try:
        with open(stat_path, "r", encoding="utf-8") as fh:
            data = fh.read()
    except OSError as exc:
        raise CollectError("cannot read %s (%s)" % (stat_path, exc)) from exc
    rp = data.rfind(")")
    if rp < 0:
        raise CollectError("malformed stat (no comm) for pid %s" % pid)
    after = data[rp + 2:].split()
    # after[0] == field 3 (state); field 22 == after[19].
    if len(after) < 20:
        raise CollectError("malformed stat (too few fields) for pid %s" % pid)
    try:
        return int(after[19])
    except ValueError as exc:
        raise CollectError("non-integer starttime for pid %s" % pid) from exc


def load_launched_procs(path, proc_root):
    """Observed child PIDs each with a starttime captured live during the run.

    Each sample MUST carry a positive integer starttime (captured while the child
    was alive). If /proc/<pid> still exists, its starttime MUST match the sample
    (else PID reuse/spoof -> fail closed). No pid or starttime is ever invented.
    """
    data = _load_json(path)
    if not isinstance(data, list) or not data:
        raise CollectError("pid samples must be a non-empty JSON list: %s" % path)
    procs = []
    seen = set()
    for entry in data:
        if not isinstance(entry, dict) or "pid" not in entry or "starttime" not in entry:
            raise CollectError("pid sample requires {pid, starttime} captured during the run")
        try:
            pid = int(entry["pid"])
            starttime = int(entry["starttime"])
        except (TypeError, ValueError) as exc:
            raise CollectError("pid/starttime must be integers") from exc
        if pid <= 0 or starttime <= 0:
            raise CollectError("pid and starttime must be > 0 (no sentinels): pid=%s" % pid)
        if pid in seen:
            raise CollectError("duplicate pid in samples: %d" % pid)
        seen.add(pid)
        alive = os.path.isdir(os.path.join(proc_root, str(pid)))
        if alive:
            live_start = read_proc_starttime(pid, proc_root)
            if live_start != starttime:
                raise CollectError(
                    "pid %d starttime mismatch (sample=%d live=%d): PID reuse/spoof"
                    % (pid, starttime, live_start)
                )
        procs.append({"pid": pid, "starttime": starttime, "alive_at_collection": alive})
    return procs


def load_persisted_rows(path, expected_count):
    """Exact `expected_count` persisted terminal rows: {row_id, status}, unique ids."""
    data = _load_json(path)
    if not isinstance(data, list):
        raise CollectError("persisted rows must be a JSON list: %s" % path)
    rows = []
    seen = set()
    for entry in data:
        if not isinstance(entry, dict) or "row_id" not in entry or "status" not in entry:
            raise CollectError("persisted row requires {row_id, status}")
        rid = entry["row_id"]
        status = entry["status"]
        if not isinstance(rid, str) or not rid.strip() or not isinstance(status, str) or not status.strip():
            raise CollectError("persisted row_id/status must be non-empty strings")
        if rid in seen:
            raise CollectError("duplicate persisted row_id: %s" % rid)
        seen.add(rid)
        rows.append({"row_id": rid.strip(), "status": status.strip()})
    if len(rows) != expected_count:
        raise CollectError(
            "persisted rows count=%d, expected exactly %d (no padding/inference)"
            % (len(rows), expected_count)
        )
    return rows


def collect(args):
    run_id = (args.run_id or "").strip()
    if not run_id:
        raise CollectError("run_id is required and is never generated")
    window_start = parse_rfc3339(args.window_start)
    window_end = parse_rfc3339(args.window_end)
    if not window_end > window_start:
        raise CollectError("window_end must be after window_start")

    expected = int(args.expected_count)
    manifest = {
        "schema": "t20-run-manifest.v1",
        "status": "COLLECTED",
        "run_id": run_id,
        "window_start": window_start.isoformat(),
        "window_end": window_end.isoformat(),
        "expected_count": expected,
        "expected_tasks": load_task_ids(args.batch_tasks, expected),
        "exports": {
            "backend": file_provenance(args.backend_export, window_start, window_end, args.grace_seconds),
            "daemon": file_provenance(args.daemon_export, window_start, window_end, args.grace_seconds),
        },
        "launched_procs": load_launched_procs(args.pid_samples, args.proc_root),
        "persisted": load_persisted_rows(args.persisted_rows, expected),
    }
    return manifest


def main(argv=None):
    parser = argparse.ArgumentParser(description="T20 real-run manifest collector (fail-closed).")
    parser.add_argument("--self-test", action="store_true", help="run unit self-test over temp fixtures and exit")
    parser.add_argument("--run-id")
    parser.add_argument("--window-start")
    parser.add_argument("--window-end")
    parser.add_argument("--batch-tasks")
    parser.add_argument("--backend-export")
    parser.add_argument("--daemon-export")
    parser.add_argument("--pid-samples")
    parser.add_argument("--persisted-rows")
    parser.add_argument("--expected-count", type=int, default=20)
    parser.add_argument("--grace-seconds", type=int, default=300)
    parser.add_argument("--proc-root", default="/proc")
    parser.add_argument("--out")
    args = parser.parse_args(argv)

    if args.self_test:
        return _run_self_test()

    required = ["run_id", "window_start", "window_end", "batch_tasks",
                "backend_export", "daemon_export", "pid_samples", "persisted_rows"]
    missing = [r for r in required if not getattr(args, r)]
    if missing:
        _fail_closed("missing required inputs: %s" % ", ".join(missing))

    try:
        manifest = collect(args)
    except CollectError as exc:
        _fail_closed(str(exc))
    out = json.dumps(manifest, indent=2, sort_keys=True)
    if args.out:
        with open(args.out, "w", encoding="utf-8") as fh:
            fh.write(out + "\n")
    else:
        sys.stdout.write(out + "\n")
    return 0


def _fail_closed(reason):
    sys.stderr.write(json.dumps({"status": "FAILED_CLOSED", "reason": reason}) + "\n")
    sys.exit(2)


# --------------------------------------------------------------------------
# Unit self-test (temp fixtures only; no live data, no /proc dependency).
# --------------------------------------------------------------------------
def _run_self_test():
    import tempfile
    import unittest

    def write_stat(proc_root, pid, starttime):
        d = os.path.join(proc_root, str(pid))
        os.makedirs(d, exist_ok=True)
        # 52 fields: field1=pid, field2=(comm), field3..52 follow. after holds
        # fields 3..52 (50 entries); field22 == after[19] (the reader parses after
        # the final ')'). No extra state token — after[0] IS field 3.
        after = [str(x) for x in range(3, 53)]  # fields 3..52 (50 entries)
        after[19] = str(starttime)              # field 22 == after[19]
        with open(os.path.join(d, "stat"), "w", encoding="utf-8") as fh:
            fh.write("%d (procx) %s\n" % (pid, " ".join(after)))

    def write_json(path, obj):
        with open(path, "w", encoding="utf-8") as fh:
            json.dump(obj, fh)

    class Args(object):
        pass

    class T(unittest.TestCase):
        def _fixture(self, tmp, n=20, pid_alive=True):
            proc = os.path.join(tmp, "proc")
            os.makedirs(proc, exist_ok=True)
            tasks = ["task-%02d-%x" % (i, i * 2654435761 & 0xffff) for i in range(n)]
            write_json(os.path.join(tmp, "tasks.json"), tasks)
            pids = []
            for i in range(n):
                pid = 4000 + i
                start = 100000 + i
                if pid_alive:
                    write_stat(proc, pid, start)
                pids.append({"pid": pid, "starttime": start})
            write_json(os.path.join(tmp, "pids.json"), pids)
            rows = [{"row_id": "res-%02d-%x" % (i, i), "status": "completed"} for i in range(n)]
            write_json(os.path.join(tmp, "rows.json"), rows)
            be = os.path.join(tmp, "backend.json")
            de = os.path.join(tmp, "daemon.json")
            with open(be, "w") as fh:
                fh.write('[{"hop":"ingress"}]')
            with open(de, "w") as fh:
                fh.write('[{"hop":"admission"}]')
            a = Args()
            a.run_id = "run-selftest-1"
            a.window_start = "2026-07-24T00:00:00Z"
            a.window_end = "2026-07-24T01:00:00Z"
            # set export mtimes inside window
            mid = _dt.datetime(2026, 7, 24, 0, 30, tzinfo=_dt.timezone.utc).timestamp()
            os.utime(be, (mid, mid))
            os.utime(de, (mid, mid))
            a.batch_tasks = os.path.join(tmp, "tasks.json")
            a.backend_export = be
            a.daemon_export = de
            a.pid_samples = os.path.join(tmp, "pids.json")
            a.persisted_rows = os.path.join(tmp, "rows.json")
            a.expected_count = n
            a.grace_seconds = 300
            a.proc_root = proc
            return a

        def test_happy_path(self):
            with tempfile.TemporaryDirectory() as tmp:
                m = collect(self._fixture(tmp))
                self.assertEqual(m["status"], "COLLECTED")
                self.assertEqual(len(m["expected_tasks"]), 20)
                self.assertEqual(len(m["launched_procs"]), 20)
                self.assertEqual(len(m["persisted"]), 20)
                self.assertEqual(len(m["exports"]["backend"]["sha256"]), 64)
                self.assertTrue(m["exports"]["backend"]["inode"] > 0)
                self.assertTrue(all(p["starttime"] > 0 for p in m["launched_procs"]))

        def test_missing_export_fails_closed(self):
            with tempfile.TemporaryDirectory() as tmp:
                a = self._fixture(tmp)
                os.remove(a.backend_export)
                self.assertRaises(CollectError, collect, a)

        def test_stale_export_fails_closed(self):
            with tempfile.TemporaryDirectory() as tmp:
                a = self._fixture(tmp)
                old = _dt.datetime(2026, 7, 20, tzinfo=_dt.timezone.utc).timestamp()
                os.utime(a.backend_export, (old, old))
                self.assertRaises(CollectError, collect, a)

        def test_wrong_task_count_fails_closed(self):
            with tempfile.TemporaryDirectory() as tmp:
                a = self._fixture(tmp, n=20)
                write_json(a.batch_tasks, ["task-%d" % i for i in range(19)])  # 19 != 20
                self.assertRaises(CollectError, collect, a)

        def test_pid_starttime_mismatch_fails_closed(self):
            with tempfile.TemporaryDirectory() as tmp:
                a = self._fixture(tmp)
                # rewrite one live /proc stat with a DIFFERENT starttime (PID reuse)
                write_stat(a.proc_root, 4000, 999999)
                self.assertRaises(CollectError, collect, a)

        def test_pid_sample_without_starttime_fails_closed(self):
            with tempfile.TemporaryDirectory() as tmp:
                a = self._fixture(tmp)
                write_json(a.pid_samples, [{"pid": 4000}])  # no starttime -> cannot infer
                self.assertRaises(CollectError, collect, a)

        def test_sentinel_pid_fails_closed(self):
            with tempfile.TemporaryDirectory() as tmp:
                a = self._fixture(tmp)
                write_json(a.pid_samples, [{"pid": 0, "starttime": 1}])
                self.assertRaises(CollectError, collect, a)

        def test_wrong_persisted_count_fails_closed(self):
            with tempfile.TemporaryDirectory() as tmp:
                a = self._fixture(tmp)
                write_json(a.persisted_rows, [{"row_id": "r%d" % i, "status": "completed"} for i in range(19)])
                self.assertRaises(CollectError, collect, a)

        def test_duplicate_persisted_row_fails_closed(self):
            with tempfile.TemporaryDirectory() as tmp:
                a = self._fixture(tmp)
                rows = [{"row_id": "dup", "status": "completed"} for _ in range(20)]
                write_json(a.persisted_rows, rows)
                self.assertRaises(CollectError, collect, a)

        def test_dead_proc_uses_sampled_starttime(self):
            # process exited by collection time (no /proc dir): sampled starttime is
            # authoritative (captured during the run); still recorded, not inferred.
            with tempfile.TemporaryDirectory() as tmp:
                a = self._fixture(tmp, pid_alive=False)
                m = collect(a)
                self.assertEqual(len(m["launched_procs"]), 20)
                self.assertTrue(all(not p["alive_at_collection"] for p in m["launched_procs"]))

        def test_missing_run_id_fails_closed(self):
            with tempfile.TemporaryDirectory() as tmp:
                a = self._fixture(tmp)
                a.run_id = "  "
                self.assertRaises(CollectError, collect, a)

    suite = unittest.TestLoader().loadTestsFromTestCase(T)
    result = unittest.TextTestRunner(verbosity=2).run(suite)
    return 0 if result.wasSuccessful() else 1


# --------------------------------------------------------------------------
# LIVE INVOCATION (documented; NOT run here; no live PASS claimed):
#
#   1) During the authorized tier-20 run, a concurrent sampler (e.g.
#      scripts/ops/t20_resource_sampler.py) records each observed child PID with
#      its /proc/<pid>/stat field-22 starttime -> pids.json.
#   2) The launcher records the exact 20 enqueued task IDs -> tasks.json, and the
#      backend records the 20 persisted terminal rows (row_id,status) -> rows.json.
#   3) The instrumented harness exports spans -> backend.json / daemon.json.
#   4) Then:
#      python3 scripts/ops/t20_manifest_collector.py \
#        --run-id "$RUN_ID" --window-start "$START" --window-end "$END" \
#        --batch-tasks tasks.json --backend-export backend.json \
#        --daemon-export daemon.json --pid-samples pids.json \
#        --persisted-rows rows.json --out t20-run-manifest.json
#   Exit 0 + manifest => provenance collected; non-zero + FAILED_CLOSED => unusable.
#   The manifest feeds the hardened verifier's t20RunManifest gates.
# --------------------------------------------------------------------------
if __name__ == "__main__":
    sys.exit(main())
