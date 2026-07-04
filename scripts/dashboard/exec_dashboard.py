#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Rotation-Parity Polyglot — Execution Dashboard (360 graus, realtime).

Cruza o manifesto canonico de tasks (.deploy-control/dashboard/tasks.json) com os
check-in/out reais dos agentes (.deploy-control/<AGENT>__<STREAM>__<UTC>.md) e renderiza
uma visao farol (verde/amarelo/vermelho/cinza) com: task, fase, agente, status, ETA e OBS.

Fonte da verdade = arquivos de check-in em disco (o mesmo protocolo que os agentes ja usam).

Uso:
  python3 scripts/dashboard/exec_dashboard.py             # ao vivo, refresh 5s
  python3 scripts/dashboard/exec_dashboard.py --once      # snapshot unico
  python3 scripts/dashboard/exec_dashboard.py --interval 3
  python3 scripts/dashboard/exec_dashboard.py --json      # saida machine-readable
  python3 scripts/dashboard/exec_dashboard.py --demo      # dados ilustrativos (nao toca o board)
  python3 scripts/dashboard/exec_dashboard.py --ascii     # sem cor/emoji (G/Y/R/.)

Sem dependencias externas (stdlib). Honra NO_COLOR.
"""
import argparse, json, os, re, sys, time, glob
from datetime import datetime, timezone

# ---------- localizacao ----------
def repo_root():
    here = os.path.dirname(os.path.abspath(__file__))
    # scripts/dashboard/ -> raiz = 2 niveis acima
    root = os.path.abspath(os.path.join(here, "..", ".."))
    return os.environ.get("RPP_REPO_ROOT", root)

# ---------- cores ----------
def use_color(ascii_mode):
    if ascii_mode or os.environ.get("NO_COLOR"): return False
    return sys.stdout.isatty() or os.environ.get("RPP_FORCE_COLOR") == "1"

class C:
    def __init__(self, on):
        self.on = on
    def _w(self, s, code): return f"\033[{code}m{s}\033[0m" if self.on else s
    def green(self, s):  return self._w(s, "32")
    def yellow(self, s): return self._w(s, "33")
    def red(self, s):    return self._w(s, "31")
    def grey(self, s):   return self._w(s, "90")
    def bold(self, s):   return self._w(s, "1")
    def cyan(self, s):   return self._w(s, "36")

# ---------- parsing ----------
def parse_ts(s):
    if not s: return None
    s = str(s).strip().strip('"').strip("'")
    if not s: return None
    fmts = ["%Y-%m-%dT%H:%M:%SZ", "%Y-%m-%dT%H:%M:%S%z", "%Y-%m-%dT%H:%M:%S",
            "%Y%m%dT%H%M%SZ", "%Y-%m-%d %H:%M:%S", "%Y-%m-%dT%H-%M-%SZ", "%Y-%m-%dT%H-%M-%S"]
    for f in fmts:
        try:
            dt = datetime.strptime(s, f)
            return dt.replace(tzinfo=timezone.utc) if dt.tzinfo is None else dt.astimezone(timezone.utc)
        except ValueError:
            continue
    return None

def parse_frontmatter(path):
    d, ts_from_name = {}, None
    m = re.search(r"__([0-9T:\-Z]+)\.md$", os.path.basename(path))
    if m: ts_from_name = m.group(1)
    try:
        with open(path, "r", encoding="utf-8", errors="replace") as fh:
            lines = fh.read().splitlines()
    except OSError:
        return None
    # front-matter no topo: aceita "key: value" e "- key: value" (drift tolerado)
    for ln in lines[:40]:
        mm = re.match(r"^\s*-?\s*([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*)$", ln)
        if mm:
            k, v = mm.group(1).strip().lower(), mm.group(2).strip()
            if k not in d and v != "":
                d[k] = v
    d["_file_ts"] = ts_from_name
    d["_path"] = path
    return d

def collect_checkins(board_dir):
    """retorna {stream_lower: record_mais_recente}"""
    out = {}
    for p in glob.glob(os.path.join(board_dir, "*__*__*.md")):
        rec = parse_frontmatter(p)
        if not rec: continue
        stream = (rec.get("stream") or "").strip().strip('"').lower()
        if not stream:
            m = re.search(r"__([A-Za-z0-9\.\-]+)__", os.path.basename(p))
            stream = m.group(1).lower() if m else ""
        if not stream: continue
        key = _sort_key(rec)
        if stream not in out or key > out[stream]["_key"]:
            rec["_key"] = key
            out[stream] = rec
    return out

def _sort_key(rec):
    t = parse_ts(rec.get("started_at")) or parse_ts(rec.get("_file_ts"))
    return t.timestamp() if t else 0.0

# ---------- calculo de farol/ETA/OBS ----------
def fmt_h(h):
    h = abs(h)
    hh = int(h); mm = int(round((h - hh) * 60))
    if mm == 60: hh += 1; mm = 0
    if hh and mm: return f"{hh}h{mm:02d}m"
    if hh: return f"{hh}h"
    return f"{mm}m"

def evaluate(task, rec, now):
    eta_h = float(task.get("eta_hours") or 0)
    if rec is None:
        eta = "planej. " + (fmt_h(eta_h) if eta_h else "cont.") if eta_h or task.get("phase") else "—"
        return {"farol": "grey", "status": "TODO", "eta": (fmt_h(eta_h) if eta_h else "cont."), "obs": ""}
    status = (rec.get("status") or "").upper().strip('"')
    started = parse_ts(rec.get("started_at")) or parse_ts(rec.get("_file_ts"))
    finished = parse_ts(rec.get("finished_at"))
    build = (rec.get("build_result") or "").strip().strip('"')
    notes = (rec.get("notes") or "").strip().strip('"')
    bl = build.lower()
    build_green = any(k in bl for k in ["green", "pass", "verde", "ok", "exit 0", "exit_status: 0", "real_exit=0"])
    build_red = any(k in bl for k in ["fail", "red", "vermelho", "error", "erro"])
    if status == "DONE":
        dur = fmt_h((finished - started).total_seconds() / 3600) if (started and finished) else "?"
        if build_red:
            return {"farol": "red", "status": "DONE*", "eta": dur, "obs": "build VERMELHO: " + build[:60]}
        if not build:
            return {"farol": "yellow", "status": "DONE*", "eta": dur, "obs": "concluido sem build_result (validar)"}
        return {"farol": "green", "status": "DONE", "eta": dur, "obs": notes[:60]}
    if status == "BLOCKED":
        return {"farol": "red", "status": "BLOCKED", "eta": "—", "obs": (notes or build or "bloqueado")[:60]}
    if status == "IN_PROGRESS":
        if started and eta_h > 0:
            elapsed = (now - started).total_seconds() / 3600
            rem = eta_h - elapsed
            if rem < 0:
                over = -rem
                farol = "red" if over > eta_h * 0.5 else "yellow"
                return {"farol": farol, "status": "EM CURSO", "eta": f"ATRASO {fmt_h(over)}",
                        "obs": (notes or f"passou da ETA em {fmt_h(over)}")[:60]}
            if rem < eta_h * 0.2:
                return {"farol": "yellow", "status": "EM CURSO", "eta": f"~{fmt_h(rem)} rest",
                        "obs": (notes or "proximo do limite de ETA")[:60]}
            return {"farol": "green", "status": "EM CURSO", "eta": f"~{fmt_h(rem)} rest", "obs": notes[:60]}
        return {"farol": "green", "status": "EM CURSO", "eta": "—", "obs": notes[:60]}
    return {"farol": "yellow", "status": (status or "?")[:10], "eta": "—", "obs": (notes or "status desconhecido")[:60]}

# ---------- render ----------
FAROL_TXT = {"green": ("● VERDE", "green", "G"), "yellow": ("● AMAREL", "yellow", "Y"),
             "red": ("● VERMEL", "red", "R"), "grey": ("○ TODO", "grey", ".")}

def clip(s, n):
    s = s or ""
    return s if len(s) <= n else s[: n - 1] + "…"

def render(manifest, checkins, now, col, ascii_mode):
    tasks = manifest.get("tasks", [])
    rows = []
    counts = {"green": 0, "yellow": 0, "red": 0, "grey": 0}
    for t in tasks:
        rec = checkins.get((t.get("stream") or "").lower())
        ev = evaluate(t, rec, now)
        counts[ev["farol"]] += 1
        rows.append((t, ev))
    total = len(tasks) or 1
    done = counts["green"]
    pct = int(round(100 * done / total))

    W = {"fase": 6, "task": 34, "agente": 17, "status": 9, "farol": 9, "eta": 14, "obs": 40}
    def cell(s, w): return clip(str(s), w).ljust(w)

    out = []
    title = f" ROTATION-PARITY POLYGLOT — EXECUTION DASHBOARD (360°) "
    ts = now.strftime("%Y-%m-%d %H:%M:%SZ")
    out.append(col.bold(col.cyan(title)))
    out.append(col.grey(f" board: {manifest.get('_board_dir','')}   |   {ts}   |   fonte: check-ins em disco"))
    out.append("")
    header = (col.bold(cell("FASE", W["fase"])) + " " + col.bold(cell("TASK", W["task"])) + " " +
              col.bold(cell("AGENTE", W["agente"])) + " " + col.bold(cell("STATUS", W["status"])) + " " +
              col.bold(cell("FAROL", W["farol"])) + " " + col.bold(cell("ETA", W["eta"])) + " " +
              col.bold(cell("OBS", W["obs"])))
    out.append(header)
    out.append(col.grey("─" * (sum(W.values()) + len(W) - 1)))
    for t, ev in rows:
        label, color, ascii_c = FAROL_TXT[ev["farol"]]
        farol_disp = ascii_c + " " + ev["status"] if ascii_mode else getattr(col, color)(label)
        paint = getattr(col, color)
        line = (cell(t.get("phase", ""), W["fase"]) + " " +
                cell(t.get("name", ""), W["task"]) + " " +
                cell(t.get("owner", ""), W["agente"]) + " " +
                cell(ev["status"], W["status"]) + " " +
                (cell(ascii_c, W["farol"]) if ascii_mode else paint(cell(label, W["farol"]))) + " " +
                cell(ev["eta"], W["eta"]) + " " +
                (paint(cell(ev["obs"], W["obs"])) if ev["farol"] in ("yellow", "red") and ev["obs"] else cell(ev["obs"], W["obs"])))
        gated = " ⚑" if t.get("gated") else ""
        out.append(line + gated)
    out.append(col.grey("─" * (sum(W.values()) + len(W) - 1)))
    summary = (f" {col.green('● VERDE ' + str(counts['green']))}   "
               f"{col.yellow('● AMARELO ' + str(counts['yellow']))}   "
               f"{col.red('● VERMELHO ' + str(counts['red']))}   "
               f"{col.grey('○ TODO ' + str(counts['grey']))}   |   "
               f"CONCLUÍDO {done}/{total} ({pct}%)   |   ⚑ = gated (deploy espera runbook do dono)")
    out.append(summary)
    # barra de progresso
    barw = 40
    fill = int(round(barw * done / total))
    bar = col.green("█" * fill) + col.grey("░" * (barw - fill))
    out.append(f" [{bar}] {pct}%")
    return "\n".join(out)

def to_json(manifest, checkins, now):
    now_iso = now.strftime("%Y-%m-%dT%H:%M:%SZ")
    items = []
    for t in manifest.get("tasks", []):
        rec = checkins.get((t.get("stream") or "").lower())
        ev = evaluate(t, rec, now)
        items.append({"stream": t.get("stream"), "phase": t.get("phase"), "name": t.get("name"),
                       "owner": t.get("owner"), "gated": bool(t.get("gated")),
                       "farol": ev["farol"], "status": ev["status"], "eta": ev["eta"], "obs": ev["obs"]})
    counts = {k: sum(1 for i in items if i["farol"] == k) for k in ("green", "yellow", "red", "grey")}
    return json.dumps({"generated_at": now_iso, "project": manifest.get("project"),
                        "counts": counts, "total": len(items),
                        "done_pct": int(round(100 * counts["green"] / (len(items) or 1))),
                        "tasks": items}, ensure_ascii=False, indent=2)

# ---------- demo ----------
def demo_checkins():
    now = datetime.now(timezone.utc)
    def iso(dt): return dt.strftime("%Y-%m-%dT%H:%M:%SZ")
    from datetime import timedelta
    return {
        "rpp-ops":        {"status": "IN_PROGRESS", "started_at": iso(now - timedelta(hours=1)), "notes": ""},
        "rpp-vendormatrix": {"status": "DONE", "started_at": iso(now - timedelta(hours=5)), "finished_at": iso(now - timedelta(hours=1)), "build_result": "verde: matriz completa"},
        "rpp-contract":   {"status": "DONE", "started_at": iso(now - timedelta(hours=6)), "finished_at": iso(now - timedelta(hours=1)), "build_result": ""},
        "rpp-state":      {"status": "IN_PROGRESS", "started_at": iso(now - timedelta(hours=2)), "notes": ""},
        "rpp-forkmap":    {"status": "IN_PROGRESS", "started_at": iso(now - timedelta(hours=9)), "notes": "aguardando doc oficial do prodex"},
        "rpp-qa":         {"status": "BLOCKED", "started_at": iso(now - timedelta(hours=1)), "notes": "depende do contrato v0 (RPP-CONTRACT) publicado"},
    }

# ---------- main ----------
def load_manifest(root):
    path = os.path.join(root, ".deploy-control", "dashboard", "tasks.json")
    with open(path, "r", encoding="utf-8") as fh:
        m = json.load(fh)
    m["_board_dir"] = os.path.join(root, m.get("board_dir", ".deploy-control"))
    return m

def snapshot(root, demo):
    manifest = load_manifest(root)
    if demo:
        ci = demo_checkins()
        for k, v in ci.items():
            v.setdefault("_file_ts", None)
        checkins = {k: {**v, "_key": _sort_key(v)} for k, v in ci.items()}
    else:
        checkins = collect_checkins(manifest["_board_dir"])
    return manifest, checkins

def main():
    ap = argparse.ArgumentParser(description="Rotation-Parity Polyglot execution dashboard (360, realtime).")
    ap.add_argument("--once", action="store_true")
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--demo", action="store_true")
    ap.add_argument("--ascii", action="store_true")
    ap.add_argument("--interval", type=float, default=5.0)
    ap.add_argument("--root", default=repo_root())
    args = ap.parse_args()
    col = C(use_color(args.ascii))

    def once():
        now = datetime.now(timezone.utc)
        manifest, checkins = snapshot(args.root, args.demo)
        if args.json:
            print(to_json(manifest, checkins, now)); return
        print(render(manifest, checkins, now, col, args.ascii))

    if args.once or args.json:
        once(); return
    try:
        while True:
            sys.stdout.write("\033[2J\033[H")
            once()
            sys.stdout.write("\n" + col.grey(f" (atualiza a cada {args.interval:g}s — Ctrl+C p/ sair)") + "\n")
            sys.stdout.flush()
            time.sleep(args.interval)
    except KeyboardInterrupt:
        print("\nbye.")

if __name__ == "__main__":
    main()
