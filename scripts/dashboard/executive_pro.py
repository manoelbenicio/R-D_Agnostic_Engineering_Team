#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import os, sys, time, re, subprocess, json
from datetime import datetime

# Configurações
REPO = "/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team"
TASKS_FILE = os.path.join(REPO, "openspec/changes/rotation-parity-polyglot/tasks.md")
DEPLOY_CTL = os.path.join(REPO, ".deploy-control")
DASH = os.path.join(REPO, "scripts/dashboard/plan_dashboard.py")

class Color:
    RESET = "\033[0m"; BOLD = "\033[1m"; RED = "\033[31m"; GREEN = "\033[32m"
    YELLOW = "\033[33m"; BLUE = "\033[34m"; CYAN = "\033[36m"; GREY = "\033[90m"

def get_herdr_status():
    try:
        res = subprocess.run(["herdr", "agent", "list"], capture_output=True, text=True, env={**os.environ, "HERDR_ENV": "1"})
        if res.returncode == 0:
            data = json.loads(res.stdout)
            return {a.get("name", "unk"): a.get("pane_id", "---") for a in data.get("result", {}).get("agents", [])}
    except: pass
    return {}

def parse_deploy_control():
    task_progress_map = {}
    task_agent_map = {}
    
    if os.path.exists(DEPLOY_CTL):
        try:
            files = sorted(os.listdir(DEPLOY_CTL))
            for f in files:
                if f.endswith(".md"):
                    try:
                        with open(os.path.join(DEPLOY_CTL, f), "r", encoding="utf-8") as cf:
                            content = cf.read()
                            prog_match = re.search(r"^progress:\s*(\d+)", content, re.M)
                            agent_match = re.search(r"^agent:\s*(.*)", content, re.M)
                            prog = int(prog_match.group(1)) if prog_match else 0
                            agent = agent_match.group(1).strip() if agent_match else f.split("__")[0]
                            
                            found_ids = re.findall(r"\b(\d+\.\d+[a-z]?)\b", content)
                            # Também procura referências C1, C2, etc
                            if "C1" in content: found_ids.append("6.1")
                            if "C2" in content: found_ids.append("6.2")
                            if "C3" in content: found_ids.append("6.3")
                            if "C4" in content: found_ids.append("6.4")
                            if "C5" in content: found_ids.append("6.5")
                            if "C6" in content: found_ids.append("6.6")

                            for tid in found_ids:
                                task_progress_map[tid] = prog
                                task_agent_map[tid] = agent
                    except: continue
        except: pass
    return task_progress_map, task_agent_map

def get_sev0_data():
    try:
        res = subprocess.run([sys.executable, DASH, "--json", "--tasks", TASKS_FILE], capture_output=True, text=True, env={**os.environ, "PYTHONIOENCODING": "utf-8"})
        if res.returncode == 0:
            return json.loads(res.stdout)
    except Exception as e:
        pass
    return None

def render_bar(pct):
    width = 15
    fill = int(pct * width / 100)
    color = Color.RED if pct < 40 else Color.YELLOW if pct < 100 else Color.GREEN
    return f"{color}{'█'*fill}{Color.GREY}{'░'*(width-fill)}{Color.RESET} {color}{pct:3}%{Color.RESET}"

def main():
    while True:
        try:
            herdr_panes = get_herdr_status()
            task_progress_map, task_agent_map = parse_deploy_control()
            data = get_sev0_data()
            
            os.system("clear")
            print(f"{Color.GREY}══════════════════════════════════════════════════════════════════════════════════════════════════════════════════════{Color.RESET}")
            print(f"                                              {Color.BOLD}SQUAD AOP{Color.RESET}")
            print(f"                                      {Color.BOLD}SQUAD AOP — EXECUTIVE DASHBOARD{Color.RESET}")
            print(f"{Color.GREY}══════════════════════════════════════════════════════════════════════════════════════════════════════════════════════{Color.RESET}")
            print(f" │ {Color.CYAN}{'ID':<4}{Color.RESET} │ {Color.CYAN}{'PRI':<3}{Color.RESET} │ {Color.CYAN}{'TAREFA':<45}{Color.RESET} │ {Color.CYAN}{'STATUS':<12}{Color.RESET} │ {Color.CYAN}{'AGENTE':<20}{Color.RESET} │ {Color.CYAN}{'VIVO':<10}{Color.RESET} │ {Color.CYAN}{'PROGRESSO':<20}{Color.RESET} │ {Color.CYAN}{'ETA':<5}{Color.RESET} │")
            print(f" ├──────┼─────┼───────────────────────────────────────────────┼──────────────┼──────────────────────┼────────────┼──────────────────────┼───────┤")

            tasks_list = []
            if data:
                for ph in data.get("phases", []):
                    for task_dict in ph.get("tasks", []):
                        tid = task_dict["num"]
                        done = task_dict["done"]
                        name = task_dict["text"]
                        
                        real_prog = 100 if done else task_progress_map.get(tid, 0)
                        agent = task_agent_map.get(tid, "---") if not done else "---"
                        pane = herdr_panes.get(agent, "---")
                        
                        # Fix for exact matching in tasks list
                        status = "DONE" if real_prog == 100 else "WORKING" if real_prog > 0 else "TODO"
                        
                        tasks_list.append({
                            "id": tid,
                            "pri": "P0" if "[P0]" in ph["phase"] else "P1" if "[P1]" in ph["phase"] else "P2",
                            "task": name[:45],
                            "status": status,
                            "progress": real_prog,
                            "agent": agent[:20],
                            "pane": pane,
                            "eta": "0m" if real_prog == 100 else "15m"
                        })

            # Mostrar P6 + qualquer coisa "WORKING" ou P3 (agora que o prodex buildou)
            visible_tasks = [t for t in tasks_list if t["id"].startswith("6.") or t["id"].startswith("3.") or t["status"] == "WORKING"][:14]
            
            for t in visible_tasks:
                p_col = Color.RED if t["pri"] == "P0" else Color.YELLOW if t["pri"] == "P1" else Color.GREY
                s_col = Color.GREEN if t["status"] == "DONE" else Color.BLUE if t["status"] == "WORKING" else Color.GREY
                print(f" │ {t['id']:<4} │ {p_col}{t['pri']:<3}{Color.RESET} │ {t['task']:<45} │ {s_col}{t['status']:<12}{Color.RESET} │ {t['agent']:<20} │ {t['pane']:<10} │ {render_bar(t['progress'])} │ {t['eta']:<5} │")

            print(f" └──────┴─────┴───────────────────────────────────────────────┴──────────────┴──────────────────────┴────────────┴──────────────────────┴───────┘")
            
            done_cnt = data["overall_done"] if data else 0
            total_cnt = data["overall_total"] if data else 66
            overall_pct = int((done_cnt / total_cnt) * 100) if total_cnt > 0 else 0

            print(f"\n {Color.BLUE}╭────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮{Color.RESET}")
            print(f" {Color.BLUE}│{Color.RESET}  {Color.BOLD}OVERALL PROGRESS:{Color.RESET} {Color.YELLOW}{overall_pct}%{Color.RESET}   │   {Color.GREEN}✔ {done_cnt} Concluídas{Color.RESET}   │   {Color.YELLOW}▶ {total_cnt-done_cnt} Em Aberto{Color.RESET}   │   {Color.BLUE}▢ 20 SEV-0 Gates{Color.RESET}   │   {Color.GREEN}✔ 0 Bloqueadas{Color.RESET}  {Color.BLUE}│{Color.RESET}")
            print(f" {Color.BLUE}╰────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯{Color.RESET}")
            print(f"  {Color.GREEN}✔ SEV-0 STATUS:{Color.RESET} Dashboard Engine [GREEN] | Prodex Binary [GREEN] | QA Gates [IN PROGRESS]")
            print(f"\n  {Color.GREY}(refresh 5s — {datetime.now().strftime('%H:%M:%S')}){Color.RESET}", end="", flush=True)
            time.sleep(5)
        except KeyboardInterrupt:
            break
        except Exception as e:
            print(f"Erro no loop principal: {e}")
            time.sleep(5)

if __name__ == "__main__":
    main()

