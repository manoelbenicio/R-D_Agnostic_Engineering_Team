#!/usr/bin/env python3
"""Fail-closed Git blob gate for staged changes and newly reachable commits."""
from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, Sequence

WARN_BYTES = 5_242_880
ALLOWLIST_BYTES = 10_485_760
HARD_REJECT_BYTES = 100_000_000
ZERO_OID = "0" * 40
OLD_ROLLBACK_REF = "refs/heads/task16-rc-16bfcb4"
DENIED_BLOB_OIDS = {
    "b0e282b997de2ff35a56886cf6a6c8b4655e1031",
    "fa62eefd5707ffbd15e372cd82d7bd10fa6e2490",
    "0693a65276ed5b6bb8ebf09f6d6d420100cec934",
    "053d8a10aa84a4e2de557880847e8f337968fb1e",
}
WINDOWS_DRIVE_ROOT = re.compile(r"^[A-Za-z]:")
SESSION_AUDIT_EXPORT = re.compile(
    r"^(?:[^/]*(?:session|audit)[^/]*\.(?:json|jsonl|ndjson|csv|zip|tar|tgz|gz)|"
    r"(?:exports?|sessions?|audit-exports?)/.*\.(?:json|jsonl|ndjson|csv|zip|tar|tgz|gz))$",
    re.IGNORECASE,
)


@dataclass(frozen=True)
class Blob:
    path: str
    oid: str
    size: int


def git(args: Sequence[str], *, data: bytes | None = None) -> bytes:
    proc = subprocess.run(
        ["git", *args], input=data, stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )
    if proc.returncode:
        message = proc.stderr.decode("utf-8", "replace").strip()
        raise RuntimeError(f"git {' '.join(args)} failed: {message}")
    return proc.stdout


def repository_root() -> Path:
    return Path(git(["rev-parse", "--show-toplevel"]).decode().strip())


def load_allowlist(root: Path) -> dict[str, tuple[int, str]]:
    result: dict[str, tuple[int, str]] = {}
    source = root / ".git-large-files.allow"
    try:
        lines = source.read_text(encoding="utf-8").splitlines()
    except OSError as exc:
        raise RuntimeError(f"cannot read {source}: {exc}") from exc
    for number, line in enumerate(lines, 1):
        if not line or line.startswith("#"):
            continue
        fields = line.split("\t")
        if len(fields) != 3:
            raise RuntimeError(f"invalid allowlist row {number}: expected three tab fields")
        path, size_text, oid = fields
        try:
            size = int(size_text)
        except ValueError as exc:
            raise RuntimeError(f"invalid allowlist size on row {number}") from exc
        if path.startswith("/") or "\\" in path or not re.fullmatch(r"[0-9a-f]{40}", oid):
            raise RuntimeError(f"invalid allowlist identity on row {number}")
        if path in result:
            raise RuntimeError(f"duplicate allowlist path on row {number}")
        result[path] = (size, oid)
    return result


def object_size(oid: str) -> int:
    return int(git(["cat-file", "-s", oid]).decode().strip())


def staged_blobs() -> list[Blob]:
    paths = git(["diff", "--cached", "--name-only", "-z", "--diff-filter=ACMR", "HEAD", "--"])
    blobs: list[Blob] = []
    for raw_path in paths.split(b"\0"):
        if not raw_path:
            continue
        path = os.fsdecode(raw_path)
        row = git(["ls-files", "--stage", "-z", "--", path]).split(b"\0", 1)[0]
        if not row:
            continue
        metadata, actual_path = row.split(b"\t", 1)
        mode, oid_bytes, stage = metadata.split()
        if stage != b"0" or mode == b"160000":
            continue
        oid = oid_bytes.decode()
        blobs.append(Blob(os.fsdecode(actual_path), oid, object_size(oid)))
    return blobs


def revision_commits(revisions: Sequence[str]) -> list[str]:
    raw = git(["rev-list", "--topo-order", *revisions])
    return [line.decode() for line in raw.splitlines() if line]


def revision_blobs(revisions: Sequence[str]) -> list[Blob]:
    commits = revision_commits(revisions)
    found: dict[tuple[str, str], Blob] = {}
    for commit in commits:
        changed = git([
            "diff-tree", "--root", "-m", "-r", "--no-commit-id", "--name-only", "-z",
            "--diff-filter=ACMR", commit,
        ])
        for raw_path in changed.split(b"\0"):
            if not raw_path:
                continue
            path = os.fsdecode(raw_path)
            rows = git(["ls-tree", "-z", commit, "--", path]).split(b"\0")
            for row in rows:
                if not row:
                    continue
                metadata, actual_path = row.split(b"\t", 1)
                mode, kind, oid_bytes = metadata.split()
                if kind != b"blob" or mode == b"160000":
                    continue
                oid = oid_bytes.decode()
                key = (os.fsdecode(actual_path), oid)
                found[key] = Blob(key[0], oid, object_size(oid))
    # Object-level denial also covers blobs whose path was not emitted or was deduplicated by Git.
    objects = git(["rev-list", "--objects", *revisions])
    for row in objects.splitlines():
        oid = row.split(b" ", 1)[0].decode()
        if oid in DENIED_BLOB_OIDS:
            found[("<denied-object>", oid)] = Blob("<denied-object>", oid, object_size(oid))
    return list(found.values())


def inspect(blobs: Iterable[Blob], allowlist: dict[str, tuple[int, str]]) -> int:
    errors: list[str] = []
    warnings: list[str] = []
    for blob in sorted(set(blobs), key=lambda item: (item.path.encode("utf-8", "surrogateescape"), item.oid)):
        if blob.oid in DENIED_BLOB_OIDS:
            errors.append(f"denied omitted-artifact blob: {blob.oid} ({blob.path})")
        if WINDOWS_DRIVE_ROOT.match(blob.path):
            errors.append(f"root Windows-drive-like path is forbidden: {blob.path}")
        if SESSION_AUDIT_EXPORT.search(blob.path):
            errors.append(f"session/audit export path is forbidden: {blob.path}")
        if blob.size > HARD_REJECT_BYTES:
            errors.append(f"blob exceeds {HARD_REJECT_BYTES} bytes: {blob.path} ({blob.size})")
        elif blob.size >= ALLOWLIST_BYTES:
            if allowlist.get(blob.path) != (blob.size, blob.oid):
                errors.append(f"blob requires exact allowlist identity: {blob.path} ({blob.size}, {blob.oid})")
        elif blob.size >= WARN_BYTES:
            warnings.append(f"large blob warning: {blob.path} ({blob.size}, {blob.oid})")
    for message in warnings:
        print(f"WARN: {message}", file=sys.stderr)
    for message in errors:
        print(f"ERROR: {message}", file=sys.stderr)
    return 1 if errors else 0


def target_remote_baselines(remote_name: str | None) -> list[str]:
    if not remote_name:
        raise RuntimeError("pre-push target remote name unavailable")
    git(["check-ref-format", f"refs/remotes/{remote_name}/baseline"])
    if not git(["remote", "get-url", "--all", remote_name]).splitlines():
        raise RuntimeError(f"pre-push target remote mapping unavailable: {remote_name}")
    raw = git([
        "for-each-ref", "--format=%(objectname)", f"refs/remotes/{remote_name}/"
    ])
    baselines = sorted({line.decode() for line in raw.splitlines() if line})
    if not baselines:
        raise RuntimeError(f"pre-push target remote baseline unavailable: {remote_name}")
    if any(not re.fullmatch(r"[0-9a-f]{40}", oid) for oid in baselines):
        raise RuntimeError(f"invalid pre-push target remote baseline: {remote_name}")
    return baselines


def ranges_from_pre_push(stdin: Iterable[str], remote_name: str | None) -> list[list[str]]:
    target_baselines = target_remote_baselines(remote_name)
    ranges: list[list[str]] = []
    for number, line in enumerate(stdin, 1):
        fields = line.split()
        if len(fields) != 4:
            raise RuntimeError(f"invalid pre-push input row {number}")
        local_ref, local_oid, _remote_ref, remote_oid = fields
        if local_ref == OLD_ROLLBACK_REF:
            raise RuntimeError(f"push forbidden from rollback-only ref {OLD_ROLLBACK_REF}")
        if local_oid == ZERO_OID:
            continue
        if not re.fullmatch(r"[0-9a-f]{40}", local_oid):
            raise RuntimeError(f"invalid local OID on pre-push row {number}")
        if remote_oid == ZERO_OID:
            # Exclude only refs known on the target remote. Another remote must never
            # hide objects that are new to this target.
            ranges.append([local_oid, *[f"^{oid}" for oid in target_baselines]])
        elif re.fullmatch(r"[0-9a-f]{40}", remote_oid):
            ranges.append([local_oid, f"^{remote_oid}"])
        else:
            raise RuntimeError(f"invalid remote OID on pre-push row {number}")
    return ranges


def main() -> int:
    parser = argparse.ArgumentParser()
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--staged", action="store_true")
    mode.add_argument("--range", dest="revision_range")
    mode.add_argument("--pre-push", action="store_true")
    parser.add_argument("--remote-name")
    args = parser.parse_args()
    try:
        allowlist = load_allowlist(repository_root())
        if args.staged:
            blobs = staged_blobs()
        elif args.revision_range:
            blobs = revision_blobs([args.revision_range])
        else:
            ranges = ranges_from_pre_push(sys.stdin, args.remote_name)
            blobs = [blob for revisions in ranges for blob in revision_blobs(revisions)]
        return inspect(blobs, allowlist)
    except (OSError, RuntimeError, ValueError) as exc:
        print(f"ERROR: integrity gate indeterminate: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
