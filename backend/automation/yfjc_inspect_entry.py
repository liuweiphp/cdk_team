import json
import os
import sys
import time
from pathlib import Path

from yfjc_runner import (
    DEFAULT_BASE_URL,
    browser_launch_options,
    header_value,
    is_cloudflare_text,
    safe_filename,
)


def main():
    payload = read_payload()
    base_url = (payload.get("base_url") or os.getenv("YFJC_BASE_URL") or DEFAULT_BASE_URL).rstrip("/")
    username = (payload.get("external_username") or payload.get("username") or "inspect").strip() or "inspect"
    profile_dir = Path(os.getenv("YFJC_PROFILE_DIR", "/tmp/yfjc_profiles")) / safe_filename(username)
    out_dir = Path(os.getenv("YFJC_DEBUG_DIR", "/tmp/yfjc_debug"))
    wait_seconds = int(os.getenv("YFJC_ENTRY_WAIT_SECONDS", "8"))
    click_register_entry = bool(payload.get("click_register_entry"))

    try:
        from playwright.sync_api import sync_playwright
    except Exception as exc:
        json.dump({"status": "error", "error": f"playwright unavailable: {exc}"}, sys.stdout, ensure_ascii=False)
        return

    out_dir.mkdir(parents=True, exist_ok=True)
    profile_dir.mkdir(parents=True, exist_ok=True)
    stamp = f"{int(time.time())}_{safe_filename(username)}"
    screenshot_path = str(out_dir / f"{stamp}_entry.png")
    html_path = str(out_dir / f"{stamp}_entry.html")
    register_screenshot_path = str(out_dir / f"{stamp}_register_entry.png")
    register_html_path = str(out_dir / f"{stamp}_register_entry.html")

    summary = {
        "status": "unknown",
        "base_url": base_url,
        "profile_dir": str(profile_dir),
        "final_url": "",
        "title": "",
        "cloudflare_detected": False,
        "body_preview": "",
        "screenshot_path": screenshot_path,
        "html_path": html_path,
        "responses": [],
        "cookies": [],
        "register_candidates": [],
        "register_click_attempted": False,
        "register_result": {},
        "error": "",
    }

    with sync_playwright() as p:
        context = p.chromium.launch_persistent_context(
            user_data_dir=str(profile_dir),
            **browser_launch_options(),
        )
        context.set_default_timeout(int(os.getenv("YFJC_BROWSER_TIMEOUT_MS", "30000")))
        page = context.new_page()

        responses = []

        def on_response(resp):
            try:
                headers = resp.headers or {}
                responses.append(
                    {
                        "url": resp.url,
                        "status": resp.status,
                        "content_type": header_value(headers, "content-type"),
                        "server": header_value(headers, "server"),
                        "cf_ray": header_value(headers, "cf-ray"),
                        "location": header_value(headers, "location"),
                    }
                )
            except Exception:
                pass

        page.on("response", on_response)

        try:
            page.goto(base_url, wait_until="domcontentloaded", timeout=30000)
            if wait_seconds > 0:
                page.wait_for_timeout(wait_seconds * 1000)

            summary["final_url"] = page.url
            summary["title"] = page.title() or ""
            body_text = page.evaluate("() => (document.body?.innerText || '').trim()")
            body_text = " ".join(body_text.split())
            summary["body_preview"] = body_text[:1000]
            summary["cloudflare_detected"] = is_cloudflare_text(
                "\n".join([summary["title"], summary["final_url"], body_text])
            )
            page.screenshot(path=screenshot_path, full_page=True)
            Path(html_path).write_text(page.content(), encoding="utf-8")
            summary["cookies"] = context.cookies(base_url)
            summary["responses"] = responses[:20]
            summary["status"] = "cloudflare_challenge" if summary["cloudflare_detected"] else "site_home"
            if not summary["cloudflare_detected"]:
                summary["register_candidates"] = collect_register_candidates(page)
                if click_register_entry:
                    summary["register_click_attempted"] = True
                    summary["register_result"] = inspect_register_entry(
                        page,
                        wait_seconds,
                        register_screenshot_path,
                        register_html_path,
                    )
        except Exception as exc:
            summary["status"] = "error"
            summary["error"] = str(exc)
            try:
                summary["final_url"] = page.url
            except Exception:
                pass
            try:
                summary["title"] = page.title() or ""
            except Exception:
                pass
            try:
                page.screenshot(path=screenshot_path, full_page=True)
            except Exception:
                pass
            try:
                Path(html_path).write_text(page.content(), encoding="utf-8")
            except Exception:
                pass
            summary["responses"] = responses[:20]
        finally:
            context.close()

    json.dump(summary, sys.stdout, ensure_ascii=False)


def read_payload():
    raw = sys.stdin.read().strip()
    if not raw:
        return {}
    return json.loads(raw)


def collect_register_candidates(page):
    try:
        return page.evaluate(
            """() => {
                const nodes = Array.from(document.querySelectorAll("a,button,[role='button']"));
                const out = [];
                for (const node of nodes) {
                    const text = (node.innerText || node.textContent || "").trim().replace(/\\s+/g, " ");
                    const href = node.getAttribute("href") || "";
                    const html = node.outerHTML.slice(0, 300);
                    const haystack = `${text} ${href}`.toLowerCase();
                    if (!haystack) continue;
                    if (!/(register|sign up|signup|create account|join|免费注册|注册)/i.test(haystack)) continue;
                    out.push({
                        text,
                        href,
                        tag: node.tagName.toLowerCase(),
                        visible: !!(node.offsetWidth || node.offsetHeight || node.getClientRects().length),
                        html,
                    });
                }
                return out.slice(0, 20);
            }"""
        )
    except Exception:
        return []


def inspect_register_entry(page, wait_seconds, screenshot_path, html_path):
    result = {
        "clicked": False,
        "selected_candidate": {},
        "final_url": page.url,
        "title": "",
        "body_preview": "",
        "cloudflare_detected": False,
        "screenshot_path": screenshot_path,
        "html_path": html_path,
        "error": "",
    }
    candidates = collect_register_candidates(page)
    if not candidates:
        result["error"] = "未找到注册入口候选元素"
        return result

    selected = pick_candidate(candidates)
    result["selected_candidate"] = selected
    selector = selector_for_candidate(selected)
    if not selector:
        result["error"] = "无法为注册入口候选元素生成选择器"
        return result

    try:
        locator = page.locator(selector).first
        locator.click(timeout=5000)
        result["clicked"] = True
        if wait_seconds > 0:
            page.wait_for_timeout(wait_seconds * 1000)
        result["final_url"] = page.url
        result["title"] = page.title() or ""
        body_text = page.evaluate("() => (document.body?.innerText || '').trim()")
        body_text = " ".join(body_text.split())
        result["body_preview"] = body_text[:1000]
        result["cloudflare_detected"] = is_cloudflare_text(
            "\n".join([result["title"], result["final_url"], body_text])
        )
        page.screenshot(path=screenshot_path, full_page=True)
        Path(html_path).write_text(page.content(), encoding="utf-8")
    except Exception as exc:
        result["error"] = str(exc)
        try:
            page.screenshot(path=screenshot_path, full_page=True)
        except Exception:
            pass
        try:
            Path(html_path).write_text(page.content(), encoding="utf-8")
        except Exception:
            pass
    return result


def pick_candidate(candidates):
    ranked = sorted(
        candidates,
        key=lambda item: (
            not item.get("visible", False),
            rank_text(item.get("text", ""), item.get("href", "")),
        ),
    )
    return ranked[0]


def rank_text(text, href):
    haystack = f"{text} {href}".lower()
    if "sign up" in haystack or "signup" in haystack:
        return 0
    if "register" in haystack or "注册" in haystack:
        return 1
    if "create account" in haystack:
        return 2
    if "join" in haystack:
        return 3
    return 9


def selector_for_candidate(candidate):
    href = (candidate.get("href") or "").strip()
    text = (candidate.get("text") or "").strip()
    tag = (candidate.get("tag") or "").strip()
    if href:
        return f'{tag}[href="{href}"]' if tag else f'[href="{href}"]'
    if text:
        escaped = text.replace("\\", "\\\\").replace('"', '\\"')
        if tag:
            return f'{tag}:has-text("{escaped}")'
        return f':has-text("{escaped}")'
    return ""


if __name__ == "__main__":
    main()
