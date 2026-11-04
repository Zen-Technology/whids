#!/usr/bin/env python3
"""Render `go test -json` output for CI.

* prints the regular human-readable test output to the job log,
* emits one GitHub error annotation per failing test (or per package for
  build failures, panics and timeouts), so failures show up on the PR and in
  the checks API without digging through logs,
* writes a pass/fail/skip table to the job summary,
* exits non-zero when anything failed.

Usage: go-test-report.py <go-test.json> [label] [go-test-exit-status]
"""

import collections
import json
import os
import sys

MAX_LINES = 60  # tail of output kept per annotation


def escape(value: str) -> str:
    return value.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")


def main() -> int:
    # Windows runners default to cp1252 for pipes; test output is UTF-8.
    for stream in (sys.stdout, sys.stderr):
        stream.reconfigure(encoding="utf-8", errors="replace")

    path = sys.argv[1]
    label = sys.argv[2] if len(sys.argv) > 2 else "go test"
    # exit status of `go test`, so a failure without a failing test is reported
    go_status = int(sys.argv[3]) if len(sys.argv) > 3 else 0

    output = collections.defaultdict(list)  # (package, test) -> lines
    status = {}  # (package, test) -> pass | fail | skip
    stray = []  # non-JSON lines (toolchain errors)

    with open(path, encoding="utf-8", errors="replace") as handle:
        for raw in handle:
            line = raw.rstrip("\n")
            if not line.startswith("{"):
                if line.strip():
                    stray.append(line)
                    print(line)
                continue
            try:
                event = json.loads(line)
            except json.JSONDecodeError:
                stray.append(line)
                continue

            action = event.get("Action")
            package = event.get("Package") or event.get("ImportPath", "")
            key = (package, event.get("Test", ""))

            if action in ("output", "build-output"):
                text = event.get("Output", "")
                sys.stdout.write(text)
                output[key].append(text.rstrip("\n"))
            elif action in ("pass", "fail", "skip"):
                status[key] = action
            elif action == "build-fail":
                status[key] = "fail"

    failed = [key for key, value in status.items() if value == "fail"]
    failed_tests = {pkg for pkg, test in failed if test}
    annotations = 0
    for package, test in sorted(failed):
        # a package fails whenever one of its tests does; only annotate the
        # package itself when no test explains it (build error, panic, timeout)
        if not test and package in failed_tests:
            continue
        lines = [entry for entry in output[(package, test)] if entry.strip()]
        if not test:
            # include build output reported under the bare import path
            lines = [entry for entry in output[(package, "")] if entry.strip()]
        message = "\n".join(lines[-MAX_LINES:]) or "failed without output"
        name = package.rsplit("/", 1)[-1] + (f".{test}" if test else "")
        print(f"::error title={escape(f'{label}: FAIL {name}')}::{escape(message)}")
        annotations += 1

    if stray and not status:
        print(f"::error title={escape(label + ': go test did not run')}::{escape(chr(10).join(stray[-MAX_LINES:]))}")
        annotations += 1
    elif go_status != 0 and annotations == 0:
        tail = stray[-MAX_LINES:] or ["no failing test was reported; see the job log"]
        print(f"::error title={escape(f'{label}: go test exited with status {go_status}')}::{escape(chr(10).join(tail))}")
        annotations += 1

    counts = collections.Counter(
        value for (package, test), value in status.items() if test
    )
    summary = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary:
        with open(summary, "a", encoding="utf-8") as handle:
            handle.write(f"### {label}\n\n")
            handle.write("| passed | failed | skipped |\n|---|---|---|\n")
            handle.write(f"| {counts['pass']} | {counts['fail']} | {counts['skip']} |\n\n")
            if failed:
                handle.write("Failed:\n\n")
                for package, test in sorted(failed):
                    if test or package not in failed_tests:
                        handle.write(f"- `{package}` {test}\n")
                handle.write("\n")

    return 1 if (failed or (stray and not status) or go_status != 0) else 0


if __name__ == "__main__":
    sys.exit(main())
