from playwright.sync_api import TimeoutError


def wait_page_ready(page, ready_selectors, loading_selectors):
    page.wait_for_load_state("domcontentloaded")

    for selector in ready_selectors:
        locator = page.locator(selector)
        if locator.count() > 0:
            locator.first.wait_for(state="visible", timeout=15000)
            break

    for selector in loading_selectors:
        locator = page.locator(selector)
        if locator.count() == 0:
            continue
        try:
            locator.first.wait_for(state="hidden", timeout=15000)
        except TimeoutError:
            pass


def wait_actionable(locator):
    locator.wait_for(state="visible", timeout=15000)
    if not locator.is_enabled():
        raise RuntimeError("locator is disabled")


def click_and_verify(locator, verify_fn):
    locator.click()
    if not verify_fn():
        raise RuntimeError("click verification failed")
