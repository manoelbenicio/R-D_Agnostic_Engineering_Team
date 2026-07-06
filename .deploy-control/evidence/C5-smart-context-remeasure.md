# C5 Smart Context Remeasure Evidence

Timestamp: 2026-07-05T23:57:25Z

## Scope

- Created OpenAI chat-format payloads under `scripts/smoke/payloads/`.
- Created fake OpenAI-compatible upstream logger at `scripts/smoke/fake-upstream-logging.py`.
- Did not edit `prodex-sidecar/`.

## Files

- `scripts/smoke/payloads/chat-16kib.json`
- `scripts/smoke/payloads/chat-64kib.json`
- `scripts/smoke/fake-upstream-logging.py`

## Payload Validation

Command:

```bash
node - <<'NODE'
const fs = require('fs');
for (const file of ['scripts/smoke/payloads/chat-16kib.json', 'scripts/smoke/payloads/chat-64kib.json']) {
  const parsed = JSON.parse(fs.readFileSync(file, 'utf8'));
  console.log(`${file}: role=${parsed.messages[0].role} content_bytes=${Buffer.byteLength(parsed.messages[0].content, 'utf8')}`);
}
NODE
```

Output:

```text
scripts/smoke/payloads/chat-16kib.json: role=user content_bytes=16384
scripts/smoke/payloads/chat-64kib.json: role=user content_bytes=65536
```

Command:

```bash
wc -c scripts/smoke/payloads/chat-16kib.json scripts/smoke/payloads/chat-64kib.json
```

Output:

```text
16508 scripts/smoke/payloads/chat-16kib.json
65660 scripts/smoke/payloads/chat-64kib.json
82168 total
```

## Fake Upstream Validation

Command:

```bash
python3 -m py_compile scripts/smoke/fake-upstream-logging.py
```

Output: success, no stderr.

Server command:

```bash
python3 scripts/smoke/fake-upstream-logging.py --host 127.0.0.1 --port 18081
```

POST commands:

```bash
curl -sS -o /tmp/fake-upstream-16k-response.json -w '16KiB status=%{http_code} response_bytes=%{size_download}\n' -H 'Content-Type: application/json' --data-binary @scripts/smoke/payloads/chat-16kib.json http://127.0.0.1:18081/v1/chat/completions
curl -sS -o /tmp/fake-upstream-64k-response.json -w '64KiB status=%{http_code} response_bytes=%{size_download}\n' -H 'Content-Type: application/json' --data-binary @scripts/smoke/payloads/chat-64kib.json http://127.0.0.1:18081/v1/chat/completions
```

Client output:

```text
16KiB status=200 response_bytes=268
64KiB status=200 response_bytes=268
```

Fake upstream stdout:

```text
fake upstream listening on http://127.0.0.1:18081
POST /v1/chat/completions body_bytes=16508
POST /v1/chat/completions body_bytes=65660
```

## Checksums

```text
d243eb6236373d8a766d3b54dbd78c8b48f4eac09794221b799895107a5ea69f  scripts/smoke/payloads/chat-16kib.json
052b41595a9466f3ad9b0a213861901964978333c0ca8c9e0a515ceb251f3db6  scripts/smoke/payloads/chat-64kib.json
20cba33d34661fe4a00c1be083bf8008d4deca0b89100f41f896be10d46fc891  scripts/smoke/fake-upstream-logging.py
```
