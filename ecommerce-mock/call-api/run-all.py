"""
call-api/run-all.py
-------------------
Run every call-api script in order and print a combined final summary.

Order
  ── Happy path (2xx) ──────────────────────────────────────────────────────────
  1. get/2xx        — all normal GET endpoints
  2. put/2xx        — all normal PUT endpoints
  3. post-delete/2xx— POST to create data, DELETE to remove it

  ── Sim / error path (5xx) ────────────────────────────────────────────────────
  4. get/5xx        — bad SELECT via GET
  5. post/5xx       — bad INSERT via POST
  6. put/5xx        — bad UPDATE via PUT
  7. delete/5xx     — bad DELETE via DELETE

Each script exits 1 on failure. run-all.py exits 1 if any script failed.

Usage:
  python call-api/run-all.py
  python call-api/run-all.py --base-url http://192.168.1.10
  python call-api/run-all.py --timeout 10
  python call-api/run-all.py --only 2xx
  python call-api/run-all.py --only 5xx
"""

import argparse
import os
import subprocess
import sys
import time
from pathlib import Path

# ─── ANSI colours ─────────────────────────────────────────────────────────────
GREEN  = "\033[32m"
RED    = "\033[31m"
YELLOW = "\033[33m"
CYAN   = "\033[36m"
BOLD   = "\033[1m"
DIM    = "\033[2m"
RESET  = "\033[0m"

# ─── Script registry ──────────────────────────────────────────────────────────
# (label, relative path, group)
SCRIPTS: list[tuple[str, str, str]] = [
    ("GET  2xx — normal reads",          "get/2xx/run.py",         "2xx"),
    ("PUT  2xx — normal updates",        "put/2xx/run.py",         "2xx"),
    ("POST+DELETE 2xx — create & clean", "post-delete/2xx/run.py", "2xx"),
    ("GET  5xx — sim bad-column",        "get/5xx/run.py",         "5xx"),
    ("POST 5xx — sim bad-insert",        "post/5xx/run.py",        "5xx"),
    ("PUT  5xx — sim bad-update",        "put/5xx/run.py",         "5xx"),
    ("DEL  5xx — sim bad-delete",        "delete/5xx/run.py",      "5xx"),
]


def run_script(
    script_path: Path,
    base_url: str,
    timeout_sec: int,
) -> tuple[bool, float]:
    """Run a single script. Streams its output. Returns (passed, elapsed_sec)."""
    cmd = [sys.executable, str(script_path), "--base-url", base_url, "--timeout", str(timeout_sec)]
    t0 = time.time()
    result = subprocess.run(cmd, check=False)
    elapsed = time.time() - t0
    return result.returncode == 0, elapsed


def main() -> None:
    parser = argparse.ArgumentParser(description="Run all call-api scripts")
    parser.add_argument("--base-url", default="http://localhost",
                        help="Base URL forwarded to each script (default: http://localhost)")
    parser.add_argument("--timeout", type=int, default=10,
                        help="Per-request timeout in seconds (default: 10)")
    parser.add_argument("--only", choices=["2xx", "5xx"],
                        help="Run only the 2xx or only the 5xx scripts")
    args = parser.parse_args()

    root = Path(__file__).parent

    scripts = [
        (label, root / path, group)
        for label, path, group in SCRIPTS
        if args.only is None or group == args.only
    ]

    print(f"\n{BOLD}{CYAN}{'═' * 56}{RESET}")
    print(f"{BOLD}{CYAN}  call-api  run-all{RESET}")
    print(f"{BOLD}{CYAN}  base: {args.base_url}{RESET}")
    if args.only:
        print(f"{BOLD}{CYAN}  filter: {args.only} only{RESET}")
    print(f"{BOLD}{CYAN}{'═' * 56}{RESET}\n")

    results: list[tuple[str, bool, float]] = []
    current_group: str | None = None

    for label, path, group in scripts:
        if group != current_group:
            current_group = group
            header = "── Happy path (2xx) ──" if group == "2xx" else "── Sim / error path (5xx) ──"
            print(f"\n{CYAN}{header}{RESET}\n")

        print(f"{BOLD}▶ {label}{RESET}")
        print(f"{DIM}  {path.relative_to(root.parent)}{RESET}\n")

        passed, elapsed = run_script(path, args.base_url, args.timeout)
        results.append((label, passed, elapsed))

        status = f"{GREEN}PASS{RESET}" if passed else f"{RED}FAIL{RESET}"
        print(f"\n  {status}  {DIM}{elapsed:.1f}s{RESET}\n")

    # ── Final summary ──────────────────────────────────────────────────────────
    total  = len(results)
    passed = sum(1 for _, ok, _ in results if ok)
    failed = total - passed

    print(f"\n{BOLD}{CYAN}{'═' * 56}{RESET}")
    print(f"{BOLD}{CYAN}  Final summary{RESET}")
    print(f"{BOLD}{CYAN}{'═' * 56}{RESET}\n")

    for label, ok, elapsed in results:
        icon = f"{GREEN}✓{RESET}" if ok else f"{RED}✗{RESET}"
        print(f"  {icon}  {label}  {DIM}({elapsed:.1f}s){RESET}")

    print(f"\n  {GREEN if failed == 0 else RED}{BOLD}passed: {passed}/{total}{RESET}   {RED}failed: {failed}/{total}{RESET}")
    print(f"\n{BOLD}{CYAN}{'═' * 56}{RESET}\n")

    if failed > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
