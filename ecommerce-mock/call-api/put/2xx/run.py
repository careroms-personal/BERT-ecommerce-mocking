"""
call-api/put/2xx/run.py
-----------------------
Call every PUT happy-path endpoint across all services and assert 2xx responses.

Endpoints covered
  api-product  PUT /products/:id          → partial update (name, price)
  api-customer PUT /customers/:id         → partial update (firstName, lastName)
  api-order    PUT /orders/:id/status     → status transition

All IDs come from seeded data — no prior POST step needed.
Neither api-cart, api-payment, nor api-search expose PUT endpoints.

Seed UUIDs used
  Product A  → 11111111-1111-1111-1111-000000000001
  Customer 1 → 22222222-2222-2222-2222-000000000001
  Order 1    → 33333333-0001-0000-0000-000000000001

Usage:
  python call-api/put/2xx/run.py
  python call-api/put/2xx/run.py --base-url http://192.168.1.10
"""

import argparse
import json
import sys
import time
import urllib.request
import urllib.error
from typing import Optional


# ─── Seed UUIDs ───────────────────────────────────────────────────────────────
PRODUCT_A  = "11111111-1111-1111-1111-000000000001"
CUSTOMER_1 = "22222222-2222-2222-2222-000000000001"
ORDER_1    = "33333333-0001-0000-0000-000000000001"

# ─── ANSI colours ─────────────────────────────────────────────────────────────
GREEN  = "\033[32m"
RED    = "\033[31m"
YELLOW = "\033[33m"
CYAN   = "\033[36m"
DIM    = "\033[2m"
RESET  = "\033[0m"


def build_cases(base: str) -> list[tuple[str, str, str, dict]]:
    """Returns list of (service, label, url, body)."""
    return [
        (
            "api-product",
            "update product (name + price)",
            f"{base}:8081/products/{PRODUCT_A}",
            {"name": "Product A", "price": 200.00},
        ),
        (
            "api-customer",
            "update customer profile",
            f"{base}:8082/customers/{CUSTOMER_1}",
            {"firstName": "customer", "lastName": "1"},
        ),
        (
            "api-order",
            "update order status → CONFIRMED",
            f"{base}:8084/orders/{ORDER_1}/status",
            {"status": "CONFIRMED"},
        ),
    ]


def call(
    url: str,
    body: dict,
    timeout: int = 5,
) -> tuple[Optional[int], Optional[dict], Optional[str]]:
    """PUT with JSON body. Returns (status_code, body_dict, error_str)."""
    data = json.dumps(body).encode()
    try:
        req = urllib.request.Request(
            url,
            data=data,
            headers={"Accept": "application/json", "Content-Type": "application/json"},
            method="PUT",
        )
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8", errors="replace")
            try:
                return resp.status, json.loads(raw), None
            except json.JSONDecodeError:
                return resp.status, {"raw": raw[:200]}, None
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", errors="replace")
        try:
            return e.code, json.loads(raw), None
        except json.JSONDecodeError:
            return e.code, {"raw": raw[:200]}, None
    except Exception as exc:
        return None, None, str(exc)


def main() -> None:
    parser = argparse.ArgumentParser(description="Call all PUT 2xx endpoints")
    parser.add_argument("--base-url", default="http://localhost")
    parser.add_argument("--timeout",  type=int, default=5)
    args = parser.parse_args()

    base  = args.base_url.rstrip("/")
    cases = build_cases(base)

    passed = 0
    failed = 0

    print(f"\n{CYAN}PUT 2xx — {base}{RESET}\n")

    current_service = None
    for service, label, url, body in cases:
        if service != current_service:
            current_service = service
            print(f"{CYAN}── {service} ──{RESET}")

        status, body_resp, err = call(url, body, args.timeout)
        path = url.replace(base, "")

        if err:
            failed += 1
            print(f"  {RED}  ERR  FAIL{RESET}  {DIM}PUT {path}{RESET}")
            print(f"        {YELLOW}{err}{RESET}")
            time.sleep(0.05)
            continue

        is_2xx = status is not None and 200 <= status < 300

        if is_2xx:
            passed += 1
            tag = f"{GREEN}  {status}  OK  {RESET}"
        else:
            failed += 1
            tag = f"{RED}  {status}  FAIL{RESET}"

        print(f"  {tag}  {DIM}PUT {path}{RESET}  {DIM}({label}){RESET}")

        if not is_2xx and body_resp and isinstance(body_resp, dict):
            msg = body_resp.get("error") or body_resp.get("message") or body_resp.get("raw", "")
            if msg:
                print(f"        {YELLOW}{str(msg)[:120]}{RESET}")

        time.sleep(0.05)

    total = passed + failed
    print(f"\n{'─' * 50}")
    print(f"  {GREEN}passed: {passed}/{total}{RESET}   {RED}failed: {failed}/{total}{RESET}")
    print(f"{'─' * 50}\n")

    if failed > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
