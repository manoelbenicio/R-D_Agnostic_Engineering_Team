# P0 — Fontes de Dependência (de ONDE o time baixa cada coisa)

> Regulariza o furo: origens explícitas + cache + estratégia offline. **IPv6 SEMPRE desabilitado**
> (`docker run --sysctl net.ipv6.conf.all.disable_ipv6=1`), pois o IPv6 do host está quebrado.

## 1. Source do prodex (código)
- **Origem oficial:** `https://github.com/christiandoxa/prodex` (Apache-2.0). Alt: pacote npm oficial.
- **Pin obrigatório:** commit `7750da9b6a5c91a6d429e18e6a4d422cab4bc144` (v0.246.0).
- **Onde já está:** cópia em `/tmp/prodex-audit-7750da9` (efêmero) → **estabilizar** em `~/runtime/prodex-src`.
- **Re-obter do zero (se /tmp sumir):**
  ```
  git clone https://github.com/christiandoxa/prodex ~/runtime/prodex-src
  git -C ~/runtime/prodex-src checkout 7750da9b6a5c91a6d429e18e6a4d422cab4bc144
  ```
- Verificar: `git rev-parse HEAD` == `7750da9b...`.

## 2. Toolchain Rust
- **NÃO instalar no host.** Usar imagem docker pinada: **`rust:1-bookworm`** (glibc).
- Alt para binário portável (sem libc dep): target `x86_64-unknown-linux-musl` (avaliar se as deps do prodex compilam em musl).

## 3. Dependências Rust (crates)
- **Registry:** crates.io padrão — índice `index.crates.io`, download `static.crates.io`.
- **Resolução:** travada pelo **`Cargo.lock`** (105 KB, já presente no source) → versões determinísticas.
- **Cache persistente:** volume docker **`prodex-cargo`** montado em `/usr/local/cargo/registry`.
- **Comando (IPv4):**
  ```
  docker run --rm --sysctl net.ipv6.conf.all.disable_ipv6=1 \
    -v ~/runtime/prodex-src:/src -v prodex-cargo:/usr/local/cargo/registry \
    -w /src rust:1-bookworm bash -lc 'cargo build --release'
  ```
- **Offline (após 1º build):** `cargo build --release --offline` (usa só o cache).
- ⚠️ **Deps de sistema** (se o build reclamar): instalar no container via `apt-get` — candidatos comuns:
  `pkg-config libssl-dev cmake protobuf-compiler`. Se faltar algo não-listado → capturar e escalar (não inventar).

## 4. Dependências Go (Multica server)
- **Proxy:** `GOPROXY=https://proxy.golang.org,direct` (padrão); checksums via `sum.golang.org`.
- **Resolução:** `go.mod`/`go.sum` do `server/`.
- **Cache persistente:** volume docker **`multica-gomod`** em `/go/pkg/mod`.
- **Toolchain:** imagem **`golang:1.26-alpine`** (+ `apk add git`).
- **Comando (IPv4):**
  ```
  docker run --rm --sysctl net.ipv6.conf.all.disable_ipv6=1 \
    -v .../multica-auth-work:/src -v multica-gomod:/go/pkg/mod \
    -w /src/server golang:1.26-alpine sh -c \
    'apk add --no-cache git; export HOME=/tmp/gohome; mkdir -p $HOME; go mod download all && go build ./...'
  ```
- **Offline (após cache):** `GOFLAGS=-mod=mod GOPROXY=off go test ./...`.

## 5. Datastores (já provisionados — docker)
- Postgres: `pgvector/pgvector:pg17` (`:5432`, healthy). Redis: `redis` (`:6379`, healthy).

## 6. Ordem de provisionamento (resumo)
1. Estabilizar source (§1) → 2. build Rust em container (§2/§3) → 3. verificar pin+hash → 4. cache Go (§4) → 5. datastores (§5, já ok).

## Nota de auditoria
Tudo determinístico: prodex por commit; crates por `Cargo.lock`; Go por `go.sum`. Caches em volumes docker
nomeados (`prodex-cargo`, `multica-gomod`) permitem builds reprodutíveis e offline após o 1º.
