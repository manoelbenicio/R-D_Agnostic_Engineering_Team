#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
PLAN DASHBOARD — progresso REAL do plano, lendo a fonte de verdade:
o `tasks.md` do OpenSpec (checkboxes `- [ ]` / `- [x]`).
Com --remote, faz fetch do fleet antes de cada render.
Encoding-safe: força UTF-8 e cai para ASCII automaticamente em terminal que não suporta unicode.

Uso:
  python3 scripts/dashboard/plan_dashboard.py            # snapshot local
  python3 scripts/dashboard/plan_dashboard.py --watch    # atualiza a cada 5s (Ctrl+C sai)
  python3 scripts/dashboard/plan_dashboard.py --watch --remote  # fetch do fleet a cada ciclo
  python3 scripts/dashboard/plan_dashboard.py --json | --ascii | --tasks <path>
"""
import argparse, os, re, sys, time, json, subprocess

DEFAULT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..",
                       "openspec", "changes", "rotation-parity-polyglot", "tasks.md")

# ---- encoding safety: tenta UTF-8; senão detecta e usa ASCII ----
def _harden_stdout():
    try:
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
        return True
    except Exception:
        return False

def _unicode_ok():
    enc = (getattr(sys.stdout, "encoding", None) or "").lower().replace("-", "")
    if enc in ("utf8", "utf16", "utf32", "utf"):
        return True
    try:
        "█●░→✔".encode(sys.stdout.encoding or "ascii"); return True
    except Exception:
        return False

# glifos: unicode vs ascii
GU = {"tl":"┌","tr":"┐","bl":"└","br":"┘","h":"─","v":"│","full":"█","empty":"░",
      "done":"●","work":"▶","todo":"○","check":"✔","box":"▢","dot":"•","arrow":"→"}
GA = {"tl":"+","tr":"+","bl":"+","br":"+","h":"-","v":"|","full":"#","empty":".",
      "done":"[x]","work":"[~]","todo":"[ ]","check":"OK","box":"..","dot":"*","arrow":"->"}
G = GU  # definido em main()

def use_color(ascii_mode):
    if ascii_mode or os.environ.get("NO_COLOR"): return False
    return sys.stdout.isatty() or os.environ.get("RPP_FORCE_COLOR") == "1"

class C:
    def __init__(self, on): self.on = on
    def w(self, s, c): return f"\033[{c}m{s}\033[0m" if self.on else s
    def g(self, s): return self.w(s, "32")
    def y(self, s): return self.w(s, "33")
    def r(self, s): return self.w(s, "31")
    def gr(self, s): return self.w(s, "90")
    def cy(self, s): return self.w(s, "36")
    def bo(self, s): return self.w(s, "1")

def parse(path):
    groups, cur = [], None
    try:
        lines = open(path, encoding="utf-8").read().splitlines()
    except OSError as e:
        return [], str(e)
    for ln in lines:
        mg = re.match(r"^##\s+(.*)$", ln)
        if mg:
            cur = (mg.group(1).strip(), []); groups.append(cur); continue
        mt = re.match(r"^\s*-\s*\[([ xX])\]\s*([0-9]+\.[0-9]+)?\s*(.*)$", ln)
        if mt and cur is not None:
            cur[1].append((mt.group(2) or "", mt.group(1).lower() == "x", mt.group(3).strip()))
    return [(g, t) for g, t in groups if t], None

def bar(col, done, total, width=22):
    pct = int(round(100 * done / total)) if total else 0
    fill = int(round(width * pct / 100))
    tier = col.g if pct == 100 else (col.y if pct >= 40 else col.r)
    return tier(G["full"] * fill) + col.gr(G["empty"] * (width - fill)) + " " + tier(f"{pct:3d}%")

def render(path, col):
    groups, err = parse(path)
    L = []
    title = " PLAN DASHBOARD - Rotation-Parity Polyglot (OpenSpec tasks) "
    L.append(col.cy(G["tl"] + G["h"] * 72 + G["tr"]))
    L.append(col.cy(G["v"]) + col.bo(col.cy(title.center(72))) + col.cy(G["v"]))
    L.append(col.cy(G["v"]) + col.gr(f" fonte: {os.path.relpath(path)} ".ljust(72)[:72]) + col.cy(G["v"]))
    L.append(col.cy(G["bl"] + G["h"] * 72 + G["br"]))
    if err:
        L.append(col.r(f" ERRO ao ler tasks.md: {err}")); return "\n".join(L)
    if not groups:
        L.append(col.r(" nenhuma task encontrada (formato `- [ ] N.M`?)")); return "\n".join(L)
    td = tt = 0
    L.append("")
    L.append(" " + col.bo(f'{"FASE":<44}{"FEITO":>7}  PROGRESSO'))
    L.append(" " + col.gr(G["h"] * 70))
    for g, tasks in groups:
        d = sum(1 for _, done, _ in tasks if done); n = len(tasks)
        td += d; tt += n
        name = re.sub(r"\s*\[.*?\]\s*$", "", g)[:44]
        mark = col.g(G["done"]) if d == n else (col.y(G["work"]) if d else col.gr(G["todo"]))
        L.append(f" {mark} {name:<42}{col.gr(f'{d}/{n}'):>16}  {bar(col, d, n)}")
    L.append(" " + col.gr(G["h"] * 70))
    faltam = tt - td
    L.append(" " + col.bo("OVERALL  ") + bar(col, td, tt, 30) + "   " +
             col.g(f"{G['check']} {td} feitas") + "   " + col.y(f"{G['box']} {faltam} restam") + f"   ({tt} tasks)")
    nxt = next(((g, tasks) for g, tasks in groups if any(not d for _, d, _ in tasks)), None)
    if nxt:
        g, tasks = nxt
        pend = [f"{num} {txt}" for num, d, txt in tasks if not d][:3]
        L.append(" " + col.bo("PROXIMA: ") + col.cy(re.sub(r"\s*\[.*?\]\s*$", "", g)))
        for p in pend: L.append("   " + col.gr(G["dot"] + " " + p[:66]))
    else:
        L.append(" " + col.g(f"{G['check']} Todas as tasks concluidas."))
    return "\n".join(L)

REMOTE_HOST = "dataops-lab@192.168.1.27"
REMOTE_TASKS = "/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/rotation-parity-polyglot/tasks.md"
SSH_OPTS = ["-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "-o", "KexAlgorithms=curve25519-sha256"]

def sync_remote(local_path):
    """Fetch latest tasks.md from fleet host via SCP."""
    try:
        subprocess.run(
            ["scp"] + SSH_OPTS + [f"{REMOTE_HOST}:{REMOTE_TASKS}", local_path],
            capture_output=True, timeout=10
        )
    except Exception:
        pass  # silently fall back to local copy

def main():
    _harden_stdout()
    ap = argparse.ArgumentParser()
    ap.add_argument("--tasks", default=os.path.abspath(DEFAULT))
    ap.add_argument("--watch", action="store_true")
    ap.add_argument("--interval", type=float, default=5.0)
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--ascii", action="store_true")
    ap.add_argument("--remote", action="store_true", help="Fetch tasks.md from fleet before each render")
    a = ap.parse_args()
    global G
    G = GU if (_unicode_ok() and not a.ascii) else GA
    col = C(use_color(a.ascii))
    if a.json:
        groups, err = parse(a.tasks)
        out = {"tasks_file": a.tasks, "error": err, "phases": [
            {"phase": g, "done": sum(1 for _, d, _ in t if d), "total": len(t),
             "tasks": [{"num": n, "done": d, "text": x} for n, d, x in t]} for g, t in groups]}
        out["overall_done"] = sum(p["done"] for p in out["phases"])
        out["overall_total"] = sum(p["total"] for p in out["phases"])
        print(json.dumps(out, ensure_ascii=False, indent=2)); return
    if not a.watch:
        if getattr(a, 'remote', False): sync_remote(a.tasks)
        print(render(a.tasks, col)); return
    tty = sys.stdout.isatty()
    remote_label = " [REMOTE SYNC]" if getattr(a, 'remote', False) else ""
    try:
        if tty: sys.stdout.write("\033[?1049h\033[?25l")
        while True:
            if getattr(a, 'remote', False): sync_remote(a.tasks)
            f = render(a.tasks, col) + "\n " + col.gr(f"(watch {a.interval:g}s{remote_label} - Ctrl+C sai)")
            if tty:
                sys.stdout.write("\033[H")
                for ln in f.split("\n"): sys.stdout.write(ln + "\033[K\n")
                sys.stdout.write("\033[J")
            else:
                sys.stdout.write("\033[2J\033[H" + f + "\n")
            sys.stdout.flush(); time.sleep(a.interval)
    except KeyboardInterrupt: pass
    finally:
        if tty: sys.stdout.write("\033[?25h\033[?1049l"); sys.stdout.flush()

if __name__ == "__main__":
    main()
