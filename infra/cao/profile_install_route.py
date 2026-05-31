"""Profile install endpoint — bridges AgentVerse deploy to CAO file-based profiles."""

import logging
from pathlib import Path

import frontmatter
from fastapi import APIRouter, Body, HTTPException, status

from cli_agent_orchestrator.constants import LOCAL_AGENT_STORE_DIR

logger = logging.getLogger(__name__)

profile_install_router = APIRouter(tags=["agents"])


@profile_install_router.post(
    "/agents/profiles/install",
    status_code=status.HTTP_201_CREATED,
)
async def install_profile(
    body: str = Body(..., media_type="text/markdown; charset=utf-8"),
):
    """Write a markdown agent profile to the local agent store.

    The body must be a frontmatter-enabled markdown document with at least
    a ``name`` field in the YAML header.  The profile is saved as
    ``<LOCAL_AGENT_STORE_DIR>/<name>.md`` and returned as JSON.
    """
    try:
        parsed = frontmatter.loads(body)
    except Exception as exc:
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
            detail=f"Invalid frontmatter: {exc}",
        ) from exc

    name = parsed.metadata.get("name")
    if not name:
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
            detail="Profile markdown must include a 'name' field in YAML frontmatter.",
        )

    # Sanitise name
    if "/" in name or "\\" in name or ".." in name:
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
            detail=f"Invalid profile name '{name}': must not contain '/', '\\\\', or '..'",
        )

    store_dir = Path(LOCAL_AGENT_STORE_DIR)
    store_dir.mkdir(parents=True, exist_ok=True)

    profile_path = store_dir / f"{name}.md"
    profile_path.write_text(body, encoding="utf-8")
    logger.info("Installed agent profile '%s' → %s", name, profile_path)

    return {
        "name": name,
        "description": parsed.metadata.get("description", ""),
        "role": parsed.metadata.get("role", ""),
        "provider": parsed.metadata.get("provider", ""),
        "model": parsed.metadata.get("model", ""),
        "source": "local",
        "path": str(profile_path),
    }
