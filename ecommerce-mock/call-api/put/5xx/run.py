"""
call-api/put/5xx/run.py
-----------------------
Call every PUT simulation endpoint across all services and assert 5xx responses.

Each endpoint intentionally runs a bad UPDATE against its database:
  api-product  → UPDATE products SET not_existed = 'sim'  (PostgreSQL SQLSTATE 42703)
  api-customer → UPDATE customers SET not_existed = 'sim' (PostgreSQL SQLSTATE 42703)
  api-cart     → bulkWrite with invalid $$ array filter    (MongoServerError)
  api-order    → UPDATE orders SET not_existed = 'sim'    (PostgreSQL SQLSTATE 42703)
  api-payment  → UPDATE Payments SET not_existed = 'sim'  (MySQL Error 1054)

A test PASSES when the response status is 5xx.
A test FAILS when the response is 2xx/4xx or the request errors out entirely.

Usage:
  python call-api/put/5xx/run.py
  python call-api/put/5xx/run.py --base-url http://192.168.1.10
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
        ("api-product",  "bad-update (PostgreSQL)",  f"{base}:8081/products/sim/bad-update"),
        ("api-customer", "bad-update (PostgreSQL)",  f"{base}:8082/sim/bad-update"),
        ("api-cart",     "bad-update (MongoDB)",     f"{base}:8083/cart/sim/bad-update"),
        ("api-order",    "bad-update (PostgreSQL)",  f"{base}:8084/orders/sim/bad-update"),
        ("api-payment",  "bad-update (MySQL)",       f"{base}:8085/sim/bad-update"),
    ]


# ─── ANSI colours ─────────────────────────────────────────────────────────────
GREEN  = "\033[32m"
RED    = "\033[31m"
YELLOW = "\033[33m"
CYAN   = "\033[36m"
DIM    = "\033[2m"
RESET  = "\033[0m"


def call(url: str, timeout: int = 5) -> tuple[Optional[int], Optional[dict], Optional[str]]:
    """PUT with empty JSON body. Returns (status_code, body_dict, error_str)."""
    try:
        req = urllib.request.Request(
            url,
            data=b"{}",
            headers={"Accept": "application/json", "Content-Type": "application/json"},
            method="PUT",
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
    parser = argparse.ArgumentParser(description="Call all PUT 5xx simulation endpoints")
    parser.add_argument("--base-url", default="http://localhost")
    parser.add_argument("--timeout",  type=int, default=5)
    args = parser.parse_args()

    base  = args.base_url.rstrip("/")
    cases = build_cases(base)

    passed = 0
    failed = 0

    print(f"\n{CYAN}PUT 5xx sim — {base}{RESET}\n")

    for service, label, url in cases:
        status, body, err = call(url, args.timeout)

        is_5xx = status is not None and 500 <= status < 600
        path   = url.replace(base, "")

        if err:
            failed += 1
            print(f"  {RED}  ERR  FAIL{RESET}  {DIM}PUT {path}{RESET}")
            print(f"        {YELLOW}{err}{RESET}")
            continue

        if is_5xx:
            passed += 1
            tag = f"{GREEN}  {status}  OK  {RESET}"
        else:
            failed += 1
            tag = f"{RED}  {status}  FAIL{RESET}"

        print(f"  {tag}  {DIM}PUT {path}{RESET}  {DIM}[{service} — {label}]{RESET}")

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
