# PLAN 00-03 Evidence - Foundation Reachability And Runtime Inventory

- executed_by: Codex#5.5
- executed_at_utc: 2026-07-05T02:36:26Z
- plan: .planning/phases/00-fundacao/00-03-PLAN.md
- secret_policy: no environment dumps; no raw DSNs, Redis URLs, passwords, tokens, or auth files recorded

## PLAN 00-01 Dependency Check

- attestation_path: /home/dataops-lab/runtime/prodex-src/attestations/prodex-build-20260705T022506Z.sha256
- expected_commit: 7750da9b6a5c91a6d429e18e6a4d422cab4bc144
- current_commit: 7750da9b6a5c91a6d429e18e6a4d422cab4bc144
- binary_path: /home/dataops-lab/runtime/prodex-src/target/release/prodex
- attested_binary_sha256: 5568ae664e2fa5b776a9e2df813175e57a24dc31c4c1dfbeb029a2d3db8e7758
- current_binary_sha256: 5568ae664e2fa5b776a9e2df813175e57a24dc31c4c1dfbeb029a2d3db8e7758
- verdict: PASS - PLAN 00-01 attestation exists and matches the current pinned binary.

## Postgres Reachability

- target: fleet-host container deploy-postgres-1 loopback port 5432
- command: docker exec deploy-postgres-1 pg_isready -h 127.0.0.1 -p 5432 -U multica -d multica
- result: PASS - 127.0.0.1:5432 accepted connections
- credential_disclosure: none

## Redis Reachability

- target: fleet-host container deploy-redis-1 loopback port 6379
- command: docker exec deploy-redis-1 sh -c 'redis-cli -h 127.0.0.1 -p 6379 -a "$REDIS_PASSWORD" --no-auth-warning ping'
- result: PASS - PONG
- credential_disclosure: none; password was consumed inside the container environment and not printed
- note: unauthenticated ping is rejected with NOAUTH, confirming auth is enforced

## Go Container Gate

- workdir: /src/server
- source_mount: /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work:/src
- image: golang:1.26-alpine
- cache_volumes:
  - go-mod-cache-00-03:/go/pkg/mod
  - go-build-cache-00-03:/root/.cache/go-build
- ipv6_mode: disabled with net.ipv6.conf.all.disable_ipv6=1 and net.ipv6.conf.default.disable_ipv6=1
- command: docker run --rm --network bridge --add-host proxy.golang.org:172.217.30.49 --add-host sum.golang.org:172.217.162.177 --sysctl net.ipv6.conf.all.disable_ipv6=1 --sysctl net.ipv6.conf.default.disable_ipv6=1 -e HOME=/tmp/root-home -v /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work:/src -v go-mod-cache-00-03:/go/pkg/mod -v go-build-cache-00-03:/root/.cache/go-build -w /src/server golang:1.26-alpine sh -c 'apk add --no-cache git >/tmp/apk-add-git.log && mkdir -p "$HOME" && go build ./... && go vet ./internal/... && go test ./...'
- result: PASS - exit 0
- green_scope:
  - go build ./...
  - go vet ./internal/...
  - go test ./...
- environment_adjustments:
  - git installed inside the ephemeral Alpine container because daemon repo-cache tests require git
  - HOME set to /tmp/root-home so root-owned container tests do not classify /root as both home and protected system root
  - proxy.golang.org and sum.golang.org pinned to IPv4 add-host entries after the default Go resolver attempted an IPv6 address while IPv6 was intentionally disabled

## Prodex Runtime Inventory

- binary: /home/dataops-lab/runtime/prodex-src/target/release/prodex
- version: prodex 0.246.0
- command: /home/dataops-lab/runtime/prodex-src/target/release/prodex --help
- top_level_commands:
  - profile
  - use
  - current
  - info
  - log
  - session
  - doctor
  - setup
  - capability
  - audit
  - context
  - cleanup
  - presidio
  - login
  - logout
  - update
  - quota
  - redeem
  - ping
  - dashboard
  - run
  - caveman
  - rtk
  - sqz
  - tokensavior
  - clawcompactor
  - ponytail
  - mem
  - super
  - expose
  - app-server-broker
  - gateway
  - claude
  - help
- profile_subcommands:
  - add
  - export
  - import
  - import-current
  - list
  - remove
  - use
  - help
- capability_subcommands:
  - list
  - super-doctor
  - help
- verdict: PASS - pinned Prodex binary is runnable and command inventory captured from local help output.

## Final Verdict

PLAN 00-03 is DONE:
- PLAN 00-01 pin and SHA-256 attestation matched the current Prodex binary.
- Postgres reachability passed without credential disclosure.
- Redis reachability passed without credential disclosure.
- Go build/vet/test gate passed green in container with IPv6 disabled.
- Prodex subcommand inventory was captured from the pinned binary.
