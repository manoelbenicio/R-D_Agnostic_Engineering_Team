#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
FLEET EXECUTIVE DASHBOARD (Herdr-over-SSH, realtime) — roda AQUI, monitora o fleet remoto.

Layout executivo: ID | PRI | TAREFA | STATUS | AGENTE(pane) | VIVO | PROGRESSO(barra+%) | ETA
+ painel KPIs & ALERTS (progresso geral, contagens, alerta de ociosidade).

Fonte:
  - VIVO/pane  = Herdr socket remoto (`herdr agent list --json`) via SSH  (estado ao vivo do agente)
  - TAREFA/STATUS/PROGRESSO/ETA = check-ins do board (.deploy-control/<AGENTE>__<STREAM>__<UTC>.md)

Campos opcionais no front-matter do check-in (melhoram a acurácia; se ausentes, são derivados):
  status:  DONE|IN_PROGRESS|BLOCKED
  progress: 0..100     priority: P0|P1|P2     phase: F1...     task: <descrição curta>
  eta: <ex. 2h / 30m>  started_at / finished_at / notes / build_result

Uso:
  python3 scripts/dashboard/fleet_dashboard.py            # ao vivo (Ctrl+C sai)
  python3 scripts/dashboard/fleet_dashboard.py --once | --json | --ascii | --interval 3
  python3 scripts/dashboard/fleet_dashboard.py --ssh <host> --board <path>
Canal Tech-Lead -> SOMENTE opus-4.8-orchestrator:
  --msg "..."   |   --status   |   --read
"""
import argparse, json, os, re, shlex, subprocess, sys, time
from datetime import datetime, timezone

DEFAULT_HOST = os.environ.get("FLEET_SSH_HOST", "manoelneto-laptop")
DEFAULT_BOARD = os.environ.get("FLEET_BOARD", "/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.deploy-control")
ORCH = os.environ.get("FLEET_ORCHESTRATOR", "opus-4.8-orchestrator")
SEP = "@@@BOARD@@@"

def use_color(ascii_mode):
    if ascii_mode or os.environ.get("NO_COLOR"): return False
    return sys.stdout.isatty() or os.environ.get("RPP_FORCE_COLOR") == "1"

class C:
    def __init__(self, on): self.on = on
    def w(self, s, c): return f"\033[{c}m{s}\033[0m" if self.on else s
    def green(self, s): return self.w(s, "32")
    def yellow(self, s): return self.w(s, "33")
    def red(self, s): return self.w(s, "31")
    def grey(self, s): return self.w(s, "90")
    def cyan(self, s): return self.w(s, "36")
    def blue(self, s): return self.w(s, "34")
    def bold(self, s): return self.w(s, "1")

# ---------- ssh ----------
def ssh(host, remote_cmd, timeout=15):
    try:
        p = subprocess.run(["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10",
                            "-o", "StrictHostKeyChecking=accept-new", host, remote_cmd],
                           capture_output=True, text=True, timeout=timeout)
        return p.returncode, p.stdout, p.stderr
    except subprocess.TimeoutExpired:
        return 124, "", "ssh timeout"
    except Exception as e:
        return 1, "", str(e)

def fetch(host, board):
    remote = ("herdr agent list 2>/dev/null; "
              f"echo {shlex.quote(SEP)}; "
              f"cd {shlex.quote(board)} 2>/dev/null && "
              "for f in *__*__*.md; do [ -e \"$f\" ] || continue; echo \"@@@F:$f\"; sed -n '1,60p' \"$f\"; done")
    return ssh(host, remote)

# ---------- parse ----------
def norm(s): return re.sub(r"[^a-z0-9]", "", (s or "").lower())

def parse_ts(s):
    if not s: return None
    s = str(s).strip().strip('"').strip("'")
    for f in ("%Y-%m-%dT%H:%M:%SZ", "%Y%m%dT%H%M%SZ", "%Y-%m-%dT%H-%M-%SZ",
              "%Y-%m-%dT%H:%M:%S%z", "%Y-%m-%dT%H:%M:%S", "%Y-%m-%d %H:%M:%S"):
        try:
            dt = datetime.strptime(s, f)
            return dt.replace(tzinfo=timezone.utc) if dt.tzinfo is None else dt.astimezone(timezone.utc)
        except ValueError: continue
    m = re.search(r"(\d{8}T\d{6}Z)", s)
    if m:
        try: return datetime.strptime(m.group(1), "%Y%m%dT%H%M%SZ").replace(tzinfo=timezone.utc)
        except ValueError: pass
    return None

def parse_agents(text):
    text = (text or "").strip()
    if not text: return {}
    try: data = json.loads(text)
    except json.JSONDecodeError:
        m = re.search(r"\{.*\}", text, re.S)
        if not m: return {}
        try: data = json.loads(m.group(0))
        except json.JSONDecodeError: return {}
    ags = (data.get("result") or {}).get("agents", []) if isinstance(data, dict) else []
    return {norm(a.get("name")): a for a in ags}

def parse_board(text):
    """lista de check-ins (todos); dedup por (agente,stream) mantendo o mais recente."""
    files, fname, buf = {}, None, []
    for ln in (text or "").splitlines():
        if ln.startswith("@@@F:"):
            if fname: files[fname] = buf
            fname, buf = ln[5:].strip(), []
        elif fname is not None:
            buf.append(ln)
    if fname: files[fname] = buf
    tasks = {}
    for fn, lines in files.items():
        d = {}
        for l in lines:
            mm = re.match(r"^\s*-?\s*([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*)$", l)
            if mm:
                k, v = mm.group(1).lower(), mm.group(2).strip().strip('"')
                if k not in d and v: d[k] = v
        m = re.match(r"^(.*?)__(.*?)__(.*)\.md$", fn)
        agent = d.get("agent") or (m.group(1) if m else fn)
        stream = d.get("stream") or (m.group(2) if m else "")
        ts = d.get("started_at") or (m.group(3) if m else "")
        d["agent"], d["stream"], d["_ts"] = agent, stream, ts
        key = (norm(agent), norm(stream))
        t = parse_ts(d.get("started_at")) or parse_ts(ts)
        d["_score"] = t.timestamp() if t else 0
        if key not in tasks or d["_score"] >= tasks[key]["_score"]:
            tasks[key] = d
    return list(tasks.values())

# ---------- derivações ----------
def derive_priority(d):
    p = (d.get("priority") or "").upper()
    if p in ("P0", "P1", "P2"): return p
    ph = (d.get("phase") or d.get("stream") or "").upper()
    if "F0" in ph or "DEPLOY" in ph or "SMOKE" in ph: return "P0"
    if any(x in ph for x in ("F1", "F2", "F3", "F4", "CONTRACT", "STATE", "INTEGRATE", "FORKMAP", "CONFORMANCE")): return "P1"
    return "P2"

def derive_status(d):
    s = (d.get("status") or "").upper()
    if any(x in s for x in ("CANCEL", "ABORT", "SUPERSED")): return "CANCELLED"
    if "FAIL" in s: return "FAILED"
    if "BLOCK" in s: return "BLOCKED"
    if "DONE" in s or "COMPLETE" in s: return "DONE"
    if "PROGRESS" in s or "WORKING" in s: return "IN_PROGRESS"
    return "TODO"

def derive_progress(d, status, now):
    if d.get("progress"):
        try: return max(0, min(100, int(float(re.sub(r"[^0-9.]", "", d["progress"])))))
        except ValueError: pass
    if status == "DONE": return 100
    if status == "BLOCKED": return 50
    if status == "IN_PROGRESS":
        st = parse_ts(d.get("started_at") or d.get("_ts")); eta = parse_eta_hours(d.get("eta"))
        if st and eta:
            frac = ((now - st).total_seconds() / 3600) / eta
            return max(5, min(90, int(frac * 100)))
        return 50
    return 0

def parse_eta_hours(s):
    if not s: return None
    s = str(s).lower(); h = 0.0
    m = re.search(r"(\d+(?:\.\d+)?)\s*h", s);  h += float(m.group(1)) if m else 0
    m = re.search(r"(\d+)\s*m", s);            h += int(m.group(1))/60 if m else 0
    if h == 0:
        m = re.fullmatch(r"\s*(\d+(?:\.\d+)?)\s*", s)
        if m: h = float(m.group(1))
    return h or None

def eta_display(d, status, prog, now):
    if status == "DONE": return "0m"
    eta_h = parse_eta_hours(d.get("eta"))
    st = parse_ts(d.get("started_at") or d.get("_ts"))
    if eta_h and st:
        rem = eta_h - (now - st).total_seconds()/3600
        return fmt_h(rem) if rem > 0 else "atraso"
    if eta_h: return fmt_h(eta_h)
    return "—"

def fmt_h(h):
    h = abs(h); hh = int(h); mm = int(round((h-hh)*60))
    if mm == 60: hh += 1; mm = 0
    if hh and mm: return f"{hh}h{mm:02d}m"
    if hh: return f"{hh}h"
    return f"{mm}m"

VIVO_COLOR = {"working": "green", "done": "green", "idle": "yellow", "blocked": "red", "unknown": "grey"}
STATUS_DISP = {"DONE": ("green", "● DONE"), "IN_PROGRESS": ("yellow", "▶ WORKING"),
               "BLOCKED": ("red", "■ BLOCKED"), "FAILED": ("red", "✖ FAILED"),
               "CANCELLED": ("grey", "⊘ CANCEL"), "TODO": ("grey", "○ TODO")}

# ---------- C-LEVEL executive report (markdown) ----------
def report_md(host, rows, now):
    c = {"DONE":0,"IN_PROGRESS":0,"BLOCKED":0,"FAILED":0,"CANCELLED":0,"TODO":0}
    progs = []
    for r in rows:
        c[r["status"]] = c.get(r["status"],0)+1
        if r["status"] != "CANCELLED": progs.append(r["prog"])
    total = len(rows)
    overall = int(round(sum(progs)/len(progs))) if progs else 0
    faltam = total - c["DONE"] - c["CANCELLED"]
    SL = {"DONE":"✅ DONE","IN_PROGRESS":"🔄 EM CURSO","BLOCKED":"⛔ BLOQUEADA",
          "FAILED":"❌ FALHADA","CANCELLED":"⊘ CANCELADA","TODO":"⬜ EM ESPERA"}
    L = []
    L.append("# STATUS EXECUTIVO (C-LEVEL) — Rotation-Parity Polyglot")
    L.append("")
    L.append(f"> **Gerado:** {now.strftime('%Y-%m-%d %H:%M:%SZ')} · **Fonte:** Herdr socket + board ({host}) · **Mantido por:** Tech-Lead (Opus 4.8)")
    L.append("")
    L.append(f"## Panorama: **OVERALL {overall}%** — FALTAM **{faltam}/{total}**")
    L.append("")
    L.append(f"✅ {c['DONE']} concluídas · 🔄 {c['IN_PROGRESS']} em curso · ⬜ {c['TODO']} em espera · ⛔ {c['BLOCKED']} bloqueadas · ❌ {c['FAILED']} falhadas · ⊘ {c['CANCELLED']} canceladas")
    L.append("")
    L.append("| # | Item | Tarefa | Status | Dono | Prog | ETA | Observação |")
    L.append("|---|------|--------|--------|------|-----:|-----|------------|")
    for r in rows:
        obs = (r.get("motivo","") or "").replace("|","/")[:70] if r["status"] != "DONE" else "—"
        nm = r["tarefa"].replace(" [GATED]","").replace("|","/")[:44]
        L.append(f"| {r.get('num','')} | {r['id']} | {nm} | {SL.get(r['status'],r['status'])} | {r['agent']} | {r['prog']}% | {r['eta']} | {obs} |")
    L.append("")
    issues = [r for r in rows if r["status"] != "DONE"]
    if issues:
        L.append("## Pendências (o que falta até GA)")
        for r in issues:
            L.append(f"- **{r['id']} — {r['tarefa'].replace(' [GATED]','')}** ({SL.get(r['status'])}, dono {r['agent']}, ETA {r['eta']}): {r.get('motivo','')}")
        L.append("")
    L.append("## Decisões pendentes do dono (owner-only)")
    L.append("- **F5 vendor sign-off:** aceitar capabilities `not_validated` como disabled-by-default (ACCEPT recomendado p/ ausências factuais; #7 Smart Context = gate; #6 OpenCode arquivado = descopar).")
    L.append("- **F7 deploy PROD:** **NO-GO** até G5/G10/F4/F6 verdes + runbook reconciliado.")
    L.append("")
    L.append("## Próximo marco (gate de GA)")
    L.append("Deploy prodex AS-IS em PROD (F0) **somente após**: G5 Smart Context (shadow→canary), G10 container/kill-switch/rollback, F4 redaction/no-SQLite e F6 conformance — todos **verdes com evidência** — e aprovação do dono.")
    return "\n".join(L)

# ---------- render ----------
def clip(s, n):
    s = str(s or ""); return s if len(s) <= n else s[:n-1] + "…"

def repo_root():
    return os.environ.get("RPP_REPO_ROOT", os.path.abspath(os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..")))

def load_plan(root):
    path = os.path.join(root, ".deploy-control", "dashboard", "tasks.json")
    try:
        with open(path, encoding="utf-8") as f:
            return json.load(f).get("tasks", [])
    except Exception:
        return []

def checkin_phase(d):
    p = norm(d.get("phase"))
    m = re.fullmatch(r"f(\d)\+?", p)
    if m: return "f" + m.group(1)
    m = re.search(r"\bf(\d)\b", (str(d.get("stream", "")) + " " + str(d.get("task", ""))).lower())
    return "f" + m.group(1) if m else None

def match_checkins(task, checkins):
    ph = norm(task.get("phase")); aliases = [norm(a) for a in task.get("aliases", [])]
    if task.get("gated"):  # deploy/gated: só casa por alias explícito (menção de "F0" em outro stream NÃO conta)
        return [d for d in checkins if norm(d.get("stream")) in aliases]
    return [d for d in checkins if norm(d.get("stream")) in aliases or checkin_phase(d) == ph]

def agg_status(matched):
    if not matched: return "TODO"
    sts = [derive_status(d) for d in matched]
    if "BLOCKED" in sts: return "BLOCKED"
    if "FAILED" in sts: return "FAILED"
    if "IN_PROGRESS" in sts: return "IN_PROGRESS"
    if all(s == "CANCELLED" for s in sts): return "CANCELLED"
    if all(s == "DONE" for s in sts): return "DONE"
    return "DONE" if "DONE" in sts else "IN_PROGRESS"

def build_tasks(plan, checkins, agents, now):
    rows = []
    for task in plan:
        matched = match_checkins(task, checkins)
        status = "TODO" if (task.get("gated") and not matched) else agg_status(matched)
        if status == "DONE": prog = 100
        elif status == "TODO": prog = 0
        else:
            ps = [derive_progress(d, derive_status(d), now) for d in matched] or [0]
            prog = int(sum(ps) / len(ps))
        latest = max(matched, key=lambda d: d.get("_score", 0)) if matched else None
        agent_name = (latest.get("agent") if latest else task.get("owner")) or "—"
        ag = agents.get(norm(agent_name)) or (agents.get(norm(task.get("owner", ""))) if task.get("owner") else None)
        vivo = (ag.get("agent_status") if ag else ("unknown" if matched else "—")) or "—"
        pane = ag.get("pane_id") if ag else ""
        eta = (eta_display(latest, status, prog, now) if latest
               else ("0m" if status == "DONE" else (fmt_h(task.get("eta_hours")) if task.get("eta_hours") else "—")))
        if status == "TODO":
            motivo = "GATED — aguarda aprovação do dono + smokes verdes" if task.get("gated") else "não iniciado"
        elif status == "IN_PROGRESS":
            motivo = (latest.get("notes") if latest else "") or f"em curso ({prog}%)"
        elif status == "BLOCKED":
            motivo = ((latest.get("blockers") or latest.get("notes")) if latest else "") or "bloqueado"
        elif status == "FAILED":
            motivo = ((latest.get("build_result") or latest.get("notes")) if latest else "") or "build falhou"
        elif status == "CANCELLED":
            motivo = (latest.get("notes") if latest else "") or "cancelado/superseded"
        else:
            motivo = ""
        rows.append({"num": task.get("num", 0), "kind": task.get("kind", "FASE"),
                     "id": task.get("phase", "?"), "pri": task.get("priority") or "P2",
                     "tarefa": (task.get("name", "—") + (" [GATED]" if task.get("gated") else "")),
                     "status": status, "agent": agent_name, "pane": pane, "vivo": vivo,
                     "prog": prog, "eta": eta, "gated": task.get("gated"),
                     "motivo": motivo, "notes": (latest.get("notes") if latest else "") or ""})
    return rows

def bar_cell(col, pct, width):
    pct = max(0, min(100, int(round(pct))))
    bw = 12; fill = int(round(bw*pct/100))
    tier = col.green if pct >= 100 else (col.yellow if pct >= 40 else col.red)
    txt = tier("█"*fill) + col.grey("░"*(bw-fill)) + " " + tier(f"{pct:3d}%")
    visible = bw + 1 + 4
    return txt + " "*max(0, width - visible)

def render(host, rows, now, col, ascii_mode, err=None):
    W = {"num": 3, "id": 4, "pri": 4, "tarefa": 34, "status": 11, "agente": 20, "vivo": 8, "prog": 18, "eta": 7}
    inner = sum(W.values()) + (len(W)-1)*3
    def cell(s, w): return clip(s, w).ljust(w)
    def row(cells): return "│ " + " │ ".join(cells) + " │"
    top = "┌" + "─"*(inner+2) + "┐"
    bot = "└" + "─"*(inner+2) + "┘"
    rule = "├" + "─"*(inner+2) + "┤"
    L = []
    L.append(col.cyan(top))
    L.append(col.cyan("│") + col.bold(col.cyan((" FLEET — EXECUTIVE DASHBOARD (Herdr-over-SSH · realtime) ").center(inner+2))) + col.cyan("│"))
    L.append(col.cyan("│") + col.grey(f" host: {host}   {now.strftime('%Y-%m-%d %H:%M:%SZ')}   fonte: herdr socket + board ".ljust(inner+2)) + col.cyan("│"))
    if err:
        L.append(col.cyan("│") + col.red((" ERRO: " + err.strip()[:inner-8]).ljust(inner+2)) + col.cyan("│"))
    L.append(col.cyan(rule))
    L.append(col.cyan(row([col.bold(cell(h, W[k])) for k, h in
                            [("num","#"),("id","FASE"),("pri","PRI"),("tarefa","TAREFA"),("status","STATUS"),
                             ("agente","AGENTE"),("vivo","VIVO"),("prog","PROGRESSO"),("eta","ETA")]])))
    L.append(col.cyan(rule))
    counts = {"DONE":0,"IN_PROGRESS":0,"BLOCKED":0,"FAILED":0,"CANCELLED":0,"TODO":0}
    idle_pend = []; progs = []; prev_kind = None
    for r in rows:
        if r.get("kind") != prev_kind:
            band = " FASES DE EXECUÇÃO (F0–F9) " if r.get("kind") == "FASE" else " GATES DE ACEITE — Definition-of-Done (G1–G10) "
            L.append(col.cyan("│") + col.bold(col.blue(band.ljust(inner+2))) + col.cyan("│"))
            prev_kind = r.get("kind")
        counts[r["status"]] = counts.get(r["status"],0)+1
        if r["status"] != "CANCELLED": progs.append(r["prog"])
        scolor, slabel = STATUS_DISP.get(r["status"], ("grey", r["status"]))
        vcolor = VIVO_COLOR.get(r["vivo"], "grey")
        pricol = col.red if r["pri"]=="P0" else (col.yellow if r["pri"]=="P1" else col.grey)
        agente = clip(f'{r["agent"]}', W["agente"]-len(r["pane"])-3) + (col.grey(f' ({r["pane"]})') if r["pane"] else "")
        vis = len(clip(f'{r["agent"]}', W["agente"]-len(r["pane"])-3)) + (len(r["pane"])+3 if r["pane"] else 0)
        agente = agente + " "*max(0, W["agente"]-vis)
        cells = [
            col.grey(cell(str(r.get("num","")), W["num"])),
            cell(r["id"], W["id"]),
            pricol(cell(r["pri"], W["pri"])),
            cell(r["tarefa"], W["tarefa"]),
            getattr(col, scolor)(cell(slabel, W["status"])),
            agente,
            getattr(col, vcolor)(cell(r["vivo"].upper(), W["vivo"])),
            bar_cell(col, r["prog"], W["prog"]),
            (col.red if r["eta"]=="atraso" else col.grey)(cell(r["eta"], W["eta"])),
        ]
        L.append(col.cyan(row(cells)))
        if r["vivo"] == "idle" and r["status"] not in ("DONE","CANCELLED"):
            idle_pend.append(f'{r["agent"]}({r["pane"] or "?"})')
    L.append(col.cyan(bot))
    # KPIs & ALERTS
    total_t = len(rows)
    overall = int(round(sum(progs)/len(progs))) if progs else 0
    faltam = total_t - counts['DONE'] - counts['CANCELLED']
    L.append("")
    L.append(col.blue("── KPIs & ALERTS " + "─"*max(0, inner-13)) if col.on else "── KPIs & ALERTS ──")
    obar = bar_cell(col, overall, 18)
    L.append(f" OVERALL {obar}   "
             f"{col.green('✔ '+str(counts['DONE'])+' Concluídas')}   "
             f"{col.yellow('▶ '+str(counts['IN_PROGRESS'])+' Em Curso')}   "
             f"{col.grey('▢ '+str(counts['TODO'])+' Em Espera')}   "
             f"{col.red('■ '+str(counts['BLOCKED'])+' Bloqueadas')}   "
             f"{col.red('✖ '+str(counts['FAILED'])+' Falhadas')}   "
             f"{col.grey('⊘ '+str(counts['CANCELLED'])+' Canceladas')}")
    fcol = col.green if faltam == 0 else col.yellow
    L.append(" " + col.bold(fcol(f"FALTAM {faltam}/{total_t}")) +
             f"  ({counts['TODO']} em espera · {counts['IN_PROGRESS']} em curso · {counts['BLOCKED']} bloqueadas · {counts['FAILED']} falhadas)"
             + ("  " + col.grey("[inclui F0 deploy GATED]") if any(r.get('gated') and r['status'] != 'DONE' for r in rows) else ""))
    if idle_pend:
        L.append(" " + col.yellow("⚠ OCIOSIDADE: agentes IDLE com tarefa pendente → " + ", ".join(idle_pend)))
    # 360: detalhe de pendências e ocorrências (o que falta / falhou / cancelou / travou)
    issues = [r for r in rows if r["status"] != "DONE"]
    if issues:
        L.append(col.bold(" PENDÊNCIAS & OCORRÊNCIAS (360°):"))
        for r in issues:
            sc, sl = STATUS_DISP.get(r["status"], ("grey", r["status"]))
            who = r["agent"] + (f' {r["pane"]}' if r["pane"] else "")
            L.append("   " + col.grey(f'#{r.get("num",""):<2} ') + getattr(col, sc)(f'{r["id"]:<3} {sl:<10}') + " " +
                     clip(r["tarefa"].replace(" [GATED]",""), 38).ljust(38) + " " +
                     col.grey(clip(who, 20).ljust(20)) + " → " + clip(r.get("motivo",""), 52))
    else:
        L.append(" " + col.green("✔ Nada pendente/falhado/cancelado — tudo concluído."))
    L.append(col.grey(f" Tech-Lead → SOMENTE {ORCH}:  --msg \"...\"  |  --status  |  --read"))
    return "\n".join(L)

# ---------- orchestrator (SOMENTE opus-4.8-orchestrator) ----------
def orch_pane(host):
    rc, out, err = ssh(host, "herdr agent list")
    try:
        for a in json.loads(out).get("result", {}).get("agents", []):
            if a.get("name") == ORCH:
                return a.get("pane_id")
    except Exception:
        pass
    return None

def msg_orch(host, text):
    # pane run = texto + Enter (submete de verdade). agent send NAO da Enter -> mensagem nao e processada.
    pane = orch_pane(host)
    if pane:
        return ssh(host, "herdr pane run " + shlex.quote(pane) + " " + shlex.quote(text))
    return ssh(host, f"herdr agent send {shlex.quote(ORCH)} {shlex.quote(text)}")

def read_orch(host, n=60): return ssh(host, f"herdr agent read {shlex.quote(ORCH)} --source recent --lines {int(n)}")

# ---------- main ----------
def snapshot(host, board):
    rc, out, err = fetch(host, board)
    if rc != 0 and not out: return {}, [], (err or f"ssh rc={rc}")
    parts = out.split(SEP, 1)
    agents = parse_agents(parts[0])
    checkins = parse_board(parts[1]) if len(parts) > 1 else []
    return agents, checkins, (None if (agents or checkins) else (err or "sem dados"))

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--ssh", default=DEFAULT_HOST); ap.add_argument("--board", default=DEFAULT_BOARD)
    ap.add_argument("--once", action="store_true"); ap.add_argument("--json", action="store_true")
    ap.add_argument("--report", action="store_true", help="emite STATUS EXECUTIVO (markdown, C-Level)")
    ap.add_argument("--ascii", action="store_true"); ap.add_argument("--interval", type=float, default=5.0)
    ap.add_argument("--msg", metavar="TEXT"); ap.add_argument("--status", action="store_true"); ap.add_argument("--read", action="store_true")
    args = ap.parse_args()
    col = C(use_color(args.ascii)); host = args.ssh

    if args.msg is not None:
        rc, o, e = msg_orch(host, args.msg); print("enviado ao orquestrador." if rc == 0 else f"falhou: {e or o}"); return
    if args.read:
        rc, o, e = read_orch(host); print(o if rc == 0 else e); return
    if args.status:
        rc, o, e = msg_orch(host, "[Tech-Lead] status geral do fleet? resumo por agente + bloqueios."); print("pedido enviado." if rc==0 else f"falhou: {e or o}")
        time.sleep(3); rc, o, e = read_orch(host); print("\n--- pane do orquestrador ---\n" + (o if rc==0 else e)); return

    plan = load_plan(repo_root())

    def frame():
        now = datetime.now(timezone.utc)
        agents, checkins, err = snapshot(host, args.board)
        rows = build_tasks(plan, checkins, agents, now)
        if args.report:
            return report_md(host, rows, now)
        if args.json:
            faltam = sum(1 for r in rows if r["status"] != "DONE")
            return json.dumps({"host": host, "at": now.strftime("%Y-%m-%dT%H:%M:%SZ"),
                               "total": len(rows), "faltam": faltam, "tasks": rows}, ensure_ascii=False, indent=2)
        return render(host, rows, now, col, args.ascii, err)

    if args.once or args.json or args.report: print(frame()); return
    tty = sys.stdout.isatty(); ALT_ON, ALT_OFF = "\033[?1049h\033[?25l", "\033[?25h\033[?1049l"
    try:
        if tty: sys.stdout.write(ALT_ON)
        while True:
            f = frame() + "\n " + col.grey(f"(refresh {args.interval:g}s — Ctrl+C sai)")
            if tty:
                sys.stdout.write("\033[H")
                for ln in f.split("\n"): sys.stdout.write(ln + "\033[K\n")
                sys.stdout.write("\033[J")
            else:
                sys.stdout.write("\033[2J\033[H" + f + "\n")
            sys.stdout.flush(); time.sleep(args.interval)
    except KeyboardInterrupt: pass
    finally:
        if tty: sys.stdout.write(ALT_OFF); sys.stdout.flush()
        print("bye.")

if __name__ == "__main__":
    main()
