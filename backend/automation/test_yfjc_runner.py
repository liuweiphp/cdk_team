import os
import sys
import unittest


sys.path.insert(0, os.path.dirname(__file__))

import yfjc_runner


class FakePage:
    def __init__(self, url):
        self.url = url


class SelectReusablePageTests(unittest.TestCase):
    def test_prefers_page_matching_base_url_host(self):
        pages = [
            FakePage("about:blank"),
            FakePage("https://example.com/"),
            FakePage("https://www.yfjc.xyz/#/dashboard"),
        ]

        selected = yfjc_runner.select_reusable_page(pages, "https://www.yfjc.xyz")

        self.assertIs(selected, pages[2])

    def test_falls_back_to_last_page_when_host_not_found(self):
        pages = [
            FakePage("about:blank"),
            FakePage("https://example.com/a"),
            FakePage("https://example.com/b"),
        ]

        selected = yfjc_runner.select_reusable_page(pages, "https://www.yfjc.xyz")

        self.assertIs(selected, pages[2])


class ErrorResponseTests(unittest.TestCase):
    def test_non_cloudflare_error_keeps_debug_artifact_paths(self):
        result = yfjc_runner.error_response(
            {"external_order_no": "ORD-1"},
            "register failed | debug=/tmp/a.png,/tmp/a.html",
        )

        self.assertEqual("needs_manual_review", result["status"])
        self.assertEqual("/tmp/a.png", result["screenshot_path"])
        self.assertEqual("/tmp/a.html", result["html_dump_path"])


if __name__ == "__main__":
    unittest.main()
