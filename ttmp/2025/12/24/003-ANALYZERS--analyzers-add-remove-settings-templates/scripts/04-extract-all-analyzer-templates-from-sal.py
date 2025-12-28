#!/usr/bin/env python3
"""
Ticket: 003-ANALYZERS — bulk template generation from Logic 2 .sal sessions

Context
-------
We want to generate reusable analyzer settings templates from a Logic 2 session file (`.sal`).
The `.sal` contains a `meta.json` with all analyzers and their settings, including dropdown option
strings (dropdownText). This allows us to avoid guessing UI-visible keys/values.

This script produces one YAML template per analyzer and writes them into an existing output directory.
It does NOT create directories (by design for this repo/workflow).

What it generates
-----------------
- YAML files with a `settings:` block that `salad analyzer add --settings-yaml ...` can consume.
- Dropdown selections are emitted as UI-visible strings (dropdownText) by default.

Usage
-----
  ./04-extract-all-analyzer-templates-from-sal.py \
    --sal "/tmp/Session 6.sal" \
    --out-dir "/home/manuel/workspaces/2025-12-27/salad-pass/salad/configs/analyzers" \
    --prefix "session6"

Optional:
  --dropdown numeric   # emit numeric codes instead of dropdownText (usually NOT desired)
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
import zipfile
from dataclasses import dataclass
from typing import Any, Dict, Iterable, List, Optional, Tuple


def _eprint(*args: object) -> None:
    print(*args, file=sys.stderr)


def _slugify(s: str, max_len: int = 80) -> str:
    s = s.strip().lower()
    s = s.replace("&", "and")
    s = re.sub(r"[^a-z0-9]+", "-", s)
    s = re.sub(r"-{2,}", "-", s).strip("-")
    if not s:
        s = "unnamed"
    if len(s) > max_len:
        s = s[:max_len].rstrip("-")
    return s


def _read_meta_from_sal(sal_path: str) -> Any:
    with zipfile.ZipFile(sal_path) as z:
        with z.open("meta.json") as f:
            return json.load(f)


def _yaml_quote_string(s: str) -> str:
    # YAML accepts JSON-style quoted strings; json.dumps handles escaping safely.
    return json.dumps(s, ensure_ascii=False)


def _yaml_scalar(v: Any) -> str:
    if isinstance(v, str):
        return _yaml_quote_string(v)
    if isinstance(v, bool):
        return "true" if v else "false"
    if isinstance(v, int):
        return str(v)
    if isinstance(v, float):
        return repr(v)
    if v is None:
        return "null"
    return _yaml_quote_string(str(v))


def _emit_yaml_settings(settings: Dict[str, Any]) -> str:
    if not settings:
        return "settings: {}\n"
    lines: List[str] = ["settings:"]
    for k in sorted(settings.keys()):
        lines.append(f"  {k}: {_yaml_scalar(settings[k])}")
    return "\n".join(lines) + "\n"


def _as_int(v: Any) -> Optional[int]:
    # meta.json numbers are usually float64 via json; normalize to int when integral.
    if isinstance(v, bool):
        return None
    if isinstance(v, int):
        return v
    if isinstance(v, float) and v.is_integer():
        return int(v)
    return None


def _values_equal(a: Any, b: Any) -> bool:
    ai = _as_int(a)
    bi = _as_int(b)
    if ai is not None and bi is not None:
        return ai == bi
    return str(a) == str(b)


def _extract_one_setting(row: Dict[str, Any], dropdown_mode: str) -> Optional[Tuple[str, Any]]:
    title = row.get("title")
    if not isinstance(title, str) or not title.strip():
        # Some analyzers include non-setting UI rows (group headers/separators) that don't have titles.
        # Those cannot be represented in the gRPC settings map, so we skip them.
        return None
    key = title.strip()

    setting = row.get("setting")
    if not isinstance(setting, dict):
        raise ValueError(f"setting[{key!r}] missing setting object")

    stype = setting.get("type")
    if not isinstance(stype, str):
        raise ValueError(f"setting[{key!r}] missing type")

    if stype == "Channel":
        # channel index
        v = setting.get("value")
        iv = _as_int(v)
        if iv is None:
            raise ValueError(f"Channel setting[{key!r}] has non-integer value {v!r}")
        return key, iv

    if stype == "NumberList":
        cur = setting.get("value")
        if dropdown_mode == "numeric":
            iv = _as_int(cur)
            if iv is None:
                raise ValueError(f"NumberList setting[{key!r}] has non-integer current value {cur!r}")
            return key, iv

        options = setting.get("options")
        if isinstance(options, list):
            for opt in options:
                if not isinstance(opt, dict):
                    continue
                if _values_equal(opt.get("value"), cur) and isinstance(opt.get("dropdownText"), str):
                    return key, opt["dropdownText"]

        # fallback to numeric if no option string is found
        iv = _as_int(cur)
        if iv is not None:
            _eprint(f"warning: NumberList {key!r} has no dropdownText match; emitting numeric {iv}")
            return key, iv
        raise ValueError(f"NumberList setting[{key!r}] could not be resolved (cur={cur!r})")

    # Best-effort fallback for other UI setting types.
    # Many will have a scalar `value` we can represent as YAML scalar.
    v = setting.get("value")
    if isinstance(v, (str, bool, int, float)) or v is None:
        # Normalize integral floats to int for stability
        iv = _as_int(v)
        if iv is not None:
            return key, iv
        return key, v

    raise ValueError(f"unsupported setting type {stype!r} for key {key!r} (value type {type(v).__name__})")


def _extract_settings_map(analyzer: Dict[str, Any], dropdown_mode: str) -> Dict[str, Any]:
    raw = analyzer.get("settings")
    if not isinstance(raw, list):
        return {}
    out: Dict[str, Any] = {}
    for row in raw:
        if not isinstance(row, dict):
            continue
        try:
            kv = _extract_one_setting(row, dropdown_mode=dropdown_mode)
            if kv is None:
                _eprint("warning: skipping meta.json setting row with empty title (likely UI separator/header)")
                continue
            k, v = kv
            out[k] = v
        except Exception as e:
            _eprint(f"warning: skipping unhandled setting row ({e})")
    return out


@dataclass(frozen=True)
class AnalyzerRef:
    node_id: int
    a_type: str
    name: str
    settings: Dict[str, Any]


def _collect_analyzers(meta: Any, dropdown_mode: str) -> List[AnalyzerRef]:
    if not isinstance(meta, dict):
        raise ValueError(f"meta root must be object, got {type(meta).__name__}")
    data = meta.get("data")
    if not isinstance(data, dict):
        return []
    analyzers = data.get("analyzers")
    out: List[AnalyzerRef] = []
    if isinstance(analyzers, list):
        for a in analyzers:
            if not isinstance(a, dict):
                continue
            node_id_raw = a.get("nodeId")
            node_id = _as_int(node_id_raw)
            if node_id is None:
                continue
            a_type = a.get("type")
            name = a.get("name")
            if not isinstance(a_type, str) or not isinstance(name, str):
                continue
            settings = _extract_settings_map(a, dropdown_mode=dropdown_mode)
            out.append(AnalyzerRef(node_id=node_id, a_type=a_type, name=name, settings=settings))
    return out


def _session_name(meta: Any, fallback: str) -> str:
    if isinstance(meta, dict):
        data = meta.get("data")
        if isinstance(data, dict) and isinstance(data.get("name"), str) and data["name"].strip():
            return data["name"].strip()
    return fallback


def main(argv: Optional[List[str]] = None) -> int:
    p = argparse.ArgumentParser(description="Generate YAML analyzer templates from a Logic 2 .sal session.")
    p.add_argument("--sal", required=True, help="Path to .sal session file")
    p.add_argument("--out-dir", required=True, help="Existing output directory to write YAML templates into")
    p.add_argument("--prefix", default="", help="Filename prefix (e.g. session6). If empty, derived from session name.")
    p.add_argument("--dropdown", choices=["text", "numeric"], default="text", help="Emit dropdown selections as text or numeric codes")
    args = p.parse_args(argv)

    if not os.path.isfile(args.sal):
        _eprint(f"error: --sal does not exist: {args.sal}")
        return 2
    if not os.path.isdir(args.out_dir):
        _eprint(f"error: --out-dir is not a directory (and will not be created): {args.out_dir}")
        return 2

    meta = _read_meta_from_sal(args.sal)
    session = _session_name(meta, fallback=os.path.basename(args.sal))

    prefix = args.prefix.strip()
    if not prefix:
        prefix = _slugify(session, max_len=40)

    analyzers = _collect_analyzers(meta, dropdown_mode=args.dropdown)
    if not analyzers:
        _eprint("no analyzers found in meta.json")
        return 1

    written = 0
    for a in analyzers:
        # Unique filename to avoid collisions: prefix + type + name slug + nodeId
        fn = f"{prefix}-{_slugify(a.a_type, 20)}-{_slugify(a.name, 60)}-nodeid-{a.node_id}.yaml"
        out_path = os.path.join(args.out_dir, fn)

        header = "\n".join(
            [
                "#",
                f"# Generated from: {args.sal} (meta.json)",
                f"# Session: {session}",
                f"# Analyzer: nodeId={a.node_id} type={a.a_type!r} name={a.name!r}",
                "#",
                "# Notes:",
                "# - Keys/strings must match Logic 2 UI labels/options exactly.",
                "# - Dropdowns are emitted as UI-visible strings (dropdownText) by default.",
                "# - Intended usage: `salad analyzer add --settings-yaml <this-file>`",
                "#",
                "",
            ]
        )

        body = _emit_yaml_settings(a.settings)
        with open(out_path, "w", encoding="utf-8") as f:
            f.write(header)
            f.write(body)
        written += 1

    print(f"wrote {written} templates to {args.out_dir} (prefix={prefix!r})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())


