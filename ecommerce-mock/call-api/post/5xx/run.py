"""
call-api/post/5xx/run.py
------------------------
Call every POST simulation endpoint across all services and assert 5xx responses.

Each endpoint intentionally runs a bad INSERT against its database:
  api-product  → INSERT INTO products (not_existed)  (PostgreSQL SQLSTATE 42703)
  api-customer → INSERT INTO customers (not_existed) (PostgreSQL SQLSTATE 42703)
  api-cart     → bulkWrite with invalid $$ array filter (MongoServerError)
  api-order    → INSERT INTO orders (not_existed)    (PostgreSQL SQLSTATE 42703)
  api-payment  → INSERT INTO Payments (not_existed)  (MySQL Error 1054)

A test PASSES when the response status is 5xx.
A test FAILS when the response is 2xx/4xx or the request errors out entirely.

Usage:
  python call-api/post/5xx/run.py
  python call-api/post/5xx/run.py --base-url http://192.168.1.10
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
        ("api-product",  "bad-insert (PostgreSQL)",  f"{base}:8081/products/sim/bad-insert"),
        ("api-customer", "bad-insert (PostgreSQL)",  f"{base}:8082/sim/bad-insert"),
        ("api-cart",     "bad-insert (MongoDB)",     f"{base}:8083/cart/sim/bad-insert"),
        ("api-order",    "bad-insert (PostgreSQL)",  f"{base}:8084/orders/sim/bad-insert"),
        ("api-payment",  "bad-insert (MySQL)",       f"{base}:8085/sim/bad-insert"),
    ]


# ─── ANSI colours ─────────────────────────────────────────────────────────────
GREEN  = "\033[32m"
RED    = "\033[31m"
YELLOW = "\033[33m"
CYAN   = "\033[36m"
DIM    = "\033[2m"
RESET  = "\033[0m"


def call(url: str, timeout: int = 5) -> tuple[Optional[int], Optional[dict], Optional[str]]:
    """POST with empty JSON body. Returns (status_code, body_dict, error_str)."""
    try:
        req = urllib.request.Request(
            url,
            data=b"{}",
            headers={"Accept": "application/json", "Content-Type": "application/json"},
            method="POST",
        )
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8", errors="replace")
            try:
                return resp.status, json.loads(raw), None
            except json.JSONDecodeError:
                return resp.status, {"raw": raw[:120]}, None
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", errors="replace")
        try:
            return e.code, json.loads(raw), None
        except json.JSONDecodeError:
            return e.code, {"raw": raw[:120]}, None
    except Exception as exc:
        return None, None, str(exc)


def main() -> None:
    parser = argparse.ArgumentParser(description="Call all POST 5xx simulation endpoints")
    parser.add_argument("--base-url", default="http://localhost")
    parser.add_argument("--timeout",  type=int, default=5)
    args = parser.parse_args()

    base  = args.base_url.rstrip("/")
    cases = build_cases(base)

    passed = 0
    failed = 0

    print(f"\n{CYAN}POST 5xx sim — {base}{RESET}\n")

    for service, label, url in cases:
        status, body, err = call(url, args.timeout)

        is_5xx = status is not None and 500 <= status < 600
        path   = url.replace(base, "")

        if err:
            failed += 1
            print(f"  {RED}  ERR  FAIL{RESET}  {DIM}POST {path}{RESET}")
            print(f"        {YELLOW}{err}{RESET}")
            continue

        if is_5xx:
            passed += 1
            tag = f"{GREEN}  {status}  OK  {RESET}"
        else:
            failed += 1
            tag = f"{RED}  {status}  FAIL{RESET}"

        print(f"  {tag}  {DIM}POST {path}{RESET}  {DIM}[{service} — {label}]{RESET}")

        if body and isinstance(body, dict):
            detail = body.get("detail") or body.get("error") or ""
            if detail:
                print(f"        {YELLOW}{str(detail)[:120]}{RESET}")

        time.sleep(0.05)

    total = passed + failed
    print(f"\n{'─' * 50}")
    print(f"  {GREEN}passed: {passed}/{total}{RESET}   {RED}failed: {failed}/{total}{RESET}")
    print(f"{'─' * 50}\n")

    if failed > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
