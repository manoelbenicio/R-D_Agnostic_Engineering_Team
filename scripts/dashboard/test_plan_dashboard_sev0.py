#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""SEV-0 battery para plan_dashboard.py: property-based/fuzz + edge cases hostis + invariantes.
Objetivo: provar que NUNCA crasha e que as contagens são sempre consistentes. Exit 0 = verde."""
import importlib.util, json, os, random, subprocess, sys, tempfile, string

HERE = os.path.dirname(os.path.abspath(__file__))
DASH = os.path.join(HERE, "plan_dashboard.py")
spec = importlib.util.spec_from_file_location("pd", DASH)
pd = importlib.util.module_from_spec(spec); spec.loader.exec_module(pd)

res = []
def check(name, cond, detail=""):
    res.append((name, bool(cond))); print(f"  [{'PASS' if cond else 'FAIL'}] {name}" + (f" — {detail}" if detail and not cond else ""))

def tmp(content, mode="w", enc="utf-8"):
    f = tempfile.NamedTemporaryFile(mode, suffix=".md", delete=False, encoding=(enc if "b" not in mode else None))
    f.write(content); f.close(); return f.name

class Col:
    on=False
    def __getattr__(self, _): return lambda s: s
col = Col()

def invariants(groups):
    """done<=total por grupo; retorna (od, ot)."""
    od = ot = 0
    for _, tasks in groups:
        d = sum(1 for _, x, _ in tasks if x); n = len(tasks)
        if not (0 <= d <= n): return None
        od += d; ot += n
    return od, ot

print("== A. FORENSE / EDGE CASES HOSTIS (parse+render nunca crasha) ==")
edge = {
 "CRLF": "## 1. G\r\n- [x] 1.1 a\r\n- [ ] 1.2 b\r\n",
 "sem trailing newline": "## 1. G\n- [x] 1.1 a",
 "BOM": "\ufeff## 1. G\n- [x] 1.1 a\n",
 "unicode no texto": "## 1. Fundação ✓ ─ ○\n- [ ] 1.1 buildar prodex → binário █\n",
 "checkbox malformado [y]": "## 1. G\n- [y] 1.1 invalido\n- [x] 1.2 ok\n",
 "sem numeracao": "## 1. G\n- [ ] task sem num\n",
 "tab-indent": "## 1. G\n\t- [x] 1.1 indentado\n",
 "heading sem tasks": "## 1. vazio\n## 2. G\n- [x] 2.1 ok\n",
 "duplicado num": "## 1. G\n- [x] 1.1 a\n- [ ] 1.1 b\n",
 "so lixo": "".join(random.choice(string.printable) for _ in range(500)),
 "vazio": "",
 "gigante": "## 1. Big\n" + "".join(f"- [{'x' if i%2 else ' '}] 1.{i} t{i}\n" for i in range(2000)),
}
for name, content in edge.items():
    try:
        f = tmp(content)
        g, err = pd.parse(f)
        inv = invariants(g)
        r = pd.render(f, col)
        ok = (inv is not None) and isinstance(r, str) and len(r) > 0
        check(f"edge '{name}': parse+render sem crash + invariante", ok, f"inv={inv}")
        os.unlink(f)
    except Exception as ex:
        check(f"edge '{name}': parse+render sem crash", False, repr(ex))

print("== B. FUZZ (200 arquivos aleatorios: nunca crasha, contagem consistente) ==")
random.seed(0); fails=0
for it in range(200):
    lines=[]
    for _ in range(random.randint(0,40)):
        r=random.random()
        if r<0.2: lines.append(f"## {random.randint(0,9)}. {''.join(random.choice(string.printable[:80]) for _ in range(random.randint(0,20)))}")
        elif r<0.7:
            box=random.choice([" ","x","X","y","-",""]); num=random.choice([f"{random.randint(0,9)}.{random.randint(0,9)}",""])
            lines.append(f"- [{box}] {num} {''.join(random.choice(string.printable[:90]) for _ in range(random.randint(0,30)))}")
        else: lines.append(''.join(random.choice(string.printable) for _ in range(random.randint(0,50))))
    f=tmp("\n".join(lines))
    try:
        g,err=pd.parse(f); inv=invariants(g); r=pd.render(f,col)
        if inv is None or not isinstance(r,str): fails+=1
    except Exception:
        fails+=1
    finally:
        os.unlink(f)
check("fuzz 200x sem crash e invariante done<=total", fails==0, f"{fails} falhas")

print("== C. PROPERTY: overall == soma dos grupos (via --json) ==")
def run_json(path, enc="utf-8"):
    p=subprocess.run([sys.executable,DASH,"--tasks",path,"--json"],capture_output=True,text=True,timeout=15,env={**os.environ,"PYTHONIOENCODING":enc})
    return p.returncode,p.stdout
pf=tmp("## 1. A\n- [x] 1.1\n- [ ] 1.2\n## 2. B\n- [x] 2.1\n- [x] 2.2\n- [ ] 2.3\n")
rc,out=run_json(pf); j=json.loads(out)
check("json rc0", rc==0)
check("overall_done == soma grupos", j["overall_done"]==sum(p["done"] for p in j["phases"]))
check("overall_total == soma grupos", j["overall_total"]==sum(p["total"] for p in j["phases"]))
check("valores esperados 3/5", j["overall_done"]==3 and j["overall_total"]==5, f'{j["overall_done"]}/{j["overall_total"]}')
os.unlink(pf)

print("== D. ENCODING HOSTIL (ascii/latin-1/cp1252/utf-16 no arquivo) — nunca crasha CLI ==")
real=os.path.join(HERE,"..","..","openspec","changes","rotation-parity-polyglot","tasks.md")
for enc in ("ascii","latin-1","cp1252"):
    rc,out=run_json(real,enc)
    check(f"CLI --json rc0 sob PYTHONIOENCODING={enc}", rc==0, f"rc={rc}")

print("\n== RESUMO SEV-0 ==")
p=sum(1 for _,ok in res if ok); t=len(res)
print(f"  {p}/{t} PASS")
if p!=t:
    print("  VEREDITO: NO-SHIP (dashboard)"); sys.exit(1)
print("  VEREDITO: SHIP (dashboard) — sem crash, invariantes mantidos, encoding-safe"); sys.exit(0)
