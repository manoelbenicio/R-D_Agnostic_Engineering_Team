# Agentic Prompts Hub (sob Automonous_Agentic)

## Estrutura
- `new_prompts/` — prompts ATUAIS, ainda NÃO consumidos (a usar / em uso agora).
- `archive/`     — prompts JÁ CONSUMIDOS (stream DONE / superseded).

## Regra de ciclo de vida (fixa)
> **Assim que um prompt é CONSUMIDO (stream DONE e validado), ele MOVE de
> `new_prompts/` → `archive/`.**
`new_prompts/` sempre mostra só o que ainda não foi usado.

## Convenção
- Nome: `PROMPT_<AGENTE>_<STREAM>.md`
- Template best-practice (XML: role/context/task/example/verification/persistence)
  + gate obrigatório SIGN-IN/OUT (agent+timestamp) no board ABSOLUTO
  `/mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/`.
- Nada inventado; arquivos disjuntos; hotspots = serial; verde no container antes de DONE.
