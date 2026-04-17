"""
call-api/get/5xx/run.py
-----------------------
Call every GET simulation endpoint across all services and assert 5xx responses.

Each endpoint intentionally runs a bad query against its database:
  api-product  → SELECT not_existed FROM products  (PostgreSQL SQLSTATE 42703)
  api-customer → SELECT not_existed FROM customers (PostgreSQL SQLSTATE 42703)
  api-cart     → aggregate with $notAStage         (MongoServerError)
  api-order    → SELECT not_existed FROM orders    (PostgreSQL SQLSTATE 42703)
  api-payment  → SELECT not_existed FROM Payments  (MySQL Error 1054)

A test PASSES when the response status is 5xx.
A test FAILS when the response is 2xx/4xx or the request errors out entirely.

Usage:
  python call-api/get/5xx/run.py
  python call-api/get/5xx/run.py --base-url http://192.168.1.10
"""

import argparse
import json
import sys
import time
import urllib.request
import urllib.error
from typing import Optional


def build_cases(base: str) -> list[tuple[str, str, str]]:
    """Returns list of (service, label, url)."""
    return [
        ("api-product",  "bad-column (PostgreSQL)",  f"{base}:8081/products/sim/bad-column"),
        ("api-customer", "bad-column (PostgreSQL)",  f"{base}:8082/sim/bad-column"),
        ("api-cart",     "bad-query  (MongoDB)",     f"{base}:8083/cart/sim/bad-query"),
        ("api-order",    "bad-column (PostgreSQL)",  f"{base}:8084/orders/sim/bad-column"),
        ("api-payment",  "bad-column (MySQL)",       f"{base}:8085/sim/bad-column"),
    ]


# ─── ANSI colours ─────────────────────────────────────────────────────────────
GREEN  = "\033[32m"
RED    = "\033[31m"
YELLOW = "\033[33m"
CYAN   = "\033[36m"
DIM    = "\033[2m"
RESET  = "\033[0m"


def call(url: str, timeout: int = 5) -> tuple[Optional[int], Optional[dict], Optional[str]]:
    """Returns (status_code, body_dict, error_str)."""
    try:
        req = urllib.request.Request(url, headers={"Accept": "application/json"})
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8", errors="replace")
            try:
                body = json.loads(raw)
            except json.JSONDecodeError:
                body = {"raw": raw[:120]}
            return resp.status, body, None
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", errors="replace")
        try:
            body = json.loads(raw)
        except json.JSONDecodeError:
            body = None
        return e.code, body, None
    except Exception as e:
        return None, None, str(e)


def main() -> None:
    parser = argparse.ArgumentParser(description="Call all GET 5xx simulation endpoints")
    parser.add_argument("--base-url", default="http://localhost", help="Base URL (default: http://localhost)")
    parser.add_argument("--timeout",  type=int, default=5, help="Request timeout in seconds (default: 5)")
    args = parser.parse_args()

    base = args.base_url.rstrip("/")
    cases = build_cases(base)

    passed = 0
    failed = 0

    print(f"\n{CYAN}GET 5xx sim — {base}{RESET}\n")

    for service, label, url in cases:
        status, body, err = call(url, args.timeout)

        is_5xx = status is not None and 500 <= status < 600

        path = url.replace(base, "")

        if err:
            # Connection-level failure — service may be down
            failed += 1
            print(f"  {RED}  ERR FAIL{RESET}  {DIM}GET {path}{RESET}")
            print(f"        {YELLOW}{err}{RESET}")
            continue

        if is_5xx:
            passed += 1
            tag = f"{GREEN}  {status} OK{RESET}"
        else:
            failed += 1
            tag = f"{RED}  {status} FAIL{RESET}"

        print(f"  {tag}  {DIM}GET {path}{RESET}  {DIM}[{service} — {label}]{RESET}")

        # Show the detail field from the response body so the DB error is visible
        if body and isinstance(body, dict):
            detail = body.get("detail") or body.get("error") or ""
            if detail:
                print(f"        {YELLOW}{detail[:120]}{RESET}")

        time.sleep(0.05)

    # ── Summary ───────────────────────────────────────────────────────────────
    total = passed + failed
    print(f"\n{'─' * 50}")
    print(f"  {GREEN}passed: {passed}/{total}{RESET}   {RED}failed: {failed}/{total}{RESET}")
    print(f"{'─' * 50}\n")

    if failed > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
