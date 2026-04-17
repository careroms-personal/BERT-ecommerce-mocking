"""
call-api/get/2xx/run.py
-----------------------
Call every GET endpoint across all services and report 2xx responses.

Seed UUIDs used (from seeds/):
  Product A       → 11111111-1111-1111-1111-000000000001
  Customer 1      → 22222222-2222-2222-2222-000000000001
  Customer 2      → 22222222-2222-2222-2222-000000000002
  Order (PENDING) → 33333333-0001-0000-0000-000000000001
  Payment (PEND.) → 44444444-0001-0000-0000-000000000001

Usage:
  python call-api/get/2xx/run.py
  python call-api/get/2xx/run.py --base-url http://192.168.1.10
"""

import argparse
import json
import sys
import time
import urllib.request
import urllib.error
from dataclasses import dataclass
from typing import Optional

# ─── Seed UUIDs ───────────────────────────────────────────────────────────────
PRODUCT_A   = "11111111-1111-1111-1111-000000000001"
CUSTOMER_1  = "22222222-2222-2222-2222-000000000001"
CUSTOMER_2  = "22222222-2222-2222-2222-000000000002"
ORDER_1     = "33333333-0001-0000-0000-000000000001"
PAYMENT_1   = "44444444-0001-0000-0000-000000000001"


@dataclass
class Case:
    service: str
    label: str
    path: str


def build_cases(base: str) -> list[tuple[str, str, str]]:
    """Returns list of (service, label, url)."""
    return [
        # ── api-product ──────────────────────────────────────────────────────
        ("api-product", "health",                  f"{base}:8081/health"),
        ("api-product", "list products",            f"{base}:8081/products"),
        ("api-product", "list products page=2",     f"{base}:8081/products?page=2&limit=2"),
        ("api-product", "list by category",         f"{base}:8081/products?category=prod1"),
        ("api-product", "get product",              f"{base}:8081/products/{PRODUCT_A}"),
        ("api-product", "get stock",                f"{base}:8081/products/{PRODUCT_A}/stock"),

        # ── api-customer ─────────────────────────────────────────────────────
        ("api-customer", "health",                  f"{base}:8082/health"),
        ("api-customer", "get customer 1",          f"{base}:8082/customers/{CUSTOMER_1}"),
        ("api-customer", "get customer 2",          f"{base}:8082/customers/{CUSTOMER_2}"),

        # ── api-cart ─────────────────────────────────────────────────────────
        ("api-cart", "health",                      f"{base}:8083/health"),
        ("api-cart", "get cart customer 1",         f"{base}:8083/cart/{CUSTOMER_1}"),
        ("api-cart", "get cart customer 2",         f"{base}:8083/cart/{CUSTOMER_2}"),

        # ── api-order ────────────────────────────────────────────────────────
        ("api-order", "health",                     f"{base}:8084/health"),
        ("api-order", "get order",                  f"{base}:8084/orders/{ORDER_1}"),
        ("api-order", "list orders customer 1",     f"{base}:8084/orders/customer/{CUSTOMER_1}"),
        ("api-order", "list orders customer 2",     f"{base}:8084/orders/customer/{CUSTOMER_2}"),

        # ── api-payment ──────────────────────────────────────────────────────
        ("api-payment", "health",                   f"{base}:8085/health"),
        ("api-payment", "get payment",              f"{base}:8085/payments/{PAYMENT_1}"),

        # ── api-search ───────────────────────────────────────────────────────
        ("api-search", "health",                    f"{base}:8086/health"),
        ("api-search", "search q=product",          f"{base}:8086/search?q=product"),
        ("api-search", "search q=Product+A",        f"{base}:8086/search?q=Product+A"),
        ("api-search", "search with category",      f"{base}:8086/search?q=product&category=prod1"),
        ("api-search", "suggest q=pr",              f"{base}:8086/search/suggest?q=pr"),
        ("api-search", "suggest q=Prod",            f"{base}:8086/search/suggest?q=Prod"),
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
        return e.code, None, raw[:120]
    except Exception as e:
        return None, None, str(e)


def main() -> None:
    parser = argparse.ArgumentParser(description="Call all GET 2xx endpoints")
    parser.add_argument("--base-url", default="http://localhost", help="Base URL (default: http://localhost)")
    parser.add_argument("--timeout",  type=int, default=5, help="Request timeout in seconds (default: 5)")
    args = parser.parse_args()

    base = args.base_url.rstrip("/")
    cases = build_cases(base)

    passed = 0
    failed = 0
    results = []

    print(f"\n{CYAN}GET 2xx — {base}{RESET}\n")

    current_service = None
    for service, label, url in cases:
        if service != current_service:
            current_service = service
            print(f"{CYAN}── {service} ──{RESET}")

        status, body, err = call(url, args.timeout)

        is_2xx = status is not None and 200 <= status < 300

        if is_2xx:
            passed += 1
            tag = f"{GREEN}  {status} OK{RESET}"
        else:
            failed += 1
            tag = f"{RED}  {status or 'ERR'} FAIL{RESET}"

        path = url.replace(base, "")
        print(f"  {tag}  {DIM}GET {path}{RESET}")
        if err and not is_2xx:
            print(f"        {YELLOW}{err}{RESET}")

        results.append({
            "service": service,
            "label":   label,
            "url":     url,
            "status":  status,
            "ok":      is_2xx,
            "error":   err,
        })

        time.sleep(0.05)  # small delay to avoid hammering

    # ── Summary ───────────────────────────────────────────────────────────────
    total = passed + failed
    print(f"\n{'─' * 50}")
    print(f"  {GREEN}passed: {passed}/{total}{RESET}   {RED}failed: {failed}/{total}{RESET}")
    print(f"{'─' * 50}\n")

    if failed > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
