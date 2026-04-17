"""
call-api/post-delete/2xx/run.py
--------------------------------
Exercise all POST and DELETE happy-path endpoints across every service.
Steps run sequentially — later steps use IDs captured from earlier responses.

Flow
  api-product   POST /products                      → create sim product
  api-customer  POST /customers/register            → register sim user
                POST /auth/login                    → get tokens
                POST /auth/refresh                  → refresh tokens
                POST /auth/logout                   → revoke token
  api-cart      POST /cart/:userId/items            → add product to cart
                DELETE /cart/:userId/items/:itemId  → remove item
                POST /cart/:userId/items            → add again
                POST /cart/:userId/sync             → sync cart
                DELETE /cart/:userId                → clear cart
  api-order     POST /orders                        → create order (cancel target)
                POST /orders                        → create order (payment target)
                DELETE /orders/:id                  → cancel first order
  api-payment   POST /payments                      → pay second order (75% success)
  api-search    POST /search/index                  → index sim product

NOTE: api-payment uses a MockPaymentProvider (75% success · 15% decline · 10% timeout).
      A 402 response is a mock decline, not a test failure — flagged with OK*.

Usage:
  python call-api/post-delete/2xx/run.py
  python call-api/post-delete/2xx/run.py --base-url http://192.168.1.10
"""

import argparse
import json
import sys
import time
import uuid
import urllib.request
import urllib.error
from typing import Optional, Any, Callable


# ─── Seed UUIDs ───────────────────────────────────────────────────────────────
PRODUCT_A  = "11111111-1111-1111-1111-000000000001"
CUSTOMER_1 = "22222222-2222-2222-2222-000000000001"

# ─── ANSI colours ─────────────────────────────────────────────────────────────
GREEN  = "\033[32m"
RED    = "\033[31m"
YELLOW = "\033[33m"
CYAN   = "\033[36m"
DIM    = "\033[2m"
RESET  = "\033[0m"


# ─── HTTP helper ──────────────────────────────────────────────────────────────
def call(
    method: str,
    url: str,
    body: Optional[dict] = None,
    headers: Optional[dict] = None,
    timeout: int = 10,
) -> tuple[Optional[int], Optional[dict], Optional[str]]:
    """Returns (status_code, body_dict, error_str)."""
    data = json.dumps(body).encode() if body else None
    h: dict[str, str] = {"Accept": "application/json"}
    if data:
        h["Content-Type"] = "application/json"
    if headers:
        h.update(headers)

    try:
        req = urllib.request.Request(url, data=data, headers=h, method=method)
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


# ─── Runner ───────────────────────────────────────────────────────────────────
class Runner:
    def __init__(self, base: str, timeout: int) -> None:
        self.base    = base
        self.timeout = timeout
        self.state: dict[str, Any] = {}
        self.passed  = 0
        self.failed  = 0
        self._svc: Optional[str] = None

    def step(
        self,
        service: str,
        method: str,
        label: str,
        url: str,
        body: Optional[dict] = None,
        headers: Optional[dict] = None,
        on_success: Optional[Callable] = None,
        accept_status: tuple[int, ...] = (),
        skip_if: Optional[Callable] = None,
    ) -> bool:
        if self._svc != service:
            self._svc = service
            print(f"\n{CYAN}── {service} ──{RESET}")

        if skip_if and skip_if(self.state):
            print(f"  {DIM} SKIP   {method} {url.replace(self.base, '')}  ({label}){RESET}")
            return False

        status, body_resp, err = call(method, url, body, headers, self.timeout)
        path = url.replace(self.base, "")

        if err:
            self.failed += 1
            print(f"  {RED}  ERR  FAIL{RESET}  {DIM}{method} {path}{RESET}  {DIM}({label}){RESET}")
            print(f"        {YELLOW}{err}{RESET}")
            return False

        is_2xx      = status is not None and 200 <= status < 300
        is_accepted = status in accept_status
        ok          = is_2xx or is_accepted

        if ok:
            self.passed += 1
            tag = f"{GREEN}  {status}  OK  {RESET}" if is_2xx else f"{YELLOW}  {status}  OK* {RESET}"
        else:
            self.failed += 1
            tag = f"{RED}  {status}  FAIL{RESET}"

        print(f"  {tag}  {DIM}{method} {path}{RESET}  {DIM}({label}){RESET}")

        if not ok and body_resp and isinstance(body_resp, dict):
            msg = body_resp.get("error") or body_resp.get("message") or body_resp.get("raw", "")
            if msg:
                print(f"        {YELLOW}{str(msg)[:120]}{RESET}")

        if ok and on_success and body_resp:
            on_success(self.state, body_resp)

        time.sleep(0.1)
        return ok

    def summary(self) -> None:
        total = self.passed + self.failed
        print(f"\n{'─' * 50}")
        print(f"  {GREEN}passed: {self.passed}/{total}{RESET}   {RED}failed: {self.failed}/{total}{RESET}")
        print(f"  {YELLOW}OK* = accepted non-2xx (mock payment decline){RESET}")
        print(f"{'─' * 50}\n")


# ─── Main ─────────────────────────────────────────────────────────────────────
def main() -> None:
    parser = argparse.ArgumentParser(description="Call all POST/DELETE 2xx endpoints")
    parser.add_argument("--base-url", default="http://localhost")
    parser.add_argument("--timeout",  type=int, default=10)
    args = parser.parse_args()

    base = args.base_url.rstrip("/")
    r    = Runner(base, args.timeout)
    s    = r.state

    # Unique email per run so re-runs don't hit 409 on register
    sim_email = f"sim-{uuid.uuid4().hex[:8]}@test.local"
    sim_pass  = "SimPass123!"

    print(f"\n{CYAN}POST / DELETE 2xx — {base}{RESET}")
    print(f"{DIM}sim email: {sim_email}{RESET}\n")

    # ── api-product ───────────────────────────────────────────────────────────
    r.step(
        "api-product", "POST", "create product",
        f"{base}:8081/products",
        body={"name": "Sim Product", "description": "created by post-delete/2xx/run.py",
              "price": 9.99, "category": "sim", "stock": 5, "discontinued": False},
        on_success=lambda st, b: st.update({"SIM_PRODUCT_ID": b.get("id")}),
    )

    # ── api-customer ──────────────────────────────────────────────────────────
    r.step(
        "api-customer", "POST", "register sim user",
        f"{base}:8082/customers/register",
        body={"email": sim_email, "password": sim_pass,
              "firstName": "Sim", "lastName": "Test"},
        on_success=lambda st, b: st.update({"SIM_CUSTOMER_ID": b.get("id")}),
    )

    r.step(
        "api-customer", "POST", "login",
        f"{base}:8082/auth/login",
        body={"email": sim_email, "password": sim_pass},
        on_success=lambda st, b: st.update({
            "ACCESS_TOKEN":  b.get("accessToken"),
            "REFRESH_TOKEN": b.get("refreshToken"),
        }),
        skip_if=lambda st: not st.get("SIM_CUSTOMER_ID"),
    )

    r.step(
        "api-customer", "POST", "refresh token",
        f"{base}:8082/auth/refresh",
        headers={"Authorization": f"Bearer {s.get('REFRESH_TOKEN', '')}"},
        on_success=lambda st, b: st.update({
            "ACCESS_TOKEN":  b.get("accessToken"),
            "REFRESH_TOKEN": b.get("refreshToken"),
        }),
        skip_if=lambda st: not st.get("REFRESH_TOKEN"),
    )

    r.step(
        "api-customer", "POST", "logout",
        f"{base}:8082/auth/logout",
        headers={"Authorization": f"Bearer {s.get('ACCESS_TOKEN', '')}"},
        skip_if=lambda st: not st.get("ACCESS_TOKEN"),
    )

    # ── api-cart ──────────────────────────────────────────────────────────────
    cust = s.get("SIM_CUSTOMER_ID") or CUSTOMER_1

    r.step(
        "api-cart", "POST", "add item to cart",
        f"{base}:8083/cart/{cust}/items",
        body={"productId": PRODUCT_A, "name": "Product A", "price": 200.00, "quantity": 1},
        on_success=lambda st, b: st.update({
            "CART_ITEM_ID": ((b.get("items") or [{}])[-1]).get("itemId"),
        }),
    )

    r.step(
        "api-cart", "DELETE", "remove cart item",
        f"{base}:8083/cart/{cust}/items/{s.get('CART_ITEM_ID', '__missing__')}",
        skip_if=lambda st: not st.get("CART_ITEM_ID"),
    )

    r.step(
        "api-cart", "POST", "add item again (for sync)",
        f"{base}:8083/cart/{cust}/items",
        body={"productId": PRODUCT_A, "name": "Product A", "price": 200.00, "quantity": 1},
    )

    r.step(
        "api-cart", "POST", "sync cart",
        f"{base}:8083/cart/{cust}/sync",
    )

    r.step(
        "api-cart", "DELETE", "clear cart",
        f"{base}:8083/cart/{cust}",
    )

    # ── api-order ─────────────────────────────────────────────────────────────
    sim_cust   = s.get("SIM_CUSTOMER_ID") or CUSTOMER_1
    order_item = [{"product_id": PRODUCT_A, "name": "Product A",
                   "price": 200.00, "quantity": 1}]

    r.step(
        "api-order", "POST", "create order (cancel target)",
        f"{base}:8084/orders",
        body={"customer_id": sim_cust, "items": order_item},
        on_success=lambda st, b: st.update({"SIM_ORDER_CANCEL_ID": b.get("id")}),
    )

    r.step(
        "api-order", "POST", "create order (payment target)",
        f"{base}:8084/orders",
        body={"customer_id": sim_cust, "items": order_item},
        on_success=lambda st, b: st.update({"SIM_ORDER_PAY_ID": b.get("id")}),
    )

    r.step(
        "api-order", "DELETE", "cancel order",
        f"{base}:8084/orders/{s.get('SIM_ORDER_CANCEL_ID', '__missing__')}",
        skip_if=lambda st: not st.get("SIM_ORDER_CANCEL_ID"),
    )

    # ── api-payment ───────────────────────────────────────────────────────────
    r.step(
        "api-payment", "POST", "process payment (75% success · 15% decline*)",
        f"{base}:8085/payments",
        body={
            "orderId":    s.get("SIM_ORDER_PAY_ID", "00000000-0000-0000-0000-000000000000"),
            "customerId": sim_cust,
            "amount":     200.00,
        },
        on_success=lambda st, b: st.update({"SIM_PAYMENT_ID": b.get("id")}),
        accept_status=(402,),
        skip_if=lambda st: not st.get("SIM_ORDER_PAY_ID"),
    )

    # ── api-search ────────────────────────────────────────────────────────────
    r.step(
        "api-search", "POST", "index product",
        f"{base}:8086/search/index",
        body={
            "id":           s.get("SIM_PRODUCT_ID") or PRODUCT_A,
            "name":         "Sim Product",
            "description":  "created by post-delete/2xx/run.py",
            "price":        9.99,
            "category":     "sim",
            "stock":        5,
            "discontinued": False,
        },
    )

    r.summary()

    if r.failed > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
