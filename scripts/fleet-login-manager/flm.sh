#!/usr/bin/env bash
# =============================================================================
# Fleet Login Manager (flm)
# -----------------------------------------------------------------------------
# Elimina de vez o "trocar login na mao": cada worker do fleet tem seu HOME/
# config ISOLADO por vendor. O login de um NUNCA sobrescreve o do outro.
#
# Base tecnica: espelha o mapa de isolamento do PRODUTO (multica execenv):
#   cada vendor -> sua "native isolation lever" (env var):
#     codex        -> CODEX_HOME
#     agy/antigrav -> HOME (isolado)
#     kiro         -> HOME (isolado, Amazon Q data home)
#     opencode     -> XDG_CONFIG_HOME / OPENCODE_CONFIG
#
# Comandos:
#   flm.sh setup            cria os homes isolados (idempotente, nao-destrutivo)
#   flm.sh status           mostra worker -> home -> conta (account_id) + clobber
#   flm.sh doctor           detecta homes compartilhados / contas duplicadas / auth ausente/expirado
#   flm.sh launch <worker>  (re)lanca o worker no pane com env isolado + cwd correto
#   flm.sh launch-all       lanca todos
#
# EDITE o mapa WORKERS abaixo para o seu fleet.
# =============================================================================
set -uo pipefail

REPO="${FLM_REPO:-/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team}"

# --- MAPA DO FLEET: "worker_id  vendor  pane  home_dir  cli_cmd" -----------------
WORKERS=(
  "CodexA  codex  w3:pJ  $HOME/.codex-a  codex"
  "CodexB  codex  w3:pM  $HOME/.codex-b  codex"
  "CodexC  codex  w3:pK  $HOME/.codex-c  codex"
  "CodexD  codex  w3:p9  $HOME/.codex-d  codex"
)

# vendor -> env var de isolamento (native isolation lever)
vendor_env() {
  case "$1" in
    codex)            echo "CODEX_HOME" ;;
    agy|antigravity)  echo "HOME" ;;
    kiro)             echo "HOME" ;;
    opencode)         echo "XDG_CONFIG_HOME" ;;
    *)                echo "" ;;
  esac
}
# vendor -> arquivo de auth dentro do home (para status/doctor)
vendor_authfile() {
  case "$1" in
    codex) echo "auth.json" ;;
    *)     echo "auth.json" ;;
  esac
}

acct_id() {
  python3 - "$1" <<'PY' 2>/dev/null || echo "?"
import json,sys
try:
    d=json.load(open(sys.argv[1]))
    print(d.get("account_id") or (d.get("tokens",{}) or {}).get("account_id") or "?")
except Exception:
    print("?")
PY
}

cmd_setup() {
  echo "[flm setup] criando homes isolados (idempotente)"
  for w in "${WORKERS[@]}"; do
    set -- $w; id=$1 vendor=$2 pane=$3 home=$4
    mkdir -p "$home" && echo "  ok  $id ($vendor) -> $home"
  done
  echo "[flm setup] pronto. Rode 'flm.sh status' e faca login 1x por worker (flm.sh launch <id>)."
}

cmd_status() {
  printf "%-7s %-9s %-7s %-26s %-14s %s\n" WORKER VENDOR PANE HOME ACCOUNT NOTA
  declare -A seen
  for w in "${WORKERS[@]}"; do
    set -- $w; id=$1 vendor=$2 pane=$3 home=$4
    af="$home/$(vendor_authfile "$vendor")"; acc="(sem login)"; nota="ok"
    if [ -f "$af" ]; then acc="$(acct_id "$af")"; fi
    if [ "$acc" != "(sem login)" ] && [ "$acc" != "?" ]; then
      if [ -n "${seen[$acc]:-}" ]; then nota="DUPLICADA -> clobber!"; else seen[$acc]=1; fi
    elif [ "$acc" = "(sem login)" ]; then nota="faca login: flm.sh launch $id"; fi
    printf "%-7s %-9s %-7s %-26s %-14s %s\n" "$id" "$vendor" "$pane" "${home/#$HOME/~}" "${acc:0:12}" "$nota"
  done
}

cmd_doctor() {
  echo "[flm doctor]"; local bad=0
  declare -A seen
  for w in "${WORKERS[@]}"; do
    set -- $w; id=$1 vendor=$2 pane=$3 home=$4
    env=$(vendor_env "$vendor")
    [ -z "$env" ] && { echo "  ! $id: vendor '$vendor' sem lever de isolamento definido"; bad=1; continue; }
    af="$home/$(vendor_authfile "$vendor")"
    if [ ! -d "$home" ]; then echo "  ! $id: home ausente ($home) -> rode setup"; bad=1; fi
    if [ ! -f "$af" ]; then echo "  ! $id: sem login em $home -> flm.sh launch $id"; bad=1; continue; fi
    acc="$(acct_id "$af")"
    if [ -n "${seen[$acc]:-}" ]; then echo "  !! $id: conta $acc DUPLICADA (clobber com outro worker)"; bad=1; else seen[$acc]=1; fi
  done
  [ "$bad" = 0 ] && echo "  OK: todos isolados, contas distintas, sem clobber."
  return $bad
}

launch_one() {
  local id=$1 vendor=$2 pane=$3 home=$4 cli=$5
  local env; env=$(vendor_env "$vendor")
  [ -z "$env" ] && { echo "  ! $id: vendor '$vendor' sem lever"; return 1; }
  # (re)lanca no pane: cwd correto + env isolado + cli
  herdr pane run "$pane" "cd '$REPO' 2>/dev/null; export $env='$home'; $cli" >/dev/null 2>&1 \
    && echo "  launched $id -> pane $pane ($env=$home, cwd=$REPO)" \
    || echo "  ! falha ao lancar $id (pane $pane)"
}

cmd_launch() {
  local target="${1:-}"; [ -z "$target" ] && { echo "uso: flm.sh launch <worker_id>"; exit 2; }
  for w in "${WORKERS[@]}"; do
    set -- $w
    [ "$1" = "$target" ] && { launch_one "$1" "$2" "$3" "$4" "$5"; return $?; }
  done
  echo "worker '$target' nao esta no mapa"; exit 2
}

cmd_launch_all() {
  for w in "${WORKERS[@]}"; do set -- $w; launch_one "$1" "$2" "$3" "$4" "$5"; done
}

case "${1:-status}" in
  setup)       cmd_setup ;;
  status)      cmd_status ;;
  doctor)      cmd_doctor ;;
  launch)      shift; cmd_launch "$@" ;;
  launch-all)  cmd_launch_all ;;
  *) echo "uso: flm.sh {setup|status|doctor|launch <id>|launch-all}"; exit 2 ;;
esac
