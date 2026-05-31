"""Auth/session discovery routes for AgentVerse's CAO runtime."""

from __future__ import annotations

import json
import logging
import os
import re
import subprocess
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Iterable, Optional

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

logger = logging.getLogger(__name__)

auth_router = APIRouter(prefix="/auth", tags=["auth"])


class LoginRequest(BaseModel):
    provider: str
    config_dir: Optional[str] = None


class RevokeRequest(BaseModel):
    provider: str
    config_dir: Optional[str] = None


PROVIDERS: dict[str, dict[str, Any]] = {
    "claude_code": {
        "config_dirs": ["/root/.claude"],
        "credential_names": [
            "credentials.json",
            "config.json",
            "auth.json",
            ".credentials.json",
        ],
        "auth_method": "oauth",
        "login": ["claude", "auth", "login"],
        "logout": ["claude", "auth", "logout"],
        "env": ["CLAUDE_CONFIG_DIR"],
    },
    "codex": {
        "config_dirs": ["/root/.codex"],
        "credential_names": [
            "auth.json",
            "credentials.json",
            "config.json",
            "token.json",
            "session.json",
        ],
        "auth_method": "oauth",
        "login": ["codex", "auth", "login"],
        "logout": ["codex", "auth", "logout"],
        "env": ["CODEX_HOME", "CODEX_CONFIG_DIR"],
    },
    "gemini_cli": {
        "config_dirs": ["/root/.gemini", "/root/.config/gcloud"],
        "credential_names": [
            "application_default_credentials.json",
            "credentials.json",
            "config.json",
            "access_tokens.db",
        ],
        "auth_method": "gcloud",
        "login": ["gcloud", "auth", "login"],
        "logout": ["gcloud", "auth", "revoke"],
        "env": ["CLOUDSDK_CONFIG", "GEMINI_CONFIG_DIR"],
    },
    "kiro_cli": {
        "config_dirs": ["/root/.kiro"],
        "credential_names": [
            "auth.json",
            "credentials.json",
            "config.json",
            "settings.json",
            "token.json",
        ],
        "auth_method": "oauth",
        "login": ["kiro", "auth", "login"],
        "logout": ["kiro", "auth", "logout"],
        "env": ["KIRO_HOME", "KIRO_CONFIG_DIR"],
    },
}

EMAIL_KEYS = {"email", "account_email", "user_email", "username", "client_email"}
EXPIRY_KEYS = {
    "expires_at",
    "expiresAt",
    "expiry",
    "expiry_date",
    "expiryDate",
    "expiration",
    "token_expiry",
    "access_token_expires_at",
}
SUBSCRIPTION_KEYS = {"subscription_type", "subscription", "plan", "tier"}
TOKEN_KEYS = {
    "access_token",
    "refresh_token",
    "id_token",
    "token",
    "api_key",
    "key",
    "secret",
}
EMAIL_RE = re.compile(r"[\w.+-]+@[\w.-]+\.[A-Za-z]{2,}")
TOKEN_RE = re.compile(r"(access[_-]?token|refresh[_-]?token|id[_-]?token|api[_-]?key)", re.I)


@auth_router.get("/sessions")
async def list_auth_sessions() -> list[dict[str, Any]]:
    """Scan mounted CLI config directories for authenticated sessions."""
    sessions: list[dict[str, Any]] = []

    for provider, provider_config in PROVIDERS.items():
        try:
            session = _discover_provider_session(provider, provider_config)
            if session:
                sessions.append(session)
        except Exception as exc:
            logger.warning("Failed to scan auth session for %s: %s", provider, exc)

    return sessions


@auth_router.post("/login")
async def login(request: LoginRequest) -> dict[str, Any]:
    provider_config = _provider_or_404(request.provider)
    env = _env_for_provider(provider_config, request.config_dir)
    command = provider_config["login"]

    try:
        process = subprocess.Popen(
            command,
            env=env,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            start_new_session=True,
        )
    except FileNotFoundError as exc:
        raise HTTPException(status_code=500, detail=f"CLI command not found: {command[0]}") from exc
    except Exception as exc:
        raise HTTPException(status_code=500, detail=f"Failed to start login: {exc}") from exc

    return {
        "status": "started",
        "provider": request.provider,
        "pid": process.pid,
        "command": " ".join(command),
    }


@auth_router.delete("/sessions/{session_id}")
async def delete_auth_session(session_id: str, request: RevokeRequest) -> dict[str, Any]:
    provider_config = _provider_or_404(request.provider)
    session = _discover_provider_session(request.provider, provider_config, request.config_dir)
    if not session or session.get("id") != session_id:
        raise HTTPException(status_code=404, detail="Session not found")

    command = provider_config["logout"]
    command_succeeded = False
    command_error: str | None = None

    try:
        result = subprocess.run(
            command,
            env=_env_for_provider(provider_config, request.config_dir),
            capture_output=True,
            text=True,
            timeout=60,
            check=False,
        )
        command_succeeded = result.returncode == 0
        if not command_succeeded:
            command_error = (result.stderr or result.stdout or "").strip()
    except FileNotFoundError as exc:
        command_error = f"CLI command not found: {command[0]}"
        logger.info("Logout command unavailable for %s: %s", request.provider, exc)
    except subprocess.TimeoutExpired:
        command_error = "Logout command timed out"
    except Exception as exc:
        command_error = str(exc)

    removed_files = _remove_credential_files(request.provider, provider_config, request.config_dir)
    if not command_succeeded and not removed_files:
        raise HTTPException(
            status_code=500,
            detail=command_error or "Unable to revoke session",
        )

    return {
        "status": "revoked",
        "provider": request.provider,
        "session_id": session_id,
        "command_succeeded": command_succeeded,
        "removed_files": removed_files,
    }


def _provider_or_404(provider: str) -> dict[str, Any]:
    provider_config = PROVIDERS.get(provider)
    if not provider_config:
        raise HTTPException(status_code=404, detail=f"Unknown provider: {provider}")
    return provider_config


def _discover_provider_session(
    provider: str,
    provider_config: dict[str, Any],
    override_config_dir: str | None = None,
) -> dict[str, Any] | None:
    for config_dir in _candidate_config_dirs(provider_config, override_config_dir):
        credential_files = _find_credential_files(config_dir, provider_config["credential_names"])
        if not credential_files:
            continue

        facts = _extract_facts(credential_files)
        expires_at = facts.get("expires_at")
        token_exists = bool(facts.get("token_exists"))
        status = _session_status(token_exists, expires_at)

        return {
            "id": f"{provider}:default",
            "cli_provider": provider,
            "account_email": facts.get("account_email") or "CLI session",
            "config_dir": str(config_dir),
            "status": status,
            "expires_at": expires_at,
            "subscription_type": facts.get("subscription_type"),
            "auth_method": provider_config["auth_method"],
        }

    return None


def _candidate_config_dirs(
    provider_config: dict[str, Any],
    override_config_dir: str | None = None,
) -> list[Path]:
    raw_dirs = [override_config_dir] if override_config_dir else provider_config["config_dirs"]
    return [Path(path).expanduser() for path in raw_dirs if path]


def _find_credential_files(config_dir: Path, credential_names: Iterable[str]) -> list[Path]:
    if not config_dir.exists() or not config_dir.is_dir():
        return []

    files: list[Path] = []
    names = set(credential_names)

    for name in names:
        candidate = config_dir / name
        if candidate.exists() and candidate.is_file():
            files.append(candidate)

    try:
        for candidate in config_dir.rglob("*"):
            if not candidate.is_file() or candidate in files:
                continue
            if candidate.name in names:
                files.append(candidate)
    except OSError as exc:
        logger.debug("Unable to recurse config dir %s: %s", config_dir, exc)

    if files:
        return files

    return _find_token_like_json_files(config_dir)


def _find_token_like_json_files(config_dir: Path) -> list[Path]:
    matches: list[Path] = []
    try:
        for candidate in config_dir.rglob("*.json"):
            if not candidate.is_file():
                continue
            text = _safe_read_text(candidate)
            if text and (TOKEN_RE.search(text) or EMAIL_RE.search(text)):
                matches.append(candidate)
    except OSError as exc:
        logger.debug("Unable to inspect config dir %s: %s", config_dir, exc)
    return matches


def _extract_facts(files: Iterable[Path]) -> dict[str, Any]:
    facts: dict[str, Any] = {"token_exists": False}

    for path in files:
        text = _safe_read_text(path)
        if not text:
            facts["token_exists"] = True
            continue

        if TOKEN_RE.search(text):
            facts["token_exists"] = True

        if "account_email" not in facts:
            email_match = EMAIL_RE.search(text)
            if email_match:
                facts["account_email"] = email_match.group(0)

        data = _safe_json_loads(text)
        if data is None:
            continue

        flattened = list(_walk_json(data))
        if any(key in TOKEN_KEYS and value not in (None, "") for key, value in flattened):
            facts["token_exists"] = True

        _fill_first(facts, "account_email", flattened, EMAIL_KEYS)
        _fill_first(facts, "subscription_type", flattened, SUBSCRIPTION_KEYS)

        if "expires_at" not in facts:
            expires_at = _first_expiry(flattened)
            if expires_at:
                facts["expires_at"] = expires_at

    return facts


def _safe_read_text(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8", errors="ignore")
    except OSError as exc:
        logger.debug("Unable to read credential file %s: %s", path, exc)
        return ""


def _safe_json_loads(text: str) -> Any | None:
    try:
        return json.loads(text.lstrip("\ufeff"))
    except json.JSONDecodeError:
        return None


def _walk_json(value: Any) -> Iterable[tuple[str, Any]]:
    if isinstance(value, dict):
        for key, child in value.items():
            yield str(key), child
            yield from _walk_json(child)
    elif isinstance(value, list):
        for child in value:
            yield from _walk_json(child)


def _fill_first(
    facts: dict[str, Any],
    fact_key: str,
    flattened: list[tuple[str, Any]],
    keys: set[str],
) -> None:
    if fact_key in facts:
        return
    for key, value in flattened:
        if key in keys and isinstance(value, str) and value.strip():
            facts[fact_key] = value.strip()
            return


def _first_expiry(flattened: list[tuple[str, Any]]) -> str | None:
    for key, value in flattened:
        if key in EXPIRY_KEYS:
            parsed = _parse_expiry(value)
            if parsed:
                return parsed
    return None


def _parse_expiry(value: Any) -> str | None:
    if value in (None, ""):
        return None

    if isinstance(value, (int, float)):
        timestamp = float(value)
        if timestamp > 10_000_000_000:
            timestamp = timestamp / 1000
        return datetime.fromtimestamp(timestamp, tz=timezone.utc).isoformat().replace("+00:00", "Z")

    if not isinstance(value, str):
        return None

    raw = value.strip()
    if raw.isdigit():
        return _parse_expiry(int(raw))

    normalized = raw.replace("Z", "+00:00")
    try:
        parsed = datetime.fromisoformat(normalized)
    except ValueError:
        return raw

    if parsed.tzinfo is None:
        parsed = parsed.replace(tzinfo=timezone.utc)
    return parsed.astimezone(timezone.utc).isoformat().replace("+00:00", "Z")


def _session_status(token_exists: bool, expires_at: str | None) -> str:
    if not token_exists:
        return "expired"
    if not expires_at:
        return "active"

    normalized = expires_at.replace("Z", "+00:00")
    try:
        parsed = datetime.fromisoformat(normalized)
    except ValueError:
        return "active"

    if parsed.tzinfo is None:
        parsed = parsed.replace(tzinfo=timezone.utc)
    return "active" if parsed.astimezone(timezone.utc) > datetime.now(timezone.utc) else "expired"


def _env_for_provider(provider_config: dict[str, Any], config_dir: str | None) -> dict[str, str]:
    env = os.environ.copy()
    if config_dir:
        for env_name in provider_config["env"]:
            env[env_name] = config_dir
    return env


def _remove_credential_files(
    provider: str,
    provider_config: dict[str, Any],
    override_config_dir: str | None = None,
) -> list[str]:
    removed: list[str] = []
    for config_dir in _candidate_config_dirs(provider_config, override_config_dir):
        files = _find_credential_files(config_dir, provider_config["credential_names"])
        for path in files:
            try:
                path.unlink()
                removed.append(str(path))
            except OSError as exc:
                logger.warning("Unable to remove %s credential file %s: %s", provider, path, exc)
    return removed
