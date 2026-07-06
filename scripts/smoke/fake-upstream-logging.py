#!/usr/bin/env python3
"""Tiny OpenAI-compatible fake upstream that logs request body sizes."""

from __future__ import annotations

import argparse
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from time import time


class LoggingHandler(BaseHTTPRequestHandler):
    server_version = "FakeOpenAIUpstream/1.0"

    def do_GET(self) -> None:
        if self.path == "/health":
            self._send_json(200, {"status": "ok"})
            return
        self._send_json(404, {"error": {"message": "not found"}})

    def do_POST(self) -> None:
        length = int(self.headers.get("Content-Length", "0") or "0")
        body = self.rfile.read(length)
        print(f"{self.command} {self.path} body_bytes={len(body)}", flush=True)

        if self.path.endswith("/chat/completions"):
            self._send_json(
                200,
                {
                    "id": "chatcmpl-fake-smoke",
                    "object": "chat.completion",
                    "created": int(time()),
                    "model": "fake-upstream-logging",
                    "choices": [
                        {
                            "index": 0,
                            "message": {
                                "role": "assistant",
                                "content": "ok",
                            },
                            "finish_reason": "stop",
                        }
                    ],
                    "usage": {
                        "prompt_tokens": 0,
                        "completion_tokens": 1,
                        "total_tokens": 1,
                    },
                },
            )
            return

        if self.path.endswith("/v1/responses") or self.path.endswith("/responses"):
            self._send_json(
                200,
                {
                    "id": "resp-fake-smoke",
                    "object": "response",
                    "created_at": int(time()),
                    "model": "fake-upstream-logging",
                    "output": [
                        {
                            "type": "message",
                            "role": "assistant",
                            "content": [
                                {
                                    "type": "output_text",
                                    "text": "ok",
                                }
                            ],
                        }
                    ],
                    "usage": {
                        "input_tokens": 8,
                        "output_tokens": 1,
                        "total_tokens": 9,
                    },
                },
            )
            return

        self._send_json(404, {"error": {"message": "not found"}})

    def log_message(self, fmt: str, *args: object) -> None:
        return

    def _send_json(self, status: int, payload: dict[str, object]) -> None:
        raw = json.dumps(payload, separators=(",", ":")).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=18080)
    args = parser.parse_args()

    httpd = ThreadingHTTPServer((args.host, args.port), LoggingHandler)
    print(f"fake upstream listening on http://{args.host}:{args.port}", flush=True)
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("fake upstream stopped", flush=True)


if __name__ == "__main__":
    main()
