#!/usr/bin/env python3
"""Run the owner-only Layer B agentic CLI participation check.

This is deliberately a guided runner. Human setup, blocker answers, and review
decisions remain human actions. The runner owns only discovery, preflight, the
managed-start invocation, and evidence capture.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import platform
import shutil
import subprocess
import sys
from pathlib import Path
from typing import Any


RESULT_CODES = {
    "cycle_completed": 0,
    "missing_tool": 20,
    "authentication_required": 21,
    "network_egress_refused": 22,
    "started_no_participation": 23,
    "product_surface_unavailable": 24,
    "unsupported_invocation": 25,
    "spawn_failed": 26,
}


def fail(result: str, message: str, evidence: dict[str, Any], report: Path | None) -> int:
    payload = {"result": result, "message": message, "evidence": evidence}
    print(json.dumps(payload, indent=2, sort_keys=True))
    if report:
        report.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    return RESULT_CODES.get(result, 1)


def run(
    argv: list[str],
    *,
    cwd: Path | None = None,
    env: dict[str, str] | None = None,
    timeout: float = 120.0,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(argv, cwd=cwd, env=env, capture_output=True, text=True, check=False, timeout=timeout)


def render(parts: list[str], values: dict[str, str]) -> list[str]:
    return [part.format(**values) for part in parts]


def contains_any(text: str, patterns: list[str]) -> str | None:
    lowered = text.lower()
    for pattern in patterns:
        if pattern.lower() in lowered:
            return pattern
    return None


def main() -> int:
    parser = argparse.ArgumentParser(description="Run the real agentic CLI Layer B cycle.")
    parser.add_argument("--descriptor", required=True, type=Path)
    parser.add_argument("--harnessing", required=True, type=Path)
    parser.add_argument("--workspace", required=True, type=Path)
    parser.add_argument("--workspace-id", required=True)
    parser.add_argument("--task-id", required=True)
    parser.add_argument("--agent-id", required=True)
    parser.add_argument("--peer-id", required=True)
    parser.add_argument("--profile-id", required=True)
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--product-revision", required=True)
    parser.add_argument("--timeout-seconds", type=float, default=120.0)
    parser.add_argument("--report", required=True, type=Path)
    args = parser.parse_args()

    descriptor = json.loads(args.descriptor.read_text(encoding="utf-8"))
    required = ("schemaVersion", "tool", "executable", "version", "auth", "invocation", "participation")
    missing = [key for key in required if key not in descriptor]
    if missing:
        return fail("unsupported_invocation", "descriptor missing required fields", {"missing": missing}, args.report)

    executable = descriptor["executable"]
    values = {
        "workspace": str(args.workspace),
        "workspace_id": args.workspace_id,
        "task_id": args.task_id,
        "agent_id": args.agent_id,
        "peer_id": args.peer_id,
        "profile_id": args.profile_id,
        "run_id": args.run_id,
    }
    evidence: dict[str, Any] = {
        "tool": descriptor["tool"],
        "descriptor": str(args.descriptor),
        "descriptorSha256": hashlib.sha256(args.descriptor.read_bytes()).hexdigest(),
        "runner": "scripts/run-agentic-cli-cycle.py",
        "os": f"{platform.system()} {platform.release()} {platform.machine()}",
        "workspace": str(args.workspace),
        "workspace_id": args.workspace_id,
        "task_id": args.task_id,
        "agent_id": args.agent_id,
        "peer_id": args.peer_id,
        "profile_id": args.profile_id,
        "run_id": args.run_id,
        "human_interventions": descriptor.get("humanInterventions", []),
        "friction": descriptor.get("friction", []),
    }

    tool_path = shutil.which(executable)
    if not tool_path:
        return fail("missing_tool", f"{executable!r} is not on PATH", evidence, args.report)
    evidence["executable"] = tool_path

    version_spec = descriptor["version"]
    try:
        version_proc = run([tool_path, *render(version_spec["args"], values)], timeout=args.timeout_seconds)
    except subprocess.TimeoutExpired:
        return fail("spawn_failed", "installed tool timed out on its version command", evidence, args.report)
    version_text = (version_proc.stdout + "\n" + version_proc.stderr).strip()
    evidence["version"] = version_text
    if version_proc.returncode not in version_spec.get("successExitCodes", [0]):
        return fail("spawn_failed", "installed tool did not answer its version command", evidence, args.report)

    help_proc = run([str(args.harnessing), "help"])
    help_text = help_proc.stdout + "\n" + help_proc.stderr
    evidence["productRevision"] = args.product_revision
    evidence["commit"] = args.product_revision
    if "start-run" not in help_text and "StartRun" not in help_text:
        return fail(
            "product_surface_unavailable",
            "the shipped harnessing command has no managed start-run surface; a manual shell start would not be Layer B evidence",
            evidence,
            args.report,
        )

    start_spec = descriptor["invocation"].get("managedStart")
    if not start_spec:
        return fail("unsupported_invocation", "descriptor has no managed-start invocation", evidence, args.report)
    command = [str(args.harnessing), *render(start_spec["args"], values)]
    try:
        start_proc = run(command, cwd=args.workspace, timeout=args.timeout_seconds)
    except subprocess.TimeoutExpired as error:
        evidence["startOutput"] = (error.stdout or "") + "\n" + (error.stderr or "")
        return fail("started_no_participation", "managed start timed out before participation evidence", evidence, args.report)
    combined = start_proc.stdout + "\n" + start_proc.stderr
    evidence["startCommand"] = command
    evidence["startOutput"] = combined

    auth_pattern = contains_any(combined, descriptor["auth"].get("requiredPatterns", []))
    if auth_pattern:
        evidence["matchedPattern"] = auth_pattern
        return fail("authentication_required", "the installed tool requires authentication; authenticate it outside this runner and retry", evidence, args.report)
    network_pattern = contains_any(combined, descriptor["auth"].get("networkRefusedPatterns", []))
    if network_pattern:
        evidence["matchedPattern"] = network_pattern
        return fail("network_egress_refused", "provider egress was refused; the runner did not retry or work around it", evidence, args.report)
    if start_proc.returncode != 0:
        return fail("spawn_failed", "managed start returned non-zero", evidence, args.report)

    signals = descriptor["participation"].get("requiredSignals", [])
    missing_signals = [signal for signal in signals if signal not in combined]
    if missing_signals:
        evidence["missingSignals"] = missing_signals
        return fail("started_no_participation", "the tool started but did not produce the required participation evidence", evidence, args.report)

    completion_signals = descriptor["participation"].get("completionSignals", [])
    missing_completion = [signal for signal in completion_signals if signal not in combined]
    if missing_completion:
        evidence["missingCompletionSignals"] = missing_completion
        return fail("started_no_participation", "the full bounded cycle has not been evidenced; do not claim completion", evidence, args.report)

    evidence["humanInterventions"] = [
        "complete initial setup and assignment",
        "answer the recorded blocker",
        "perform the one rejection and final acceptance",
    ]
    return fail("cycle_completed", "managed start produced the declared full-cycle evidence", evidence, args.report)


if __name__ == "__main__":
    sys.exit(main())
