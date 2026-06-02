import json
import sys


def main():
    json.dump(
        {
            "status": "pending_payment",
            "external_order_no": "",
            "subscribe_url": "",
            "error": "",
        },
        sys.stdout,
    )


if __name__ == "__main__":
    main()
