# SEV-0 QA — plan_dashboard.py → VEREDITO: SHIP
- Suíte base: 29/29 PASS (scripts/dashboard/test_plan_dashboard.py)
- Bateria SEV-0: 20/20 PASS (scripts/dashboard/test_plan_dashboard_sev0.py)
  - A. 12 edge cases hostis (CRLF/BOM/unicode/malformado/gigante/lixo/vazio) — parse+render sem crash + invariante done<=total
  - B. Fuzz 200 arquivos aleatorios — 0 crash, contagem consistente
  - C. Property: overall == soma dos grupos
  - D. Encoding hostil ascii/latin-1/cp1252 — CLI rc=0 (regressao ISSUE-003 travada)
- Re-rodar: python3 scripts/dashboard/test_plan_dashboard{,_sev0}.py
