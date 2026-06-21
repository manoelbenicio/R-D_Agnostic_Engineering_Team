#!/bin/bash
# ═══════════════════════════════════════════════════════════════════
# AgentVerse Sprint 3 — Production Dashboard v2.0
# Fortune 500 Grade — Real-time Monitoring
# ═══════════════════════════════════════════════════════════════════
PROJECT="/mnt/p/Automonous_Agentic"
LEDGER="$PROJECT/.planning/AGENT_LEDGER_S3.md"
CACHE="/tmp/dash_cao_files.txt"
SPRINT_START="2026-06-21T08:20:00"
REFRESH=30

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
GRAY='\033[0;90m'
BOLD='\033[1m'
NC='\033[0m'

progress_bar() {
  local pct=$1 width=${2:-30} label=$3
  local filled=$((pct * width / 100))
  local empty=$((width - filled))
  local color=$GREEN
  [ "$pct" -lt 50 ] && color=$RED
  [ "$pct" -ge 50 ] && [ "$pct" -lt 80 ] && color=$YELLOW
  printf "${color}"
  for ((i=0; i<filled; i++)); do printf "█"; done
  printf "${GRAY}"
  for ((i=0; i<empty; i++)); do printf "░"; done
  printf "${NC} ${WHITE}%3d%%${NC}" "$pct"
}

elapsed_since() {
  local start_epoch=$(date -d "$1" +%s 2>/dev/null || echo 0)
  local now_epoch=$(date +%s)
  local diff=$((now_epoch - start_epoch))
  local hours=$((diff / 3600))
  local mins=$(( (diff % 3600) / 60 ))
  printf "%dh%02dm" $hours $mins
}

agent_status() {
  local agent=$1 ledger_id=$2
  local last_action=$(grep "$ledger_id" "$LEDGER" 2>/dev/null | tail -1)
  local status="🟢 IDLE"
  local file="-"
  local action_time=""

  if echo "$last_action" | grep -q "IN PROGRESS"; then
    status="🟡 ATIVO"
    file=$(echo "$last_action" | grep -oP 'src/[^ |]+' | head -1)
    action_time=$(echo "$last_action" | grep -oP '\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}' | head -1)
  elif echo "$last_action" | grep -q "CHECK-OUT.*DONE"; then
    status="🟢 DONE"
    file=$(echo "$last_action" | grep -oP 'src/[^ |]+' | head -1)
  fi

  local done_count=$(grep "$ledger_id.*CHECK-OUT.*DONE" "$LEDGER" 2>/dev/null | wc -l)

  printf "  ${WHITE}%-18s${NC}" "$agent"
  printf " %s" "$status"
  printf "  ${CYAN}%-40s${NC}" "$file"
  printf "  ${GRAY}(%d done)${NC}" "$done_count"
  echo ""
}

echo -e "${YELLOW}⏳ Coletando dados do projeto...${NC}"

while true; do
  # Data collection
  grep -rl "caoClient\|CaoClient" "$PROJECT/src" \
    --include="*.ts" --include="*.tsx" 2>/dev/null \
    | grep -v "__tests__" | grep -v "go-core-client.ts" \
    | grep -v "cao-client.ts" | grep -v "index.ts" \
    | sed "s|$PROJECT/||" | sort > "$CACHE" 2>/dev/null

  REMAINING=$(wc -l < "$CACHE" 2>/dev/null || echo 0)
  REMAINING=$(echo $REMAINING | tr -d ' ')
  TOTAL_MIGRATION=19
  DONE_MIGRATION=$((TOTAL_MIGRATION - REMAINING))
  [ "$DONE_MIGRATION" -lt 0 ] && DONE_MIGRATION=0
  PCT_MIGRATION=$((DONE_MIGRATION * 100 / TOTAL_MIGRATION))

  TOTAL_CHECKOUTS=$(grep -c "CHECK-OUT.*DONE" "$LEDGER" 2>/dev/null || echo 0)
  TOTAL_CHECKINS=$(grep -c "CHECK-IN.*IN PROGRESS" "$LEDGER" 2>/dev/null || echo 0)

  NOW=$(date +"%Y-%m-%d %H:%M:%S")
  ELAPSED=$(elapsed_since "2026-06-21 08:20:00")

  # TSC status (check if last run was clean)
  TSC_LABEL="⏳ não verificado"

  # GATE status
  GATE1="❌ PENDENTE"
  [ "$REMAINING" -eq 0 ] && GATE1="✅ APROVADO"

  clear

  echo -e "${BOLD}${CYAN}"
  echo "  ╔═══════════════════════════════════════════════════════════════════════╗"
  echo "  ║                                                                     ║"
  echo "  ║   █▀█ █▀▀ █▀▀ █▄░█ ▀█▀ █░█ █▀▀ █▀█ █▀ █▀▀                        ║"
  echo "  ║   █▀█ █▄█ ██▄ █░▀█ ░█░ ▀▄▀ ██▄ █▀▄ ▄█ ██▄                        ║"
  echo "  ║                                                                     ║"
  echo "  ║   SPRINT 3 — PRE-PRODUCTION DASHBOARD        $NOW   ║"
  echo "  ║   Fortune 500 Enterprise Grade | GO Core Migration                 ║"
  echo "  ║   Elapsed: $ELAPSED | Agents: 7 | Orquestrador: opus46             ║"
  echo "  ║                                                                     ║"
  echo -e "  ╠═══════════════════════════════════════════════════════════════════════╣${NC}"

  # Overall progress
  echo -e "  ║                                                                     ║"
  printf "  ║  ${BOLD}CRIT-003 GO Core Migration${NC}   "
  progress_bar $PCT_MIGRATION 30
  printf "  ${WHITE}%d/%d${NC}\n" $DONE_MIGRATION $TOTAL_MIGRATION
  echo -e "  ║  ${GRAY}Arquivos pendentes: $REMAINING | CHECK-OUTs: $TOTAL_CHECKOUTS | TypeScript: $TSC_LABEL${NC}"
  echo -e "  ║                                                                     ║"

  echo -e "  ${CYAN}╠═══════════════════════════════════════════════════════════════════════╣${NC}"
  echo -e "  ║  ${BOLD}${WHITE}AGENTES — STATUS EM TEMPO REAL${NC}                                       ║"
  echo -e "  ${CYAN}╠═══════════════════════════════════════════════════════════════════════╣${NC}"

  # Per-agent status
  agent_status "nemotron_ultra" "nemotron_ultra"
  agent_status "codex#1 (CX1)" "CX1"
  agent_status "codex#2 (CX2)" "CX2"
  agent_status "Codex#3 (CX3)" "CX3"
  agent_status "Codex#4 (OP1)" "OP1"
  agent_status "codex#5 (CX5)" "CX5"
  agent_status "nemotron_ultra#2" "NM2\|nemotron_ultra2"

  echo -e ""
  echo -e "  ${CYAN}╠═══════════════════════════════════════════════════════════════════════╣${NC}"
  echo -e "  ║  ${BOLD}${WHITE}QUALITY GATES${NC}                                                        ║"
  echo -e "  ${CYAN}╠═══════════════════════════════════════════════════════════════════════╣${NC}"

  if [ "$REMAINING" -eq 0 ]; then
    echo -e "  ${GREEN}  ✅ GATE 1 — Zero caoClient refs          APROVADO${NC}"
  else
    echo -e "  ${RED}  ❌ GATE 1 — Zero caoClient refs          $REMAINING arquivos restantes${NC}"
  fi
  echo -e "  ${YELLOW}  ⏳ GATE 2 — Build (npm run build)        Aguarda GATE 1${NC}"
  echo -e "  ${YELLOW}  ⏳ GATE 3 — Tests (vitest run)           Aguarda GATE 2${NC}"
  echo -e "  ${YELLOW}  ⏳ GATE 4 — Lint (eslint)                Aguarda GATE 2${NC}"
  echo -e "  ${YELLOW}  ⏳ GATE 5 — E2E (playwright)             Aguarda GO Core server${NC}"
  echo -e "  ${YELLOW}  ⏳ GATE 6 — QA Audit Final               Aguarda GATE 2-5${NC}"
  echo -e "  ${YELLOW}  ⏳ GATE 7 — Security Review              Aguarda GATE 6${NC}"

  echo -e ""
  echo -e "  ${CYAN}╠═══════════════════════════════════════════════════════════════════════╣${NC}"
  echo -e "  ║  ${BOLD}${WHITE}ARQUIVOS PENDENTES (CRIT-003)${NC}                                        ║"
  echo -e "  ${CYAN}╠═══════════════════════════════════════════════════════════════════════╣${NC}"

  if [ "$REMAINING" -eq 0 ]; then
    echo -e "  ${GREEN}  ✅ ZERO PENDÊNCIAS — GATE 1 PRONTO PARA APROVAÇÃO!${NC}"
  else
    while IFS= read -r f; do
      [ -z "$f" ] && continue
      printf "  ${RED}  ⬡${NC}  ${WHITE}%-60s${NC}\n" "$f"
    done < "$CACHE"
  fi

  echo -e ""
  echo -e "  ${CYAN}╠═══════════════════════════════════════════════════════════════════════╣${NC}"
  echo -e "  ║  ${BOLD}${WHITE}BACKLOG PÓS-GATE 1${NC}                                                  ║"
  echo -e "  ${CYAN}╠═══════════════════════════════════════════════════════════════════════╣${NC}"
  echo -e "  ${GRAY}  ┌─ GATE 2: Build${NC}"
  echo -e "  ${GRAY}  │  └─ npm run build (tsc -b && vite build)${NC}"
  echo -e "  ${GRAY}  ├─ GATE 3: Unit Tests${NC}"
  echo -e "  ${GRAY}  │  └─ npx vitest run --reporter=verbose${NC}"
  echo -e "  ${GRAY}  ├─ GATE 4: Lint${NC}"
  echo -e "  ${GRAY}  │  └─ npx eslint src/ --max-warnings 0${NC}"
  echo -e "  ${GRAY}  ├─ GATE 5: E2E Tests${NC}"
  echo -e "  ${GRAY}  │  └─ npx playwright test (requer GO Core server :8080)${NC}"
  echo -e "  ${GRAY}  ├─ GATE 6: QA Audit Final${NC}"
  echo -e "  ${GRAY}  │  ├─ Varredura: zero imports legados em prod${NC}"
  echo -e "  ${GRAY}  │  ├─ Varredura: zero env vars CAO_ em uso${NC}"
  echo -e "  ${GRAY}  │  ├─ Varredura: contract tests cobertura 100%${NC}"
  echo -e "  ${GRAY}  │  ├─ Review: keystore migration schema v4${NC}"
  echo -e "  ${GRAY}  │  └─ Review: backward compat aliases removíveis${NC}"
  echo -e "  ${GRAY}  ├─ GATE 7: Security Review${NC}"
  echo -e "  ${GRAY}  │  ├─ npm audit (vulnerabilidades)${NC}"
  echo -e "  ${GRAY}  │  ├─ Secrets scan (.env, hardcoded keys)${NC}"
  echo -e "  ${GRAY}  │  └─ CORS/CSP headers no GO Core${NC}"
  echo -e "  ${GRAY}  └─ RELEASE${NC}"
  echo -e "  ${GRAY}     ├─ git add -A && git commit (orquestrador)${NC}"
  echo -e "  ${GRAY}     ├─ Tag: v1.0.0-rc1${NC}"
  echo -e "  ${GRAY}     └─ Deploy staging${NC}"

  echo -e ""
  echo -e "  ${CYAN}╠═══════════════════════════════════════════════════════════════════════╣${NC}"
  echo -e "  ║  ${BOLD}${WHITE}ÚLTIMAS ATIVIDADES${NC}                                                  ║"
  echo -e "  ${CYAN}╠═══════════════════════════════════════════════════════════════════════╣${NC}"

  grep "CHECK-OUT.*DONE\|CHECK-IN.*IN PROGRESS" "$LEDGER" 2>/dev/null | tail -8 | while IFS= read -r line; do
    TS=$(echo "$line" | grep -oP '\d{2}:\d{2}:\d{2}' | head -1)
    AGENT=$(echo "$line" | awk -F'|' '{print $3}' | tr -d ' ' | head -c12)
    ACTION=$(echo "$line" | grep -oP 'CHECK-(IN|OUT)')
    FILE=$(echo "$line" | grep -oP 'src/[^ |]+' | head -1 | sed 's|.*/||')
    STATUS=$(echo "$line" | grep -oP 'DONE|IN PROGRESS')

    if [ "$STATUS" = "DONE" ]; then
      printf "  ${GREEN}  ✓ ${GRAY}%s${NC} ${WHITE}%-12s${NC} ${GREEN}%-10s${NC} %-20s ${GREEN}%s${NC}\n" \
        "$TS" "$AGENT" "$ACTION" "$FILE" "$STATUS"
    else
      printf "  ${YELLOW}  ◉ ${GRAY}%s${NC} ${WHITE}%-12s${NC} ${YELLOW}%-10s${NC} %-20s ${YELLOW}%s${NC}\n" \
        "$TS" "$AGENT" "$ACTION" "$FILE" "$STATUS"
    fi
  done

  echo -e ""
  echo -e "  ${CYAN}╠═══════════════════════════════════════════════════════════════════════╣${NC}"
  echo -e "  ║  ${BOLD}${WHITE}BLOCKERS${NC}                                                             ║"
  echo -e "  ${CYAN}╠═══════════════════════════════════════════════════════════════════════╣${NC}"
  echo -e "  ${YELLOW}  ⚠️  B1: node_modules @rollup — workaround manual (cp) ativo${NC}"
  echo -e "  ${RED}  🔴 B2: GO Core server não rodando — bloqueia GATE 5 (E2E)${NC}"
  echo -e "  ${YELLOW}  ⚠️  B3: Firebase auth.ts — 3 erros TS (adiado, não é Sprint 3)${NC}"
  echo -e "  ${YELLOW}  ⚠️  B4: npm install falha em /mnt/p/ (symlink EPERM)${NC}"

  echo -e ""
  echo -e "  ${CYAN}╚═══════════════════════════════════════════════════════════════════════╝${NC}"
  echo -e ""
  echo -e "  ${GRAY}Auto-refresh: ${REFRESH}s | Ctrl+C para sair | v2.0 Production Grade${NC}"

  sleep $REFRESH
done
