#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
PLAN DASHBOARD — progresso REAL do plano, lendo a fonte de verdade local:
o `tasks.md` do OpenSpec (checkboxes `- [ ]` / `- [x]`). Sem SSH, sem Herdr, sem inferência.

Uso:
  python3 scripts/dashboard/plan_dashboard.py            # snapshot
  python3 scripts/dashboard/plan_dashboard.py --watch    # atualiza a cada 5s (Ctrl+C sai)
  python3 scripts/dashboard/plan_dashboard.py --json
  python3 scripts/dashboard/plan_dashboard.py --tasks <caminho/tasks.md>
"""
import argparse, os, re, sys, time, json

DEFAULT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..",
                       "openspec", "changes", "rotation-parity-polyglot", "tasks.md")

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
    """Retorna [(grupo, [(num, done, texto), ...]), ...]"""
    groups, cur = [], None
    try:
        lines = open(path, encoding="utf-8").read().splitlines()
    except OSError as e:
        return [], str(e)
    for ln in lines:
        mg = re.match(r"^##\s+(.*)$", ln)
        if mg:
            cur = (mg.group(1).strip(), [])
            groups.append(cur); continue
        mt = re.match(r"^\s*-\s*\[([ xX])\]\s*([0-9]+\.[0-9]+)?\s*(.*)$", ln)
        if mt and cur is not None:
            done = mt.group(1).lower() == "x"
            cur[1].append((mt.group(2) or "", done, mt.group(3).strip()))
    return [(g, t) for g, t in groups if t], None

def bar(col, done, total, width=22):
    pct = int(round(100 * done / total)) if total else 0
    fill = int(round(width * pct / 100))
    tier = col.g if pct == 100 else (col.y if pct >= 40 else col.r)
    return tier("█" * fill) + col.gr("░" * (width - fill)) + " " + tier(f"{pct:3d}%")

def render(path, col):
    groups, err = parse(path)
    L = []
    title = " PLAN DASHBOARD — Rotation-Parity Polyglot (OpenSpec tasks) "
    L.append(col.cy("┌" + "─" * 72 + "┐"))
    L.append(col.cy("│") + col.bo(col.cy(title.center(72))) + col.cy("│"))
    L.append(col.cy("│") + col.gr(f" fonte: {os.path.relpath(path)} ".ljust(72)[:72]) + col.cy("│"))
    L.append(col.cy("└" + "─" * 72 + "┘"))
    if err:
        L.append(col.r(f" ERRO ao ler tasks.md: {err}")); return "\n".join(L)
    if not groups:
        L.append(col.r(" nenhuma task encontrada (formato `- [ ] N.M`?)")); return "\n".join(L)
    td = tt = 0
    L.append("")
    L.append(" " + col.bo(f'{"FASE":<44}{"FEITO":>7}  PROGRESSO'))
    L.append(" " + col.gr("─" * 70))
    for g, tasks in groups:
        d = sum(1 for _, done, _ in tasks if done); n = len(tasks)
        td += d; tt += n
        name = re.sub(r"\s*\[.*?\]\s*$", "", g)[:44]
        mark = col.g("●") if d == n else (col.y("▶") if d else col.gr("○"))
        L.append(f" {mark} {name:<42}{col.gr(f'{d}/{n}'):>16}  {bar(col, d, n)}")
    L.append(" " + col.gr("─" * 70))
    ov = int(round(100 * td / tt)) if tt else 0
    faltam = tt - td
    ovbar = bar(col, td, tt, 30)
    L.append(" " + col.bo(f"OVERALL  ") + ovbar + "   " +
             col.g(f"✔ {td} feitas") + "   " + col.y(f"▢ {faltam} restam") + f"   ({tt} tasks)")
    # próxima fase não concluída
    nxt = next(((g, tasks) for g, tasks in groups if any(not d for _, d, _ in tasks)), None)
    if nxt:
        g, tasks = nxt
        pend = [f"{num} {txt}" for num, d, txt in tasks if not d][:3]
        L.append(" " + col.bo("PRÓXIMA: ") + col.cy(re.sub(r"\s*\[.*?\]\s*$", "", g)))
        for p in pend: L.append("   " + col.gr("• " + p[:66]))
    else:
        L.append(" " + col.g("✔ Todas as tasks concluídas."))
    return "\n".join(L)

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--tasks", default=os.path.abspath(DEFAULT))
    ap.add_argument("--watch", action="store_true")
    ap.add_argument("--interval", type=float, default=5.0)
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--ascii", action="store_true")
    a = ap.parse_args()
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
        print(render(a.tasks, col)); return
    tty = sys.stdout.isatty()
    try:
        if tty: sys.stdout.write("\033[?1049h\033[?25l")
        while True:
            f = render(a.tasks, col) + "\n " + col.gr(f"(watch {a.interval:g}s — Ctrl+C sai)")
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
