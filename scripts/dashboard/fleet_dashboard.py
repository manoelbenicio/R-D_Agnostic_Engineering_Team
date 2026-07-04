#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Fleet Dashboard (Herdr-over-SSH, realtime) — roda AQUI, monitora o fleet no host remoto.

Fonte da verdade = Herdr socket no host do fleet (`herdr agent list --json`), lido via SSH.
Enriquecido com os check-ins do board remoto (.deploy-control/<AGENT>__<STREAM>__<UTC>.md)
para stream/ETA/OBS. Sem git, sem sync — estado real dos agentes despachados, ao vivo.

Uso:
  python3 scripts/dashboard/fleet_dashboard.py                 # live (refresh 5s)
  python3 scripts/dashboard/fleet_dashboard.py --once
  python3 scripts/dashboard/fleet_dashboard.py --json
  python3 scripts/dashboard/fleet_dashboard.py --interval 3
  python3 scripts/dashboard/fleet_dashboard.py --ssh manoelneto-laptop
Canal Tech-Lead -> APENAS o orquestrador (opus-4.8-orchestrator):
  python3 scripts/dashboard/fleet_dashboard.py --msg "status geral do fleet?"
  python3 scripts/dashboard/fleet_dashboard.py --status      # pede status + le a resposta do pane
  python3 scripts/dashboard/fleet_dashboard.py --read        # le o pane do orquestrador

Sem dependencias externas (stdlib). Requer SSH configurado p/ o host (BatchMode).
"""
import argparse, json, os, re, shlex, subprocess, sys, time
from datetime import datetime, timezone

DEFAULT_HOST = os.environ.get("FLEET_SSH_HOST", "manoelneto-laptop")
DEFAULT_BOARD = os.environ.get("FLEET_BOARD", "/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.deploy-control")
ORCH = os.environ.get("FLEET_ORCHESTRATOR", "opus-4.8-orchestrator")
SEP = "@@@BOARD@@@"

# ---------- cor ----------
def use_color(ascii_mode):
    if ascii_mode or os.environ.get("NO_COLOR"): return False
    return sys.stdout.isatty() or os.environ.get("RPP_FORCE_COLOR") == "1"

class C:
    def __init__(self, on): self.on = on
    def _w(self, s, c): return f"\033[{c}m{s}\033[0m" if self.on else s
    def green(self, s): return self._w(s, "32")
    def yellow(self, s): return self._w(s, "33")
    def red(self, s): return self._w(s, "31")
    def grey(self, s): return self._w(s, "90")
    def cyan(self, s): return self._w(s, "36")
    def bold(self, s): return self._w(s, "1")

# ---------- ssh ----------
def ssh(host, remote_cmd, timeout=15):
    try:
        p = subprocess.run(
            ["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10",
             "-o", "StrictHostKeyChecking=accept-new", host, remote_cmd],
            capture_output=True, text=True, timeout=timeout)
        return p.returncode, p.stdout, p.stderr
    except subprocess.TimeoutExpired:
        return 124, "", "ssh timeout"
    except Exception as e:
        return 1, "", str(e)

def fetch(host, board):
    remote = (
        "herdr agent list 2>/dev/null; "
        f"echo {shlex.quote(SEP)}; "
        f"cd {shlex.quote(board)} 2>/dev/null && "
        "for f in *__*__*.md; do [ -e \"$f\" ] || continue; "
        "echo \"@@@F:$f\"; sed -n '1,40p' \"$f\"; done"
    )
    rc, out, err = ssh(host, remote)
    return rc, out, err

# ---------- parse ----------
def norm(s):
    return re.sub(r"[^a-z0-9]", "", (s or "").lower())

def parse_ts(s):
    if not s: return None
    s = str(s).strip().strip('"').strip("'")
    for f in ("%Y-%m-%dT%H:%M:%SZ", "%Y%m%dT%H%M%SZ", "%Y-%m-%dT%H-%M-%SZ",
              "%Y-%m-%dT%H:%M:%S%z", "%Y-%m-%dT%H:%M:%S", "%Y-%m-%d %H:%M:%S"):
        try:
            dt = datetime.strptime(s, f)
            return dt.replace(tzinfo=timezone.utc) if dt.tzinfo is None else dt.astimezone(timezone.utc)
        except ValueError:
            continue
    m = re.search(r"(\d{8}T\d{6}Z)", s)
    if m:
        try: return datetime.strptime(m.group(1), "%Y%m%dT%H%M%SZ").replace(tzinfo=timezone.utc)
        except ValueError: pass
    return None

def parse_agents(text):
    text = text.strip()
    if not text: return []
    try:
        data = json.loads(text)
    except json.JSONDecodeError:
        m = re.search(r"\{.*\}", text, re.S)
        if not m: return []
        try: data = json.loads(m.group(0))
        except json.JSONDecodeError: return []
    return (data.get("result") or {}).get("agents", []) if isinstance(data, dict) else []

def parse_board(text):
    """retorna {norm_agent: latest_checkin_dict}"""
    files = {}
    cur, buf, fname = None, [], None
    for ln in text.splitlines():
        if ln.startswith("@@@F:"):
            if fname: files[fname] = buf
            fname, buf = ln[5:].strip(), []
        elif fname is not None:
            buf.append(ln)
    if fname: files[fname] = buf
    latest = {}
    for fn, lines in files.items():
        d = {}
        for l in lines[:40]:
            mm = re.match(r"^\s*-?\s*([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*)$", l)
            if mm:
                k, v = mm.group(1).lower(), mm.group(2).strip()
                if k not in d and v: d[k] = v
        m = re.match(r"^(.*?)__(.*?)__(.*)\.md$", fn)
        agent_f = m.group(1) if m else d.get("agent", "")
        stream_f = m.group(2) if m else d.get("stream", "")
        ts_f = m.group(3) if m else ""
        d.setdefault("stream", stream_f)
        d["_agent_file"] = agent_f
        d["_ts"] = d.get("started_at") or ts_f
        key = norm(d.get("agent") or agent_f)
        t = parse_ts(d.get("started_at")) or parse_ts(ts_f)
        score = t.timestamp() if t else 0
        if key not in latest or score >= latest[key]["_score"]:
            d["_score"] = score
            latest[key] = d
    return latest

# ---------- farol ----------
def fmt_dur(delta_s):
    delta_s = abs(int(delta_s)); h = delta_s // 3600; m = (delta_s % 3600) // 60
    if h and m: return f"{h}h{m:02d}m"
    if h: return f"{h}h"
    return f"{m}m"

STATUS_FAROL = {"working": ("green", "● WORKING"), "done": ("green", "● DONE"),
                "idle": ("yellow", "● IDLE"), "blocked": ("red", "● BLOCKED"),
                "unknown": ("grey", "○ UNKNOWN")}

def evaluate(agent, checkin, now):
    st = (agent.get("agent_status") or "unknown").lower()
    color, label = STATUS_FAROL.get(st, ("grey", "○ " + st.upper()))
    started = parse_ts(checkin.get("started_at") or checkin.get("_ts")) if checkin else None
    finished = parse_ts(checkin.get("finished_at")) if checkin else None
    tempo = "—"
    if started and st == "working":
        tempo = "há " + fmt_dur((now - started).total_seconds())
    elif started and finished:
        tempo = "✓ " + fmt_dur((finished - started).total_seconds())
    elif started:
        tempo = "há " + fmt_dur((now - started).total_seconds())
    build = (checkin.get("build_result") or "").strip().strip('"') if checkin else ""
    notes = (checkin.get("notes") or "").strip().strip('"') if checkin else ""
    obs = ""
    if st == "blocked": obs = notes or build or "bloqueado"
    elif st == "idle": obs = notes or "idle (aguardando/entre tasks)"
    elif st == "unknown": obs = "sem deteccao de estado (screen-detection)"
    elif build and any(k in build.lower() for k in ("fail", "red", "error", "erro")):
        color, obs = "red", "build VERMELHO: " + build[:50]
    else: obs = notes
    stream = (checkin.get("stream") or "—") if checkin else "—"
    return {"farol": color, "label": label, "status": st, "stream": stream, "tempo": tempo, "obs": obs[:46]}

# ---------- render ----------
def clip(s, n):
    s = str(s or ""); return s if len(s) <= n else s[:n-1] + "…"

def build_rows(agents, board, now):
    rows = []
    for a in agents:
        ck = board.get(norm(a.get("name")))
        rows.append((a, evaluate(a, ck, now)))
    order = {"blocked": 0, "working": 1, "idle": 2, "unknown": 3, "done": 4}
    rows.sort(key=lambda r: order.get(r[1]["status"], 9))
    return rows

def render(host, agents, board, now, col, ascii_mode, err=None):
    W = {"agent": 22, "type": 9, "stream": 22, "farol": 11, "tempo": 12, "obs": 46}
    def cell(s, w): return clip(s, w).ljust(w)
    out = []
    out.append(col.bold(col.cyan(" FLEET DASHBOARD — Herdr-over-SSH (realtime) ")))
    out.append(col.grey(f" host: {host}   |   {now.strftime('%Y-%m-%d %H:%M:%SZ')}   |   fonte: herdr agent list (socket) + board"))
    if err:
        out.append(col.red(" ERRO SSH/Herdr: " + err.strip()[:100]))
    out.append("")
    out.append(col.bold(cell("AGENTE", W["agent"]) + " " + cell("TIPO", W["type"]) + " " +
                        cell("STREAM/TASK", W["stream"]) + " " + cell("FAROL", W["farol"]) + " " +
                        cell("TEMPO", W["tempo"]) + " " + cell("OBS", W["obs"])))
    out.append(col.grey("─" * (sum(W.values()) + len(W) - 1)))
    counts = {"green": 0, "yellow": 0, "red": 0, "grey": 0}
    rows = build_rows(agents, board, now)
    for a, ev in rows:
        counts[ev["farol"]] += 1
        paint = getattr(col, ev["farol"])
        farol_disp = ev["label"][:3] if ascii_mode else paint(cell(ev["label"], W["farol"]))
        line = (cell(a.get("name", "?"), W["agent"]) + " " +
                cell(a.get("agent", "—"), W["type"]) + " " +
                cell(ev["stream"], W["stream"]) + " " +
                (cell(ev["label"], W["farol"]) if ascii_mode else paint(cell(ev["label"], W["farol"]))) + " " +
                cell(ev["tempo"], W["tempo"]) + " " +
                (paint(cell(ev["obs"], W["obs"])) if ev["farol"] in ("red", "yellow") and ev["obs"] else cell(ev["obs"], W["obs"])))
        out.append(line)
    out.append(col.grey("─" * (sum(W.values()) + len(W) - 1)))
    total = len(rows) or 1
    out.append(f" {col.green('● WORKING/DONE ' + str(counts['green']))}   "
               f"{col.yellow('● IDLE ' + str(counts['yellow']))}   "
               f"{col.red('● BLOCKED ' + str(counts['red']))}   "
               f"{col.grey('○ UNKNOWN ' + str(counts['grey']))}   |   {total} agentes")
    out.append(col.grey(f" Tech-Lead → orquestrador (somente {ORCH}): "
                        f"--msg \"...\"  |  --status  |  --read"))
    return "\n".join(out)

# ---------- orchestrator channel (SOMENTE opus-4.8-orchestrator) ----------
def msg_orchestrator(host, text):
    rc, out, err = ssh(host, f"herdr agent send {shlex.quote(ORCH)} {shlex.quote(text)}")
    return rc, (out or err)

def read_orchestrator(host, lines=60):
    rc, out, err = ssh(host, f"herdr agent read {shlex.quote(ORCH)} --source recent --lines {int(lines)}")
    return rc, (out if rc == 0 else err)

# ---------- main ----------
def snapshot(host, board):
    rc, out, err = fetch(host, board)
    if rc != 0 and not out:
        return [], {}, (err or f"ssh rc={rc}")
    parts = out.split(SEP, 1)
    agents = parse_agents(parts[0])
    board_d = parse_board(parts[1]) if len(parts) > 1 else {}
    return agents, board_d, (None if agents else (err or "sem agentes"))

def main():
    ap = argparse.ArgumentParser(description="Fleet dashboard Herdr-over-SSH (realtime).")
    ap.add_argument("--ssh", default=DEFAULT_HOST)
    ap.add_argument("--board", default=DEFAULT_BOARD)
    ap.add_argument("--once", action="store_true")
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--ascii", action="store_true")
    ap.add_argument("--interval", type=float, default=5.0)
    ap.add_argument("--msg", metavar="TEXT", help=f"envia mensagem SOMENTE ao {ORCH}")
    ap.add_argument("--status", action="store_true", help=f"pede status ao {ORCH} e le a resposta")
    ap.add_argument("--read", action="store_true", help=f"le o pane do {ORCH}")
    args = ap.parse_args()
    col = C(use_color(args.ascii))
    host = args.ssh

    if args.msg is not None:
        rc, o = msg_orchestrator(host, args.msg)
        print("enviado ao orquestrador." if rc == 0 else f"falhou: {o}"); return
    if args.read:
        rc, o = read_orchestrator(host); print(o); return
    if args.status:
        rc, o = msg_orchestrator(host, "[Tech-Lead] status geral do fleet? resumo por agente + bloqueios, por favor.")
        print("pedido de status enviado ao orquestrador." if rc == 0 else f"falhou: {o}")
        time.sleep(3); rc, o = read_orchestrator(host); print("\n--- resposta (pane do orquestrador) ---\n" + o); return

    def frame():
        now = datetime.now(timezone.utc)
        agents, board_d, err = snapshot(host, args.board)
        if args.json:
            return json.dumps({"host": host, "generated_at": now.strftime("%Y-%m-%dT%H:%M:%SZ"),
                               "agents": [{**a, **evaluate(a, board_d.get(norm(a.get('name'))), now)} for a in agents]},
                              ensure_ascii=False, indent=2)
        return render(host, agents, board_d, now, col, args.ascii, err)

    if args.once or args.json:
        print(frame()); return

    tty = sys.stdout.isatty()
    ALT_ON, ALT_OFF = "\033[?1049h\033[?25l", "\033[?25h\033[?1049l"
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
    except KeyboardInterrupt:
        pass
    finally:
        if tty: sys.stdout.write(ALT_OFF); sys.stdout.flush()
        print("bye.")

if __name__ == "__main__":
    main()
