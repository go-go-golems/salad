#!/usr/bin/env python3
"""
Ticket: 003-ANALYZERS — analyzer settings templates

Context
-------
Saleae Logic 2 .sal session files contain a `meta.json` that includes the analyzers configured in the UI,
including their setting keys, selected values, and (for dropdowns) the full list of options.

The Saleae gRPC Automation API does NOT provide a method to read back analyzer settings from a running
session, so this script supports a "forensics" workflow:

  UI → save session (.sal) → unzip → meta.json → extract settings → commit a reusable YAML template.

What it does
------------
- Lists analyzers found in a `meta.json` (type/name/nodeId).
- Extracts the settings for a single analyzer into:
  - YAML suitable for `salad analyzer add --settings-yaml ...` (our parser accepts either top-level map
    or a `settings:` wrapper), or
  - JSON suitable for `--settings-json ...`

Important notes
---------------
- Settings keys in templates must match the Logic 2 UI labels exactly.
- For dropdowns, Saleae's Automation docs expect the UI-visible *dropdown text* (string). The `meta.json`
  also contains numeric "value" codes; this script defaults to emitting dropdownText strings.
- Fields like `showInDataTable` / `streamToTerminal` appear in `meta.json` but are NOT part of the gRPC
  analyzer settings map; they cannot be applied via `AddAnalyzerRequest.settings`.

Usage
-----
List analyzers:
  ./02-extract-analyzer-settings-from-meta-json.py --meta /tmp/meta.json --list

Extract a specific analyzer (by nodeId) as YAML:
  ./02-extract-analyzer-settings-from-meta-json.py --meta /tmp/meta.json --node-id 10028 --format yaml

Extract as JSON:
  ./02-extract-analyzer-settings-from-meta-json.py --meta /tmp/meta.json --node-id 10028 --format json

Extract using numeric dropdown values (usually NOT what the API wants):
  ./02-extract-analyzer-settings-from-meta-json.py --meta /tmp/meta.json --node-id 10028 --format yaml --dropdown numeric
"""

from __future__ import annotations

import argparse
import json
import sys
from typing import Any, Dict, Iterable, List, Optional, Tuple


def _eprint(*args: object) -> None:
    print(*args, file=sys.stderr)


def _load_json(path: str) -> Any:
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def _get_analyzers(doc: Any) -> List[Dict[str, Any]]:
    if not isinstance(doc, dict):
        raise ValueError(f"meta.json root must be an object, got {type(doc).__name__}")
    data = doc.get("data")
    if not isinstance(data, dict):
        return []
    analyzers = data.get("analyzers")
    if not isinstance(analyzers, list):
        return []
    out: List[Dict[str, Any]] = []
    for a in analyzers:
        if isinstance(a, dict):
            out.append(a)
    return out


def _fmt_analyzer_row(a: Dict[str, Any]) -> str:
    node_id = a.get("nodeId")
    a_type = a.get("type")
    name = a.get("name")
    return f"nodeId={node_id} type={a_type!r} name={name!r}"


def _find_analyzer(analyzers: List[Dict[str, Any]], node_id: int) -> Dict[str, Any]:
    for a in analyzers:
        if a.get("nodeId") == node_id:
            return a
    raise KeyError(f"analyzer nodeId {node_id} not found")


def _extract_setting_title_and_value(
    s: Dict[str, Any],
    dropdown_mode: str,
) -> Tuple[str, Any]:
    """
    Convert a single meta.json analyzer setting entry into a key/value pair.

    meta.json shape (observed):
      {"title": "...", "setting": {"type": "Channel", "value": 0}}
      {"title": "...", "setting": {"type": "NumberList", "options": [{"dropdownText": "...", "value": 0}, ...], "value": 0}}
    """
    title = s.get("title")
    if not isinstance(title, str) or not title.strip():
        raise ValueError("setting entry missing non-empty 'title'")
    title = title.strip()

    setting = s.get("setting")
    if not isinstance(setting, dict):
        raise ValueError(f"setting[{title!r}] missing 'setting' object")

    stype = setting.get("type")
    if not isinstance(stype, str):
        raise ValueError(f"setting[{title!r}] missing 'type'")

    if stype == "Channel":
        # expected int channel index
        return title, setting.get("value")

    if stype == "NumberList":
        current = setting.get("value")
        if dropdown_mode == "numeric":
            return title, current
        # default: map numeric -> dropdownText string
        options = setting.get("options")
        if isinstance(options, list):
            for opt in options:
                if not isinstance(opt, dict):
                    continue
                if opt.get("value") == current and isinstance(opt.get("dropdownText"), str):
                    return title, opt["dropdownText"]
        # fallback: still return numeric (but warn)
        _eprint(f"warning: no dropdownText found for {title!r} value={current!r}; emitting numeric")
        return title, current

    # unknown types: keep for debugging
    raise ValueError(f"unsupported setting type {stype!r} for title {title!r}")


def _extract_settings_map(analyzer: Dict[str, Any], dropdown_mode: str) -> Dict[str, Any]:
    raw_settings = analyzer.get("settings")
    if not isinstance(raw_settings, list):
        return {}

    out: Dict[str, Any] = {}
    for s in raw_settings:
        if not isinstance(s, dict):
            continue
        k, v = _extract_setting_title_and_value(s, dropdown_mode=dropdown_mode)
        out[k] = v
    return out


def _yaml_quote_string(s: str) -> str:
    # Use JSON string escaping for safety; YAML accepts JSON-style quoted strings.
    return json.dumps(s, ensure_ascii=False)


def _yaml_scalar(v: Any) -> str:
    if isinstance(v, str):
        return _yaml_quote_string(v)
    if isinstance(v, bool):
        return "true" if v else "false"
    if isinstance(v, int):
        return str(v)
    if isinstance(v, float):
        # keep plain float formatting
        return repr(v)
    if v is None:
        # meta.json shouldn't have null values for settings; still handle for safety
        return "null"
    # unknown scalars: serialize as JSON string
    return _yaml_quote_string(str(v))


def _emit_yaml_settings(settings: Dict[str, Any], wrapper: bool) -> str:
    lines: List[str] = []
    if wrapper:
        lines.append("settings:")
        indent = "  "
    else:
        indent = ""
    for k in sorted(settings.keys()):
        v = settings[k]
        lines.append(f"{indent}{k}: {_yaml_scalar(v)}")
    if not lines:
        return "settings: {}\n" if wrapper else "{}\n"
    return "\n".join(lines) + "\n"


def main(argv: Optional[List[str]] = None) -> int:
    p = argparse.ArgumentParser(description="Extract Saleae analyzer settings from meta.json into YAML/JSON templates.")
    p.add_argument("--meta", required=True, help="Path to meta.json extracted from a .sal session")
    p.add_argument("--list", action="store_true", help="List analyzers found in meta.json and exit")
    p.add_argument("--node-id", type=int, help="Analyzer nodeId to extract (see --list)")
    p.add_argument("--format", choices=["yaml", "json"], default="yaml", help="Output format")
    p.add_argument("--wrapper", choices=["settings", "none"], default="settings", help="Emit with `settings:` wrapper (recommended)")
    p.add_argument(
        "--dropdown",
        choices=["text", "numeric"],
        default="text",
        help="How to emit dropdown values: UI dropdown text (default) or numeric codes",
    )
    args = p.parse_args(argv)

    doc = _load_json(args.meta)
    analyzers = _get_analyzers(doc)

    if args.list:
        for a in analyzers:
            print(_fmt_analyzer_row(a))
        return 0

    if args.node_id is None:
        p.error("--node-id is required unless --list is provided")

    analyzer = _find_analyzer(analyzers, args.node_id)
    settings = _extract_settings_map(analyzer, dropdown_mode=args.dropdown)

    # Helpful header comments (YAML only)
    if args.format == "yaml":
        _eprint(f"# extracted from {args.meta}")
        _eprint(f"# {_fmt_analyzer_row(analyzer)}")
        _eprint("# NOTE: showInDataTable/streamToTerminal are not part of gRPC analyzer settings.")

    wrapper = args.wrapper == "settings"
    if args.format == "json":
        payload = {"settings": settings} if wrapper else settings
        print(json.dumps(payload, indent=2, ensure_ascii=False, sort_keys=True))
        return 0

    # yaml
    sys.stdout.write(_emit_yaml_settings(settings, wrapper=wrapper))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())


