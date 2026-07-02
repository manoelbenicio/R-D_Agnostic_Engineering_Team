#!/usr/bin/env python3
"""Generate Grafana dashboard JSON from a small YAML component spec."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

try:
    import yaml
except ImportError:  # pragma: no cover - exercised only on missing dependency.
    yaml = None


CATALOG_METRICS = {
    "rotation_total",
    "rotation_duration_seconds",
    "all_accounts_exhausted",
    "accounts_available",
    "account_status",
    "account_tokens_used",
    "account_window_seconds_remaining",
    "exhaustion_detected_total",
    "credential_restore_total",
    "cred_env_injection_total",
    "credential_prepare_seconds",
}

PANEL_TYPES = {"timeseries", "stat", "gauge", "table"}
NAME_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9_-]*$")


class SpecError(ValueError):
    """Raised for invalid component specs."""


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate Grafana dashboard JSON from a YAML component spec."
    )
    parser.add_argument("spec", type=Path, help="YAML component spec to read")
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=Path("deploy/observability/grafana/dashboards"),
        help="Grafana dashboards directory (default: %(default)s)",
    )
    return parser.parse_args()


def load_yaml(path: Path) -> dict[str, Any]:
    if yaml is None:
        raise SpecError("PyYAML is required. Install it with: pip install pyyaml")

    try:
        with path.open("r", encoding="utf-8") as fh:
            loaded = yaml.safe_load(fh)
    except OSError as exc:
        raise SpecError(f"cannot read {path}: {exc}") from exc
    except yaml.YAMLError as exc:
        raise SpecError(f"invalid YAML in {path}: {exc}") from exc

    if loaded is None:
        raise SpecError(f"{path} is empty")
    if not isinstance(loaded, dict):
        raise SpecError("spec root must be a mapping with a 'components' list")
    return loaded


def require_string(value: Any, path: str) -> str:
    if not isinstance(value, str) or not value.strip():
        raise SpecError(f"{path} must be a non-empty string")
    return value.strip()


def validate_spec(spec: dict[str, Any]) -> list[dict[str, Any]]:
    components = spec.get("components")
    if not isinstance(components, list) or not components:
        raise SpecError("components must be a non-empty list")

    names: set[str] = set()
    validated: list[dict[str, Any]] = []

    for component_index, component in enumerate(components):
        component_path = f"components[{component_index}]"
        if not isinstance(component, dict):
            raise SpecError(f"{component_path} must be a mapping")

        name = require_string(component.get("name"), f"{component_path}.name")
        if not NAME_RE.fullmatch(name):
            raise SpecError(
                f"{component_path}.name must match {NAME_RE.pattern}; got {name!r}"
            )
        if name in names:
            raise SpecError(f"duplicate component name: {name}")
        names.add(name)

        title = require_string(component.get("title"), f"{component_path}.title")
        panels = component.get("panels")
        if not isinstance(panels, list) or not panels:
            raise SpecError(f"{component_path}.panels must be a non-empty list")

        validated_panels: list[dict[str, str]] = []
        for panel_index, panel in enumerate(panels):
            panel_path = f"{component_path}.panels[{panel_index}]"
            if not isinstance(panel, dict):
                raise SpecError(f"{panel_path} must be a mapping")

            panel_title = require_string(panel.get("title"), f"{panel_path}.title")
            metric = require_string(panel.get("metric"), f"{panel_path}.metric")
            if metric not in CATALOG_METRICS:
                allowed = ", ".join(sorted(CATALOG_METRICS))
                raise SpecError(
                    f"{panel_path}.metric unknown metric {metric!r}; "
                    f"allowed metrics: {allowed}"
                )

            panel_type = require_string(panel.get("type"), f"{panel_path}.type")
            if panel_type not in PANEL_TYPES:
                allowed = ", ".join(sorted(PANEL_TYPES))
                raise SpecError(
                    f"{panel_path}.type must be one of: {allowed}; got {panel_type!r}"
                )

            query = require_string(panel.get("query"), f"{panel_path}.query")
            if metric not in query:
                raise SpecError(
                    f"{panel_path}.query must reference declared metric {metric!r}"
                )

            validated_panels.append(
                {
                    "metric": metric,
                    "query": query,
                    "title": panel_title,
                    "type": panel_type,
                }
            )

        validated.append({"name": name, "panels": validated_panels, "title": title})

    return validated


def grid_position(index: int) -> dict[str, int]:
    width = 12
    height = 8
    return {"h": height, "w": width, "x": (index % 2) * width, "y": (index // 2) * height}


def panel_options(panel_type: str) -> dict[str, Any]:
    if panel_type == "timeseries":
        return {
            "legend": {
                "displayMode": "table",
                "placement": "bottom",
                "showLegend": True,
            },
            "tooltip": {"mode": "multi", "sort": "none"},
        }
    if panel_type == "table":
        return {"showHeader": True}
    if panel_type == "gauge":
        return {
            "orientation": "auto",
            "reduceOptions": {
                "calcs": ["lastNotNull"],
                "fields": "",
                "values": False,
            },
            "showThresholdLabels": False,
            "showThresholdMarkers": True,
        }
    return {
        "colorMode": "value",
        "graphMode": "none",
        "justifyMode": "auto",
        "orientation": "auto",
        "reduceOptions": {
            "calcs": ["lastNotNull"],
            "fields": "",
            "values": False,
        },
        "textMode": "auto",
    }


def panel_unit(metric: str) -> str:
    if metric.endswith("_seconds") or metric.endswith("_seconds_remaining"):
        return "s"
    if metric.endswith("_total"):
        return "ops"
    return "short"


def build_panel(panel: dict[str, str], index: int) -> dict[str, Any]:
    panel_type = panel["type"]
    target: dict[str, Any] = {
        "expr": panel["query"],
        "legendFormat": "",
        "refId": "A",
    }
    if panel_type == "table":
        target["format"] = "table"
        target["instant"] = True

    return {
        "datasource": {"type": "prometheus", "uid": "prometheus"},
        "fieldConfig": {
            "defaults": {"mappings": [], "unit": panel_unit(panel["metric"])},
            "overrides": [],
        },
        "gridPos": grid_position(index),
        "id": index + 1,
        "options": panel_options(panel_type),
        "targets": [target],
        "title": panel["title"],
        "type": panel_type,
    }


def build_dashboard(component: dict[str, Any]) -> dict[str, Any]:
    return {
        "annotations": {"list": []},
        "editable": True,
        "fiscalYearStartMonth": 0,
        "graphTooltip": 0,
        "id": None,
        "links": [],
        "liveNow": False,
        "panels": [
            build_panel(panel, index)
            for index, panel in enumerate(component["panels"])
        ],
        "refresh": "30s",
        "schemaVersion": 39,
        "tags": ["credential-isolation", "observability", "generated"],
        "templating": {"list": []},
        "time": {"from": "now-6h", "to": "now"},
        "timezone": "browser",
        "title": component["title"],
        "uid": component["name"],
        "version": 1,
        "weekStart": "",
    }


def write_dashboards(components: list[dict[str, Any]], output_dir: Path) -> list[Path]:
    try:
        output_dir.mkdir(parents=True, exist_ok=True)
    except OSError as exc:
        raise SpecError(f"cannot create output directory {output_dir}: {exc}") from exc

    written: list[Path] = []
    for component in components:
        output_path = output_dir / f"{component['name']}.generated.json"
        dashboard = build_dashboard(component)
        encoded = json.dumps(dashboard, indent=2, sort_keys=True) + "\n"
        try:
            output_path.write_text(encoded, encoding="utf-8")
        except OSError as exc:
            raise SpecError(f"cannot write {output_path}: {exc}") from exc
        written.append(output_path)
    return written


def main() -> int:
    args = parse_args()
    try:
        spec = load_yaml(args.spec)
        components = validate_spec(spec)
        written = write_dashboards(components, args.output_dir)
    except SpecError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1

    for path in written:
        print(f"wrote {path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
