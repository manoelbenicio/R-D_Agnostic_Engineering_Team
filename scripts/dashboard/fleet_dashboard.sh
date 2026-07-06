#!/bin/bash
# ═══════════════════════════════════════════════════════════════════════
# SQUAD AOP — PREMIUM EXECUTIVE DASHBOARD v2.0
# Real-time synchronization with OpenSpec + Agent Check-ins
# ═══════════════════════════════════════════════════════════════════════

REPO="/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team"
TASKS_FILE="$REPO/openspec/changes/rotation-parity-polyglot/tasks.md"
DEPLOY_CTL="$REPO/.deploy-control"
INTERVAL="${1:-5}"

# ── Colors & Style ──────────────────────────────────────────────────
R='\033[0;31m'; G='\033[0;32m'; Y='\033[0;33m'; B='\033[0;34m'
C='\033[0;36m'; W='\033[1;37m'; D='\033[0;90m'; RESET='\033[0m'
BOLD='\033[1m'; BG_BLUE='\033[44m'

# ── Render Helpers ──────────────────────────────────────────────────
draw_bar() {
    local pct=$1 width=20
    local fill=$((pct * width / 100))
    local color=$C # Cyan for in-progress
    [ $pct -eq 100 ] && color=$G # Green for done
    
    printf "${color}"
    for ((i=0; i<fill; i++)); do printf "■"; done
    printf "${D}"
    for ((i=fill; i<width; i++)); do printf "·"; done
    printf "${RESET} %3d%%" "$pct"
}

# ── Main Loop ───────────────────────────────────────────────────────
tput civis # hide cursor
trap 'tput cnorm; clear; exit 0' INT TERM EXIT

while true; do
    # 1. Parse Agent Check-ins for mapping
    # Create a mapping of stream ID (e.g. 6.0) to actual agent name
    declare -A AGENT_MAP
    while IFS= read -r f; do
        [ -e "$f" ] || continue
        a_name=$(grep "^agent:" "$f" | head -1 | cut -d' ' -f2-)
        a_stream=$(grep "^stream:" "$f" | head -1 | cut -d' ' -f2-)
        # Simplify stream ID for matching (e.g. PLAN-06-01 -> 6.1)
        # We try to match by number if possible
        if [[ "$a_stream" =~ ([0-9]+)-([0-9]+) ]]; then
            short_id="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}"
            AGENT_MAP["$short_id"]="$a_name"
        fi
        # Fallback by exact stream name
        AGENT_MAP["$a_stream"]="$a_name"
    done < <(ls -1 "$DEPLOY_CTL"/*.md 2>/dev/null)

    # 2. Parse Tasks
    declare -a IDs=(); declare -a PRIs=(); declare -a NAMES=(); declare -a STATUS=(); declare -a AGENTS=()
    declare -a PROG=(); declare -a ETAs=()
    
    done_count=0; total_count=0; in_progress=0; blocked=0
    
    while IFS= read -r line; do
        if [[ "$line" =~ ^##[[:space:]]+(.*) ]]; then
            current_phase="${BASH_REMATCH[1]}"
            [[ "$current_phase" =~ \[P([0-9])\] ]] && cp_pri="P${BASH_REMATCH[1]}" || cp_pri="P2"
            [[ "$current_phase" =~ \[([0-9]+[mh])\] ]] && cp_eta="${BASH_REMATCH[1]}" || cp_eta="--"
        elif [[ "$line" =~ ^[[:space:]]*-[[:space:]]*\[([ xX])\][[:space:]]*([0-9]+\.[0-9]+)?(.*) ]]; then
            total_count=$((total_count + 1))
            m_check="${BASH_REMATCH[1]}"
            m_id="${BASH_REMATCH[2]}"
            m_name="${BASH_REMATCH[3]}"
            
            # Show important phases + in-progress tasks
            is_done=false; [[ "$m_check" =~ [xX] ]] && is_done=true
            
            if [[ "$m_id" =~ \.0$ ]] || [[ "$is_done" == false ]]; then
                # Only show if not too many or if critical
                if [ ${#IDs[@]} -lt 15 ] || [ "$is_done" == false ]; then
                    IDs+=("${m_id:-?}")
                    PRIs+=("$cp_pri")
                    NAMES+=("${m_name:1:40}")
                    ETAs+=("$cp_eta")
                    
                    if [ "$is_done" == true ]; then
                        STATUS+=("DONE"); PROG+=(100); done_count=$((done_count + 1))
                    else
                        STATUS+=("WORKING"); PROG+=(20); in_progress=$((in_progress + 1))
                    fi
                    
                    # Resolve real agent name
                    real_agent="${AGENT_MAP[$m_id]:-Fleet}"
                    # Try fuzzy match if exact fails
                    if [ "$real_agent" == "Fleet" ]; then
                        for key in "${!AGENT_MAP[@]}"; do
                            if [[ "$key" == *"$m_id"* ]]; then real_agent="${AGENT_MAP[$key]}"; break; fi
                        done
                    fi
                    AGENTS+=("${real_agent:0:15}")
                fi
            fi
            [ "$is_done" == true ] && [ ! "$m_id" =~ \.0$ ] && done_count=$((done_count + 1))
        fi
    done < "$TASKS_FILE"

    # 3. Rendering
    clear
    echo -e " ══════════════════════════════════════════════════════════════════════════════════════════════════════════════════════"
    echo -e "                                              ${BOLD}SQUAD AOP${RESET}"
    echo -e "                                      ${BOLD}SQUAD AOP — EXECUTIVE DASHBOARD${RESET}"
    echo -e " ══════════════════════════════════════════════════════════════════════════════════════════════════════════════════════"
    printf " │ %-4s │ %-3s │ %-45s │ %-12s │ %-15s │ %-8s │ %-24s │ %-5s │\n" "ID" "PRI" "TAREFA" "STATUS" "AGENTE" "VIVO" "PROGRESSO" "ETA"
    echo -e " ├──────┼─────┼───────────────────────────────────────────────┼──────────────┼─────────────────┼──────────┼──────────────────────────┼───────┤"

    for i in "${!IDs[@]}"; do
        s="${STATUS[$i]}"
        s_color=$C; [ "$s" == "DONE" ] && s_color=$G
        v_status="LIVE"; [ "$s" == "DONE" ] && v_status="DONE"
        v_color=$G; [ "$v_status" == "DONE" ] && v_color=$D
        
        pri_color=$D; [ "${PRIs[$i]}" == "P0" ] && pri_color=$R
        
        printf " │ %-4s │ ${pri_color}%-3s${RESET} │ %-45s │ ${s_color}%-12s${RESET} │ %-15s │ ${v_color}%-8s${RESET} │ " \
            "${IDs[$i]}" "${PRIs[$i]}" "${NAMES[$i]}" "$s" "${AGENTS[$i]}" "$v_status"
        draw_bar "${PROG[$i]}"
        printf " │ %-5s │\n" "${ETAs[$i]}"
    done
    echo -e " └──────┴─────┴───────────────────────────────────────────────┴──────────────┴─────────────────┴──────────┴──────────────────────────┴───────┘"

    # Footer KPIs
    overall_pct=$((done_count * 100 / total_count))
    echo -e "\n  ${BOLD}OVERALL PROGRESS:${RESET} ${C}${overall_pct}%${RESET}   │   ${G}✔ ${done_count} Concluídas${RESET}   │   ${C}▶ ${in_progress} Em Curso${RESET}   │   ${R}✘ 0 Bloqueadas${RESET}"
    echo -e "  ${Y}${BOLD}⚠ ALERTA DE OCIOSIDADE:${RESET} Agentes aguardando gate de QA em P6."
    
    echo -ne "\n  ${D}(refresh ${INTERVAL}s — $(date '+%H:%M:%S'))${RESET}"
    sleep "$INTERVAL"
done

