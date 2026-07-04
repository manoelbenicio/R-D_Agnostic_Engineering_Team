# Agentic Prompts Hub

Fonte de verdade dos prompts de agentes (projeto Automonous_Agentic).

## Estrutura
- **Raiz (`./PROMPT_*.md`)** — prompts A SEREM USADOS AGORA. Só o que está em uso.
- **`archive/`** — prompts JÁ UTILIZADOS (concluídos / superseded). Histórico.

## Ciclo de vida
1. Prompt novo nasce na RAIZ do hub.
2. Quando o stream é DONE (validado pelo Opus) → move p/ `archive/`.
3. A raiz sempre reflete só o que os agentes vão rodar / estão rodando agora.

## Convenção
- Nome: `PROMPT_<AGENTE>_<STREAM>.md`
- Template best-practice (XML: role/context/task/example/verification/persistence)
  + gate obrigatório SIGN-IN/OUT (agent+timestamp) no board ABSOLUTO
  `/mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/`.
- Nada inventado; arquivos disjuntos; hotspots = serial; verde no container antes de DONE.

## Estado atual (2026-07-03)
- Raiz: rotation-router Wave 1 → RR-POLICY, RR-FALLBACK, RR-REGISTRY, RR-OBSERV
- archive/: 17 concluídos
