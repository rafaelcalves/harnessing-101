#!/usr/bin/env python3
"""Run an owner-only Layer B agentic CLI preflight.

The descriptor describes the selected installed tool. Product-managed start is
kept in this consumer, and authoritative cycle records must come from product
state; tool output never proves participation or completion.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import platform
import re
import shutil
import subprocess
import sys
from pathlib import Path
from typing import Any

RESULT_CODES = {"cycle_completed": 0, "owner_deferred": 19, "missing_tool": 20, "authentication_required": 21, "network_egress_refused": 22, "started_no_participation": 23, "product_surface_unavailable": 24, "unsupported_invocation": 25, "spawn_failed": 26}
HOST_TIMEOUT_SECONDS = 120.0
MAX_DIAGNOSTIC_BYTES = 16_000
KNOWN_SLOTS = {"workspace", "workspace_id", "task_id", "agent_id", "peer_id", "profile_id", "run_id", "prompt"}


def fail(result: str, message: str, evidence: dict[str, Any], report: Path) -> int:
    payload = {"result": result, "message": message, "evidence": evidence}
    print(json.dumps(payload, indent=2, sort_keys=True))
    report.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    return RESULT_CODES.get(result, 1)


def run(argv: list[str], *, cwd: Path | None = None, timeout: float = HOST_TIMEOUT_SECONDS) -> subprocess.CompletedProcess[str]:
    return subprocess.run(argv, cwd=cwd, capture_output=True, text=True, check=False, timeout=timeout)


def diagnostic_text(proc: subprocess.CompletedProcess[str]) -> str:
    return (proc.stdout + "\n" + proc.stderr)[-MAX_DIAGNOSTIC_BYTES:]


def validate_descriptor(descriptor: dict[str, Any]) -> str | None:
    if descriptor.get("schemaVersion") != 2:
        return "descriptor schemaVersion must be 2"
    for key in ("toolId", "displayName", "descriptorRevision", "deferred"):
        if key not in descriptor or (key != "deferred" and not descriptor.get(key)):
            return f"descriptor missing required field {key!r}"
    if not isinstance(descriptor["deferred"], bool):
        return "descriptor deferred must be boolean"
    evidence = descriptor.get("evidence")
    if not isinstance(evidence, dict) or not evidence.get("scenarioRef") or not evidence.get("scenarioVersion") or not evidence.get("plannedInterventions"):
        return "descriptor evidence requires scenarioRef, scenarioVersion, and plannedInterventions"
    if descriptor["deferred"]:
        if not descriptor.get("deferral", {}).get("reason") or not descriptor.get("deferral", {}).get("reference"):
            return "deferred descriptor requires deferral.reason and deferral.reference"
        if "toolSpec" in descriptor:
            return "deferred descriptor must not contain toolSpec"
        return None
    spec = descriptor.get("toolSpec")
    if not isinstance(spec, dict):
        return "non-deferred descriptor requires toolSpec"
    executable = spec.get("executable", {})
    if not isinstance(executable, dict) or not executable.get("lookup"):
        return "toolSpec.executable.lookup is required"
    probe = spec.get("versionProbe", {})
    if not isinstance(probe, dict) or not isinstance(probe.get("args"), list) or not probe["args"]:
        return "toolSpec.versionProbe.args must be nonempty"
    codes = probe.get("acceptedExitCodes")
    if not isinstance(codes, list) or not codes or len(set(codes)) != len(codes):
        return "toolSpec.versionProbe.acceptedExitCodes must be nonempty and unique"
    invocation = spec.get("invocation", {})
    if not isinstance(invocation, dict) or not isinstance(invocation.get("args"), list):
        return "toolSpec.invocation.args is required"
    context = spec.get("context", {})
    if context.get("transport") not in {"argument", "stdin", "context-file"}:
        return "toolSpec.context.transport is unsupported or missing"
    if context.get("encoding") != "utf-8" or not context.get("formatVersion"):
        return "toolSpec.context requires utf-8 encoding and formatVersion"
    allowed = set(context.get("allowedSlots", []))
    if not allowed or not allowed.issubset(KNOWN_SLOTS):
        return "toolSpec.context.allowedSlots contains unsupported or no slots"
    for element in invocation["args"]:
        if not isinstance(element, dict) or set(element) not in ({"literal"}, {"contextSlot"}):
            return "invocation args must be literal or contextSlot elements"
        if "contextSlot" in element and element["contextSlot"] not in allowed:
            return f"invocation uses unsupported context slot {element['contextSlot']!r}"
    diagnostics = spec.get("diagnostics", {})
    if not isinstance(diagnostics, dict) or not diagnostics.get("hints"):
        return "toolSpec.diagnostics.hints must be nonempty"
    for hint in diagnostics["hints"]:
        if set(hint) != {"code", "source", "matching", "pattern", "classification"}:
            return "diagnostic hints have unsupported fields"
        if hint["source"] not in {"stdout", "stderr", "exit-status"} or hint["matching"] not in {"substring", "regex"}:
            return "diagnostic hint source or matching mode is unsupported"
    if spec.get("authentication", {}).get("strategy") != "runtime-diagnostic-hints":
        return "toolSpec.authentication.strategy is unsupported"
    return None


def expand_args(elements: list[dict[str, str]], values: dict[str, str]) -> list[str]:
    return [element["literal"] if "literal" in element else values[element["contextSlot"]] for element in elements]


def match_hint(hint: dict[str, str], proc: subprocess.CompletedProcess[str]) -> bool:
    if hint["source"] == "exit-status":
        text = str(proc.returncode)
    elif hint["source"] == "stdout":
        text = proc.stdout[-MAX_DIAGNOSTIC_BYTES:]
    else:
        text = proc.stderr[-MAX_DIAGNOSTIC_BYTES:]
    if hint["matching"] == "regex":
        return re.search(hint["pattern"], text, flags=re.IGNORECASE) is not None
    return hint["pattern"].lower() in text.lower()


def main() -> int:
    parser = argparse.ArgumentParser(description="Run the real agentic CLI Layer B preflight.")
    for name in ("workspace-id", "task-id", "agent-id", "peer-id", "profile-id", "run-id", "product-revision"):
        parser.add_argument(f"--{name}", required=True)
    parser.add_argument("--descriptor", required=True, type=Path)
    parser.add_argument("--harnessing", required=True, type=Path)
    parser.add_argument("--workspace", required=True, type=Path)
    parser.add_argument("--report", required=True, type=Path)
    args = parser.parse_args()

    descriptor_path, report_path, harnessing_path, workspace_path = (args.descriptor.resolve(), args.report.resolve(), args.harnessing.resolve(), args.workspace.resolve())
    descriptor = json.loads(descriptor_path.read_text(encoding="utf-8"))
    evidence: dict[str, Any] = {"observed": {"runner": "scripts/run-agentic-cli-cycle.py", "descriptor": str(descriptor_path), "descriptorSha256": hashlib.sha256(descriptor_path.read_bytes()).hexdigest(), "os": f"{platform.system()} {platform.release()} {platform.machine()}", "productRevision": args.product_revision, "workspace": str(workspace_path), "workspace_id": args.workspace_id, "task_id": args.task_id, "agent_id": args.agent_id, "peer_id": args.peer_id, "profile_id": args.profile_id, "run_id": args.run_id}, "planned": {}, "diagnostics": {}}
    error = validate_descriptor(descriptor)
    if error:
        return fail("unsupported_invocation", error, evidence, report_path)
    evidence["planned"] = descriptor.get("evidence", {}).get("plannedInterventions", [])
    if descriptor["deferred"]:
        evidence["observed"]["deferral"] = descriptor["deferral"]
        return fail("owner_deferred", "owner deferral recorded; no executable invocation attempted", evidence, report_path)

    spec = descriptor["toolSpec"]
    lookup = spec["executable"]["lookup"]
    tool_path = shutil.which(lookup)
    if not tool_path:
        return fail("missing_tool", f"{lookup!r} is not on PATH", evidence, report_path)
    evidence["observed"]["executable"] = tool_path
    probe = spec["versionProbe"]
    try:
        version_proc = run([tool_path, *probe["args"]], timeout=min(float(probe.get("timeoutSeconds", HOST_TIMEOUT_SECONDS)), HOST_TIMEOUT_SECONDS))
    except subprocess.TimeoutExpired:
        return fail("spawn_failed", "installed tool timed out on its version probe", evidence, report_path)
    evidence["observed"]["version"] = diagnostic_text(version_proc)
    if version_proc.returncode not in probe["acceptedExitCodes"]:
        return fail("spawn_failed", "installed tool did not answer its version probe", evidence, report_path)
    try:
        help_proc = run([str(harnessing_path), "help"])
    except FileNotFoundError:
        return fail("product_surface_unavailable", f"harnessing executable not found: {harnessing_path}", evidence, report_path)
    if "start-run" not in help_proc.stdout + "\n" + help_proc.stderr and "StartRun" not in help_proc.stdout + "\n" + help_proc.stderr:
        return fail("product_surface_unavailable", "the shipped harnessing command has no managed start-run surface; a manual shell start would not be Layer B evidence", evidence, report_path)

    values = {"workspace": str(workspace_path), "workspace_id": args.workspace_id, "task_id": args.task_id, "agent_id": args.agent_id, "peer_id": args.peer_id, "profile_id": args.profile_id, "run_id": args.run_id, "prompt": json.dumps({"workspace_id": args.workspace_id, "task_id": args.task_id, "agent_id": args.agent_id, "peer_id": args.peer_id, "run_id": args.run_id}, separators=(",", ":"))}
    tool_args = expand_args(spec["invocation"]["args"], values)
    evidence["observed"]["toolArgv"] = [tool_path, *tool_args]
    product_command = [str(harnessing_path), "start-run", "-workspace", str(workspace_path), "-workspace-id", args.workspace_id, "-run", args.run_id, "-agent", args.agent_id, "-profile-id", args.profile_id, "-task", args.task_id, "-tool-executable", tool_path, "-tool-argv-json", json.dumps(tool_args)]
    evidence["observed"]["productCommand"] = product_command
    try:
        started = run(product_command, cwd=workspace_path)
    except subprocess.TimeoutExpired as error:
        evidence["diagnostics"]["tail"] = str(error)
        return fail("started_no_participation", "managed start timed out; authoritative cycle records were not observed", evidence, report_path)
    evidence["diagnostics"]["tail"] = diagnostic_text(started)
    for hint in spec["diagnostics"]["hints"]:
        if match_hint(hint, started):
            evidence["diagnostics"]["matched"] = hint
            return fail(hint["classification"], "managed start matched a diagnostic hint; this is not participation or completion proof", evidence, report_path)
    if started.returncode != 0:
        return fail("spawn_failed", "managed start returned non-zero without a typed diagnostic", evidence, report_path)
    return fail("started_no_participation", "start receipt is not authoritative cycle completion; query product records before any claim", evidence, report_path)


if __name__ == "__main__":
    sys.exit(main())
