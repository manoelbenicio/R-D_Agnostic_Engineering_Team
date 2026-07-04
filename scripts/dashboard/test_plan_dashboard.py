#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""QA suite do plan_dashboard.py — unit (parse) + integração (CLI) + edge cases.
Roda tudo e imprime PASS/FAIL por caso. Exit 0 = tudo verde."""
import importlib.util, json, os, re, subprocess, sys, tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
DASH = os.path.join(HERE, "plan_dashboard.py")

spec = importlib.util.spec_from_file_location("pd", DASH)
pd = importlib.util.module_from_spec(spec); spec.loader.exec_module(pd)

results = []
def check(name, cond, detail=""):
    results.append((name, bool(cond), detail))
    print(f"  [{'PASS' if cond else 'FAIL'}] {name}" + (f" — {detail}" if detail and not cond else ""))

def write_tmp(content):
    f = tempfile.NamedTemporaryFile("w", suffix=".md", delete=False, encoding="utf-8")
    f.write(content); f.close(); return f.name

def run(args):
    p = subprocess.run([sys.executable, DASH] + args, capture_output=True, text=True, timeout=15,
                       env={**os.environ, "NO_COLOR": "1"})
    return p.returncode, p.stdout, p.stderr

print("== 1. UNIT: parse() ==")
# mixed fixture
mixed = write_tmp("## 1. A\n\n- [x] 1.1 feita\n- [ ] 1.2 pendente\n\n## 2. B\n\n- [x] 2.1 feita\n- [x] 2.2 feita\n")
g, err = pd.parse(mixed)
check("parse sem erro", err is None, str(err))
check("2 grupos", len(g) == 2, f"got {len(g)}")
check("grupo1 1/2 done", sum(1 for _,d,_ in g[0][1] if d) == 1)
check("grupo2 2/2 done", sum(1 for _,d,_ in g[1][1] if d) == 2)

# no numbering
nonum = write_tmp("## X\n\n- [ ] task sem numero\n- [x] outra\n")
g2, _ = pd.parse(nonum)
check("parse sem numeracao", len(g2) == 1 and len(g2[0][1]) == 2)

# empty / no tasks
empty = write_tmp("# titulo\n\ntexto sem tasks\n")
g3, e3 = pd.parse(empty)
check("empty -> 0 grupos, sem crash", g3 == [] and e3 is None)

# missing file
gm, em = pd.parse("/caminho/inexistente_xyz.md")
check("arquivo ausente -> erro tratado (sem crash)", gm == [] and em is not None)

# X maiusculo
caps = write_tmp("## 1. G\n- [X] 1.1 feita maiuscula\n")
g4, _ = pd.parse(caps)
check("checkbox [X] maiusculo conta como done", g4 and g4[0][1][0][1] is True)

print("== 2. CLI: --json ==")
rc, out, errs = run(["--tasks", mixed, "--json"])
check("json exit 0", rc == 0, f"rc={rc} err={errs[:120]}")
try:
    j = json.loads(out); okj = True
except Exception as ex:
    okj = False; j = {}
    check("json parseavel", False, str(ex))
if okj:
    check("json parseavel", True)
    check("json overall_done=3", j.get("overall_done") == 3, str(j.get("overall_done")))
    check("json overall_total=4", j.get("overall_total") == 4, str(j.get("overall_total")))
    check("json 2 phases", len(j.get("phases", [])) == 2)

print("== 3. CLI: --ascii (sem ANSI) ==")
rc, out, _ = run(["--tasks", mixed, "--ascii"])
check("ascii exit 0", rc == 0)
check("ascii sem codigos ANSI", "\033[" not in out, "encontrou ANSI")
check("ascii mostra OVERALL", "OVERALL" in out)
check("ascii mostra 3/4-ish overall (75%)", "75%" in out, "esperava 75%")

print("== 4. CLI: arquivo real do projeto ==")
real = os.path.join(HERE, "..", "..", "openspec", "changes", "rotation-parity-polyglot", "tasks.md")
rc, out, _ = run(["--tasks", real, "--ascii"])
check("real exit 0", rc == 0)
import json as _j; _tot=_j.loads(run(["--tasks", real, "--json"])[1])["overall_total"]; check("real: tasks>=50 e bate json", _tot>=50 and (f"{_tot} tasks" in out), f"tot={_tot}")
check("real: 11 fases (0..10)", len(re.findall(r"\b\d+/\d+\b", out)) >= 11, f'{len(re.findall(r"\\b\\d+/\\d+\\b", out))} rows')

print("== 5. CLI: consistencia json x ascii (overall) ==")
_, oj, _ = run(["--tasks", real, "--json"]); jr = json.loads(oj)
_, oa, _ = run(["--tasks", real, "--ascii"])
pct = int(round(100*jr["overall_done"]/jr["overall_total"])) if jr["overall_total"] else 0
check("overall json bate com ascii", f"{pct:3d}%" in oa or f"{pct}%" in oa, f"pct={pct}")

print("== 6. CLI: arquivo ausente nao crasha ==")
rc, out, _ = run(["--tasks", "/nao/existe_qa.md", "--ascii"])
check("ausente exit 0 (graceful)", rc == 0)
check("ausente mostra ERRO", "ERRO" in out or "erro" in out)

print("== 7. --watch renderiza e sai limpo (timeout) ==")
try:
    p = subprocess.run([sys.executable, DASH, "--tasks", mixed, "--watch", "--interval", "0.3"],
                       capture_output=True, text=True, timeout=2,
                       env={**os.environ, "NO_COLOR": "1"})
    watched = p.stdout
except subprocess.TimeoutExpired as ex:
    watched = (ex.stdout or b"").decode() if isinstance(ex.stdout, bytes) else (ex.stdout or "")
check("watch produziu frame", "OVERALL" in (watched or ""))

# limpeza
for f in (mixed, nonum, empty, caps):
    try: os.unlink(f)
    except OSError: pass

print("== 8. ENCODING SAFETY (o bug que quebrou na tela do usuario) ==")
def run_enc(enc, args):
    p = subprocess.run([sys.executable, DASH] + args, capture_output=True, text=True, timeout=15,
                       env={**os.environ, "PYTHONIOENCODING": enc})
    return p.returncode, p.stdout, p.stderr
for enc in ("ascii", "latin-1", "cp1252"):
    rc, out, er = run_enc(enc, ["--tasks", real])
    check(f"snapshot nao crasha em {enc}", rc == 0 and "OVERALL" in out, f"rc={rc} err={er[:80]}")
# modo --ascii deve usar glifos ascii (sem unicode)
rc, out, _ = run(["--tasks", real, "--ascii"])
check("--ascii sem unicode (sem box/bar unicode)", not any(c in out for c in "█●░┌┐└┘│→✔○▶▢"), "ainda tem unicode")
check("--ascii usa glifos ascii (# e/ou [ ])", ("#" in out or "[ ]" in out or "[x]" in out))

print("\n== RESUMO ==")
passed = sum(1 for _, ok, _ in results if ok); total = len(results)
print(f"  {passed}/{total} PASS")
fails = [n for n, ok, _ in results if not ok]
if fails:
    print("  FALHAS:", ", ".join(fails)); sys.exit(1)
print("  TODOS VERDES ✅"); sys.exit(0)
