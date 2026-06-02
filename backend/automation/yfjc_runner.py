import json
import sys


def main():
    payload = json.load(sys.stdin)
    action = payload.get("action", "prepare_order")
    if action == "fetch_subscribe":
        response = {
            "status": "needs_manual_review",
            "external_order_no": payload.get("external_order_no", ""),
            "subscribe_url": "",
            "error": "fetch_subscribe 未接入真实站点流程，请在后台手动补录或后续接入自动抓取",
        }
    else:
        response = {
            "status": "pending_payment",
            "external_order_no": "",
            "subscribe_url": "",
            "error": "",
        }
    json.dump(
        response,
        sys.stdout,
    )


if __name__ == "__main__":
    main()
