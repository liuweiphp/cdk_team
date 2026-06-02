import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request


DEFAULT_BASE_URL = "https://www.yfjc.xyz"
DEFAULT_PRODUCT_NAME = "月卡一月"


def main():
    payload = json.load(sys.stdin)
    action = payload.get("action", "prepare_order")
    try:
        if action == "prepare_order":
            response = prepare_order(payload)
        elif action == "fetch_subscribe":
            response = fetch_subscribe(payload)
        else:
            response = manual_review("", f"unsupported action: {action}")
    except Exception as exc:
        response = manual_review(payload.get("external_order_no", ""), str(exc))
    json.dump(response, sys.stdout, ensure_ascii=False)


def prepare_order(payload):
    base_url = os.getenv("YFJC_BASE_URL", DEFAULT_BASE_URL).rstrip("/")
    order_payload = build_order_payload(payload)

    save_resp = post_json(f"{base_url}/api/v1/user/order/save", order_payload)
    trade_no = find_first(save_resp, ["trade_no", "tradeNo", "order_no", "orderNo"])
    if not trade_no:
        return manual_review("", "order/save 未返回订单号", {"order_save": save_resp})

    detail_resp = get_json(f"{base_url}/api/v1/user/order/detail?{urllib.parse.urlencode({'trade_no': trade_no})}")
    checkout_result = try_checkout_in_browser(base_url, trade_no, payload)

    return {
        "status": "pending_payment",
        "external_order_no": trade_no,
        "subscribe_url": "",
        "payment_status": "unpaid",
        "payload_json": json.dumps(
            {
                "order_save": save_resp,
                "order_detail": detail_resp,
                "checkout": checkout_result,
                "order_payload": order_payload,
            },
            ensure_ascii=False,
        ),
        "error": "",
    }


def fetch_subscribe(payload):
    base_url = os.getenv("YFJC_BASE_URL", DEFAULT_BASE_URL).rstrip("/")
    resp = get_json(f"{base_url}/api/v1/user/getSubscribe")
    subscribe_url = find_first(resp, ["subscribe_url", "subscribeUrl"])
    if subscribe_url:
        return {
            "status": "ready",
            "external_order_no": payload.get("external_order_no", ""),
            "subscribe_url": subscribe_url,
            "payment_status": "paid",
            "payload_json": json.dumps({"get_subscribe": resp}, ensure_ascii=False),
            "error": "",
        }
    return manual_review(payload.get("external_order_no", ""), "getSubscribe 未返回 data.subscribe_url", {"get_subscribe": resp})


def build_order_payload(payload):
    configured = os.getenv("YFJC_ORDER_PAYLOAD_JSON", "").strip()
    if configured:
        body = json.loads(configured)
    else:
        target_code = payload.get("target_code") or "month_1"
        target_name = payload.get("target_name") or DEFAULT_PRODUCT_NAME
        body = {
            "product": target_code,
            "plan": target_code,
            "period": target_code,
            "name": target_name,
        }
    body.setdefault("account_name", payload.get("account_name", ""))
    return body


def post_json(url, body):
    data = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, data=data, method="POST", headers=headers())
    return request_json(req)


def get_json(url):
    req = urllib.request.Request(url, method="GET", headers=headers())
    return request_json(req)


def request_json(req):
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read().decode("utf-8")
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{req.full_url} returned {exc.code}: {raw}")
    if not raw:
        return {}
    return json.loads(raw)


def headers():
    out = {
        "Content-Type": "application/json",
        "Accept": "application/json",
        "User-Agent": os.getenv("YFJC_USER_AGENT", "Mozilla/5.0"),
    }
    cookie = os.getenv("YFJC_COOKIE", "").strip()
    if cookie:
        out["Cookie"] = cookie
    token = os.getenv("YFJC_AUTH_TOKEN", "").strip()
    if token:
        out["Authorization"] = token if token.lower().startswith("bearer ") else f"Bearer {token}"
    return out


def try_checkout_in_browser(base_url, trade_no, payload):
    if os.getenv("YFJC_USE_BROWSER", "1") == "0":
        return {"status": "skipped", "reason": "YFJC_USE_BROWSER=0"}
    try:
        from playwright.sync_api import sync_playwright
    except Exception as exc:
        return {"status": "skipped", "reason": f"playwright unavailable: {exc}"}

    detail_url = f"{base_url}/#/order/detail?trade_no={urllib.parse.quote(trade_no)}"
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=os.getenv("YFJC_HEADLESS", "1") != "0")
        context = browser.new_context()
        page = context.new_page()
        page.goto(detail_url, wait_until="domcontentloaded", timeout=30000)
        click_checkout(page)
        browser.close()
    return {"status": "clicked", "detail_url": detail_url}


def click_checkout(page):
    candidates = [
        "text=结账",
        "text=去结账",
        "text=立即支付",
        "text=支付",
        "button:has-text('结账')",
        "button:has-text('支付')",
    ]
    for selector in candidates:
        locator = page.locator(selector)
        if locator.count() == 0:
            continue
        locator.first.click(timeout=10000)
        page.wait_for_load_state("domcontentloaded", timeout=10000)
        return
    raise RuntimeError("订单详情页未找到结账按钮")


def find_first(value, keys):
    if isinstance(value, dict):
        for key in keys:
            current = value.get(key)
            if isinstance(current, str) and current:
                return current
        for nested in value.values():
            found = find_first(nested, keys)
            if found:
                return found
    elif isinstance(value, list):
        for item in value:
            found = find_first(item, keys)
            if found:
                return found
    return ""


def manual_review(order_no, error, payload=None):
    return {
        "status": "needs_manual_review",
        "external_order_no": order_no or "",
        "subscribe_url": "",
        "manual_review_reason": error,
        "payload_json": json.dumps(payload or {}, ensure_ascii=False),
        "error": error,
    }


if __name__ == "__main__":
    main()
