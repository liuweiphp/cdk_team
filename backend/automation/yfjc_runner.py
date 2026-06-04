import json
import os
import re
import sys
import time
from http.cookies import SimpleCookie
from pathlib import Path
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
        response = error_response(payload, str(exc))
    json.dump(response, sys.stdout, ensure_ascii=False)


def prepare_order(payload):
    base_url = os.getenv("YFJC_BASE_URL", DEFAULT_BASE_URL).rstrip("/")
    username = (payload.get("external_username") or "").strip()
    password = (payload.get("external_password") or "").strip()
    if not username:
        return manual_review("", "外部采购账号用户名不能为空")
    if not password:
        return manual_review("", "外部采购账号密码不能为空")

    register_resp, login_resp, auth = register_and_login(base_url, username, password)
    authed_headers = headers(auth)

    order_payload = build_order_payload(payload)

    save_resp = post_json(f"{base_url}/api/v1/user/order/save", order_payload, authed_headers)
    trade_no = find_first(save_resp, ["trade_no", "tradeNo", "order_no", "orderNo"])
    if not trade_no:
        return manual_review("", "order/save 未返回订单号", {"order_save": save_resp})

    detail_resp = get_json(
        f"{base_url}/api/v1/user/order/detail?{urllib.parse.urlencode({'trade_no': trade_no})}",
        authed_headers,
    )
    checkout_result = try_checkout_in_browser(base_url, trade_no, payload)

    return {
        "status": "pending_payment",
        "external_order_no": trade_no,
        "external_username": username,
        "external_password": password,
        "subscribe_url": "",
        "payment_status": "unpaid",
        "payload_json": json.dumps(
            {
                "external_account": {"username": username, "password": password},
                "auth": auth,
                "register": register_resp,
                "login": login_resp,
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


def build_register_payload(username, password):
    configured = os.getenv("YFJC_REGISTER_PAYLOAD_JSON", "").strip()
    if configured:
        body = json.loads(configured)
    else:
        body = {}
    body.setdefault("email", username)
    body.setdefault("password", password)
    invite_code = os.getenv("YFJC_INVITE_CODE", "").strip()
    if invite_code:
        body.setdefault("invite_code", invite_code)
    email_code = os.getenv("YFJC_EMAIL_CODE", "").strip()
    if email_code:
        body.setdefault("email_code", email_code)
    return body


def build_login_payload(username, password):
    configured = os.getenv("YFJC_LOGIN_PAYLOAD_JSON", "").strip()
    if configured:
        body = json.loads(configured)
    else:
        body = {}
    body.setdefault("email", username)
    body.setdefault("password", password)
    return body


def register_and_login(base_url, username, password):
    mode = os.getenv("YFJC_AUTH_MODE", "browser").strip().lower()
    if mode == "api":
        return register_and_login_via_api(base_url, username, password)
    try:
        return register_and_login_via_browser(base_url, username, password)
    except Exception as exc:
        if os.getenv("YFJC_AUTH_BROWSER_FALLBACK_API", "0") == "1":
            return register_and_login_via_api(base_url, username, password)
        raise RuntimeError(f"浏览器注册登录失败: {exc}")


def register_and_login_via_api(base_url, username, password):
    register_payload = build_register_payload(username, password)
    try:
        register_resp = post_json(f"{base_url}{path_env('YFJC_REGISTER_PATH', '/api/v1/passport/auth/register')}", register_payload)
        ensure_api_success(register_resp, "注册")
    except Exception as exc:
        if not is_already_registered_error(exc):
            raise
        register_resp = {"message": str(exc), "ignored": "account_already_registered"}

    login_payload = build_login_payload(username, password)
    login_resp, login_headers = post_json_with_headers(
        f"{base_url}{path_env('YFJC_LOGIN_PATH', '/api/v1/passport/auth/login')}",
        login_payload,
    )
    ensure_api_success(login_resp, "登录")
    auth = extract_auth(login_resp, login_headers)
    ensure_auth_present(auth)
    return register_resp, login_resp, auth


def register_and_login_via_browser(base_url, username, password):
    try:
        from playwright.sync_api import sync_playwright
    except Exception as exc:
        raise RuntimeError(f"playwright unavailable: {exc}")

    register_payload = build_register_payload(username, password)
    login_payload = build_login_payload(username, password)
    wait_seconds = int(os.getenv("YFJC_AUTH_WAIT_SECONDS", "8"))
    profile_dir = Path(os.getenv("YFJC_PROFILE_DIR", "/tmp/yfjc_profiles")) / safe_filename(username)

    with sync_playwright() as p:
        session = open_browser_session(p, base_url, profile_dir)
        context = session["context"]
        page = session["page"]
        try:
            page.goto(base_url, wait_until="domcontentloaded", timeout=30000)
            if not wait_for_cloudflare(page, int(os.getenv("YFJC_CLOUDFLARE_WAIT_SECONDS", "45"))):
                raise RuntimeError("目标站仍返回 Cloudflare 人机验证页")
            if wait_seconds > 0:
                page.wait_for_timeout(wait_seconds * 1000)

            register_resp = page_post_json(
                page,
                f"{base_url}{path_env('YFJC_REGISTER_PATH', '/api/v1/passport/auth/register')}",
                register_payload,
            )
            try:
                ensure_api_success(register_resp["body"], "注册")
            except Exception as exc:
                if not is_already_registered_error(exc):
                    raise
                register_resp["body"] = {"message": str(exc), "ignored": "account_already_registered"}

            login_resp = page_post_json(
                page,
                f"{base_url}{path_env('YFJC_LOGIN_PATH', '/api/v1/passport/auth/login')}",
                login_payload,
            )
            ensure_api_success(login_resp["body"], "登录")
            auth = extract_auth(login_resp["body"], login_resp["headers"])
            auth["cookie"] = merge_cookie_headers(auth.get("cookie", ""), cookies_to_header(context.cookies(base_url)))
            ensure_auth_present(auth)
            return register_resp["body"], login_resp["body"], auth
        except Exception as exc:
            artifact = save_browser_debug_artifacts(page, username, "register_login_failed")
            if artifact:
                raise RuntimeError(f"{exc} | debug={artifact}") from exc
            raise
        finally:
            session["cleanup"]()


def open_browser_session(playwright, base_url, profile_dir):
    cdp_url = browser_cdp_url()
    if cdp_url:
        try:
            return connect_remote_browser_session(playwright, base_url, cdp_url)
        except Exception as exc:
            if os.getenv("YFJC_BROWSER_CDP_FALLBACK_LOCAL", "1") != "1":
                raise RuntimeError(f"连接远程已验证浏览器失败: {exc}") from exc
    return launch_local_browser_session(playwright, profile_dir)


def browser_cdp_url():
    path = os.getenv("YFJC_BROWSER_CDP_URL_FILE", "").strip()
    if path:
        try:
            value = Path(path).read_text(encoding="utf-8").strip()
            if value:
                return value
        except FileNotFoundError:
            pass
    return os.getenv("YFJC_BROWSER_CDP_URL", "").strip()


def connect_remote_browser_session(playwright, base_url, cdp_url):
    timeout_ms = int(os.getenv("YFJC_BROWSER_TIMEOUT_MS", "30000"))
    browser = playwright.chromium.connect_over_cdp(cdp_url, timeout=timeout_ms)
    if not browser.contexts:
        raise RuntimeError("远程浏览器未返回可复用 context")
    context_index = int(os.getenv("YFJC_BROWSER_CONTEXT_INDEX", "0"))
    if context_index < 0 or context_index >= len(browser.contexts):
        raise RuntimeError(f"远程浏览器 context 索引越界: {context_index}")
    context = browser.contexts[context_index]
    context.set_default_timeout(timeout_ms)
    page = select_reusable_page(context.pages, base_url)
    created = False
    if page is None:
        page = context.new_page()
        created = True

    def cleanup():
        if created:
            try:
                page.close()
            except Exception:
                pass

    return {"context": context, "page": page, "cleanup": cleanup}


def launch_local_browser_session(playwright, profile_dir):
    profile_dir.mkdir(parents=True, exist_ok=True)
    launch_options = browser_launch_options()
    context = playwright.chromium.launch_persistent_context(
        user_data_dir=str(profile_dir),
        **launch_options,
    )
    context.set_default_timeout(int(os.getenv("YFJC_BROWSER_TIMEOUT_MS", "30000")))
    page = context.new_page()

    def cleanup():
        context.close()

    return {"context": context, "page": page, "cleanup": cleanup}


def select_reusable_page(pages, base_url):
    if not pages:
        return None
    host = urllib.parse.urlparse(base_url).netloc.lower()
    for page in reversed(list(pages)):
        page_url = str(getattr(page, "url", "") or "").lower()
        if host and host in page_url:
            return page
    return list(pages)[-1]


def browser_launch_options():
    headless = os.getenv("YFJC_HEADLESS", "1") != "0"
    channel = os.getenv("YFJC_BROWSER_CHANNEL", "").strip() or None
    width = int(os.getenv("YFJC_VIEWPORT_WIDTH", "1440"))
    height = int(os.getenv("YFJC_VIEWPORT_HEIGHT", "900"))
    args = [
        "--disable-blink-features=AutomationControlled",
        "--disable-dev-shm-usage",
        "--no-sandbox",
        "--disable-gpu",
        "--disable-features=IsolateOrigins,site-per-process,AutomationControlled",
        "--disable-site-isolation-trials",
        "--no-default-browser-check",
        "--no-first-run",
        "--disable-infobars",
        "--disable-notifications",
        "--disable-popup-blocking",
        "--disable-extensions",
        "--lang=zh-CN,zh,en-US,en",
        f"--window-size={width},{height}",
    ]
    if headless and os.getenv("YFJC_HEADLESS_NEW", "1") == "1" and not channel:
        args.append("--headless=new")

    options = {
        "headless": False if (headless and os.getenv("YFJC_HEADLESS_NEW", "1") == "1" and not channel) else headless,
        "viewport": {"width": width, "height": height},
        "screen": {"width": width, "height": height},
        "user_agent": browser_user_agent(),
        "locale": os.getenv("YFJC_LOCALE", "zh-CN"),
        "timezone_id": os.getenv("YFJC_TIMEZONE", "Asia/Shanghai"),
        "device_scale_factor": float(os.getenv("YFJC_DEVICE_SCALE_FACTOR", "1")),
        "has_touch": os.getenv("YFJC_HAS_TOUCH", "0") == "1",
        "is_mobile": os.getenv("YFJC_IS_MOBILE", "0") == "1",
        "color_scheme": os.getenv("YFJC_COLOR_SCHEME", "light"),
        "reduced_motion": os.getenv("YFJC_REDUCED_MOTION", "no-preference"),
        "args": args,
    }
    if channel:
        options["channel"] = channel
    proxy = parse_proxy(os.getenv("YFJC_PROXY") or os.getenv("PLAYWRIGHT_PROXY_URL") or "")
    if proxy:
        options["proxy"] = proxy
    return options


def browser_user_agent():
    return os.getenv(
        "YFJC_USER_AGENT",
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
        "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36",
    )


def parse_proxy(value):
    value = (value or "").strip()
    if not value:
        return None
    parsed = urllib.parse.urlparse(value)
    if not parsed.scheme or not parsed.hostname:
        return {"server": value}
    proxy = {"server": f"{parsed.scheme}://{parsed.hostname}"}
    if parsed.port:
        proxy["server"] = f"{proxy['server']}:{parsed.port}"
    if parsed.username:
        proxy["username"] = urllib.parse.unquote(parsed.username)
    if parsed.password:
        proxy["password"] = urllib.parse.unquote(parsed.password)
    return proxy


def page_post_json(page, url, body):
    response = page.evaluate(
        """async ({ url, body }) => {
            const resp = await fetch(url, {
                method: 'POST',
                credentials: 'include',
                headers: {
                    'Content-Type': 'application/json',
                    'Accept': 'application/json'
                },
                body: JSON.stringify(body)
            });
            const headers = {};
            resp.headers.forEach((value, key) => { headers[key] = value; });
            const text = await resp.text();
            return { ok: resp.ok, status: resp.status, headers, text };
        }""",
        {"url": url, "body": body},
    )
    raw = response.get("text") or ""
    content_type = header_value(response.get("headers") or {}, "content-type")
    if not raw.strip():
        raise RuntimeError(f"{url} returned {response.get('status')} empty response")
    if "json" not in content_type.lower() and not raw.lstrip().startswith(("{", "[")):
        raise RuntimeError(f"{url} returned non-json {response.get('status')} {content_type}: {summarize_response(raw)}")
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"{url} returned invalid json {response.get('status')} {content_type}: {summarize_response(raw)}") from exc
    if not response.get("ok"):
        raise RuntimeError(f"{url} returned {response.get('status')}: {summarize_response(raw)}")
    return {"body": parsed, "headers": response.get("headers") or {}}


def path_env(name, default):
    value = os.getenv(name, default).strip()
    if not value.startswith("/"):
        value = "/" + value
    return value


def post_json(url, body, request_headers=None):
    resp, _ = post_json_with_headers(url, body, request_headers)
    return resp


def post_json_with_headers(url, body, request_headers=None):
    data = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, data=data, method="POST", headers=request_headers or headers())
    return request_json_with_headers(req)


def get_json(url, request_headers=None):
    req = urllib.request.Request(url, method="GET", headers=request_headers or headers())
    return request_json(req)


def request_json(req):
    body, _ = request_json_with_headers(req)
    return body


def request_json_with_headers(req):
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read().decode("utf-8")
            response_headers = dict(resp.headers.items())
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{req.full_url} returned {exc.code}: {raw}")
    if not raw:
        return {}, response_headers
    content_type = header_value(response_headers, "content-type")
    if "json" not in content_type.lower() and not raw.lstrip().startswith(("{", "[")):
        raise RuntimeError(f"{req.full_url} returned non-json {content_type}: {summarize_response(raw)}")
    try:
        return json.loads(raw), response_headers
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"{req.full_url} returned invalid json {content_type}: {summarize_response(raw)}") from exc


def headers(auth=None):
    out = {
        "Content-Type": "application/json",
        "Accept": "application/json",
        "User-Agent": browser_user_agent(),
    }
    cookie = os.getenv("YFJC_COOKIE", "").strip()
    if cookie:
        out["Cookie"] = cookie
    token = os.getenv("YFJC_AUTH_TOKEN", "").strip()
    if token:
        out["Authorization"] = token if token.lower().startswith("bearer ") else f"Bearer {token}"
    if auth:
        auth_cookie = (auth.get("cookie") or "").strip()
        if auth_cookie:
            out["Cookie"] = auth_cookie
        auth_token = (auth.get("token") or "").strip()
        if auth_token:
            out["Authorization"] = auth_token if auth_token.lower().startswith("bearer ") else f"Bearer {auth_token}"
    return out


def extract_auth(body, response_headers):
    cookie = set_cookie_to_cookie_header(header_value(response_headers, "set-cookie"))
    token = find_first(body, ["auth_data", "token", "access_token", "accessToken", "authorization", "data"])
    return {"cookie": cookie, "token": token}


def ensure_api_success(body, step):
    if not isinstance(body, dict):
        raise RuntimeError(f"{step}返回格式异常: {body}")
    code = body.get("code")
    if code not in (None, 0, 200, "0", "200"):
        raise RuntimeError(f"{step}失败: {compact_json(body)}")
    status = body.get("status")
    if status is False or status in ("fail", "failed", "error"):
        raise RuntimeError(f"{step}失败: {compact_json(body)}")
    message = str(body.get("message") or body.get("msg") or "")
    data = body.get("data")
    if data is None and message and not is_success_message(message):
        raise RuntimeError(f"{step}失败: {compact_json(body)}")


def ensure_auth_present(auth):
    if not (auth.get("token") or auth.get("cookie")):
        raise RuntimeError("登录成功但未获取到 token 或 cookie")


def is_success_message(message):
    lowered = message.lower()
    return any(word in lowered for word in ["success", "successful", "ok"]) or any(word in message for word in ["成功"])


def is_already_registered_error(exc):
    text = str(exc).lower()
    return any(word in text for word in ["already", "exist", "registered"]) or any(word in str(exc) for word in ["已存在", "已经注册", "邮箱已被使用"])


def header_value(headers_map, name):
    name = name.lower()
    for key, value in (headers_map or {}).items():
        if key.lower() == name:
            return value
    return ""


def cookies_to_header(cookies):
    return "; ".join([f"{cookie['name']}={cookie['value']}" for cookie in cookies if cookie.get("name")])


def set_cookie_to_cookie_header(value):
    if not value:
        return ""
    cookie = SimpleCookie()
    cookie.load(value)
    return "; ".join([f"{key}={morsel.value}" for key, morsel in cookie.items()])


def merge_cookie_headers(*values):
    parts = []
    for value in values:
        if not value:
            continue
        parts.extend([part.strip() for part in value.split(";") if part.strip()])
    return "; ".join(dict.fromkeys(parts))


def compact_json(value):
    return json.dumps(value, ensure_ascii=False, separators=(",", ":"))[:500]


def summarize_response(raw):
    text = " ".join(str(raw).replace("\r", " ").replace("\n", " ").split())
    if "cf-mitigated" in text.lower() or "just a moment" in text.lower() or "challenges.cloudflare.com" in text.lower():
        return "Cloudflare challenge page"
    return text[:500]


def wait_for_cloudflare(page, timeout_seconds):
    deadline = time.monotonic() + max(timeout_seconds, 0)
    while time.monotonic() < deadline:
        try:
            title = (page.title() or "").lower()
            text = page.evaluate("() => (document.body?.innerText || '').toLowerCase()")[:2000]
            if not is_cloudflare_text(title + "\n" + text):
                return True
        except Exception:
            pass
        page.wait_for_timeout(2000)
    try:
        title = (page.title() or "").lower()
        text = page.evaluate("() => (document.body?.innerText || '').toLowerCase()")[:2000]
        return not is_cloudflare_text(title + "\n" + text)
    except Exception:
        return False


def is_cloudflare_text(text):
    lowered = (text or "").lower()
    return "just a moment" in lowered or "checking your browser" in lowered or "cloudflare" in lowered


def save_browser_debug_artifacts(page, username, label):
    if os.getenv("YFJC_SAVE_DEBUG_ARTIFACTS", "1") == "0":
        return ""
    out_dir = Path(os.getenv("YFJC_DEBUG_DIR", "/tmp/yfjc_debug"))
    out_dir.mkdir(parents=True, exist_ok=True)
    base = out_dir / f"{int(time.time())}_{safe_filename(username)}_{safe_filename(label)}"
    paths = []
    try:
        png_path = base.with_suffix(".png")
        page.screenshot(path=str(png_path), full_page=True)
        paths.append(str(png_path))
    except Exception:
        pass
    try:
        html_path = base.with_suffix(".html")
        html_path.write_text(page.content(), encoding="utf-8")
        paths.append(str(html_path))
    except Exception:
        pass
    return ",".join(paths)


def safe_filename(value):
    value = str(value or "").strip()
    value = re.sub(r"[^a-zA-Z0-9._-]+", "_", value)
    return value[:120] or "item"


def try_checkout_in_browser(base_url, trade_no, payload):
    if os.getenv("YFJC_USE_BROWSER", "1") == "0":
        return {"status": "skipped", "reason": "YFJC_USE_BROWSER=0"}
    try:
        from playwright.sync_api import sync_playwright
    except Exception as exc:
        return {"status": "skipped", "reason": f"playwright unavailable: {exc}"}

    detail_url = f"{base_url}/#/order/detail?trade_no={urllib.parse.quote(trade_no)}"
    profile_dir = Path(os.getenv("YFJC_PROFILE_DIR", "/tmp/yfjc_profiles")) / safe_filename(
        payload.get("external_username") or payload.get("account_name") or trade_no
    )
    with sync_playwright() as p:
        session = open_browser_session(p, base_url, profile_dir)
        page = session["page"]
        try:
            page.goto(detail_url, wait_until="domcontentloaded", timeout=30000)
            click_checkout(page)
        finally:
            session["cleanup"]()
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


def error_response(payload, error):
    order_no = payload.get("external_order_no", "")
    screenshot_path, html_dump_path = extract_debug_paths(error)
    if "Cloudflare 人机验证页" in error:
        reason = first_line(error)
        return {
            "status": "entry_challenge_required",
            "external_order_no": order_no or "",
            "subscribe_url": "",
            "manual_review_reason": reason,
            "screenshot_path": screenshot_path,
            "html_dump_path": html_dump_path,
            "payload_json": json.dumps({}, ensure_ascii=False),
            "error": error,
        }
    result = manual_review(order_no, error)
    if screenshot_path:
        result["screenshot_path"] = screenshot_path
    if html_dump_path:
        result["html_dump_path"] = html_dump_path
    return result


def extract_debug_paths(error):
    marker = "debug="
    idx = error.find(marker)
    if idx == -1:
        return "", ""
    tail = error[idx + len(marker):].strip()
    paths = [item.strip() for item in tail.split(",") if item.strip()]
    screenshot_path = ""
    html_dump_path = ""
    for path in paths:
        lowered = path.lower()
        if lowered.endswith(".png") and not screenshot_path:
            screenshot_path = path
        elif lowered.endswith(".html") and not html_dump_path:
            html_dump_path = path
    return screenshot_path, html_dump_path


def first_line(value):
    return str(value or "").split("|", 1)[0].strip()


if __name__ == "__main__":
    main()
