# Spec - credential-account-home

## ADDED Requirements

### Requirement: REQ-01 Canonical authority and fail-closed boundary

This change MUST govern Runtime Standards, Runtime Sessions, workspace bindings, runtime
configuration, account-home selection, and task snapshots. Every launch MUST use the owner-approved
`native_credential_home` binding. OmniRouter/OmniRoute MUST NOT be a current solution, credential
authority, dependency, acceptance source, or fallback.

#### Scenario: Router binding requested
- **WHEN** configuration requests OmniRouter, OmniRoute, a credentialless gateway, or another
  non-native binding
- **THEN** validation and admission MUST fail closed before launch
- **AND** the request MUST NOT become a credential authority, dependency, acceptance source, or
  fallback route

#### Scenario: Native credential-home binding
- **WHEN** the pinned binding is `native_credential_home`
- **THEN** R3 MUST resolve exactly one approved exclusive opaque home for one existing logical
  runtime/agent before launch
- **AND** the daemon MUST supply only daemon-local isolated-home references required by the CLI
- **AND** OmniRouter/OmniRoute MUST NOT be probed, contacted, or used
- **AND** global HOME, raw path, account identity, and credential data MUST NOT enter product
  APIs, events, logs, or evidence

#### Scenario: Binding ambiguity or failure
- **WHEN** the binding is missing, ambiguous, stale, unauthorized, unhealthy, unsupported, or
  conflicts with an active assignment
- **THEN** admission MUST fail closed
- **AND** the system MUST NOT fallback, translate, rotate, or retry between bindings or homes

### Requirement: REQ-02 Owner-global versioned Runtime Standards

The platform MUST provide owner-global Runtime Standards with immutable versions, validation,
activation, audit and rollback.

#### Scenario: Activate a standard version
- **WHEN** an owner activates a validated version using the expected active version
- **THEN** activation MUST be compare-and-swap and auditable
- **AND** existing versions MUST remain immutable
- **AND** rollback MUST activate a prior version without rewriting history

### Requirement: REQ-03 Accountless reusable Runtime Sessions

Runtime Sessions MUST be owner-global, reusable and accountless.

#### Scenario: Reuse a session in another authorized workspace
- **WHEN** an owner enrolls a session into an authorized workspace
- **THEN** the platform MUST project it onto an existing runtime row, agent and daemon
- **AND** MUST NOT create or recreate an agent, runtime, daemon, container or credential home
- **AND** subscription/home attachment MUST remain a separate operation

### Requirement: REQ-04 Existing-row workspace binding

Each active workspace binding MUST identify exactly one existing agent, runtime row and Runtime
Session, and MUST preserve existing ORQ2-dev reservations.

#### Scenario: Enrollment references a missing or reserved row
- **WHEN** enrollment references a nonexistent row, another workspace, or protected ORQ2-dev binding
- **THEN** enrollment MUST fail before mutation
- **AND** MUST NOT move, share, borrow or steal any existing assignment

### Requirement: REQ-05 Opaque dynamic controlled-root registry

The daemon MUST discover arbitrary child homes under validated controlled roots without slot
allowlists or product-visible raw paths.

#### Scenario: New valid child appears beyond prior capacity
- **WHEN** a full reconciliation sees a valid arbitrary child not present in the prior generation
- **THEN** it MUST issue a new opaque home reference and publish a higher atomic generation
- **AND** no slot number grammar or fixed capacity allowlist may reject it
- **AND** APIs, events, logs and evidence MUST NOT expose the raw path

### Requirement: REQ-06 Reconciliation completeness and monotonicity

Filesystem notifications MUST be hints; startup, overflow, hint loss, watcher restart and
interval expiry MUST trigger a complete scan.

#### Scenario: Watcher overflows
- **WHEN** the filesystem watcher reports overflow
- **THEN** admission MUST use the last complete generation or fail closed
- **AND** a full scan MUST publish only after completion
- **AND** catalog generation MUST increase monotonically and never be reused

### Requirement: REQ-07 Catalog lifecycle safety

Catalog lifecycle MUST implement filesystem-identity deduplication, TTL, quarantine, active
reference protection, watermarks, tombstones and retention. Catalog lifecycle alone MUST NOT
authorize physical source-home deletion.

#### Scenario: Home disappears while referenced
- **WHEN** a home is absent from a full scan but an active task references its generation
- **THEN** it MUST become missing/draining and remain tombstoned
- **AND** it MUST NOT be reassigned, retired or have its opaque reference reused
- **AND** source data MUST NOT be copied, moved, deleted, truncated, sanitized, overwritten,
  chmodded, or have ownership changed
- **AND** cleanup MUST be limited to task-local non-source material after active-reference checks

#### Scenario: Whole-home deletion reaches policy eligibility
- **WHEN** a home is authoritatively `INACTIVE_PROVEN`, older than 24 hours, non-admissible,
  unambiguously mapped, and has zero assignments, reservations, and active references
- **THEN** it MAY become `POLICY_NOT_RETAINED` but MUST NOT yet be physically deleted
- **AND** physical whole-home deletion MUST require independently accepted ORQ-96
  database/admission/fencing/no-bypass gates and separate destructive cutover authorization
- **AND** age or mtime alone and force semantics MUST NOT bypass any guard

#### Scenario: Duplicate or invalid identity
- **WHEN** aliases, symlink escape, wrong ownership/mode, invalid layout or duplicate filesystem
  identity is detected
- **THEN** every ambiguous candidate MUST be quarantined metadata-only
- **AND** new admission MUST fail closed

### Requirement: REQ-08 Exclusive account-home assignment

A healthy native account-home MUST be exclusive to one active persistent agent/runtime binding,
and one binding MUST have at most one active home assignment.

#### Scenario: Concurrent assignment
- **WHEN** two bindings race to assign the same opaque home reference
- **THEN** exactly one compare-and-swap MUST succeed
- **AND** the loser MUST receive `exclusive_assignment_conflict`

#### Scenario: Attach a new subscription
- **WHEN** a legitimate newly enrolled home becomes healthy and unassigned
- **THEN** it MAY attach to the existing binding
- **AND** agent, runtime, session and daemon IDs MUST remain unchanged

### Requirement: REQ-09 Configuration completeness

Every versioned runtime configuration MUST fix `transport_binding` to `native_credential_home`, and cover provider and opaque
subscription reference, literal model,
reasoning, context/input/output/total token limits, concurrency, timeout/retry, CLI flags,
environment allowlist, skills, tools/MCP, filesystem/network permissions, eligibility, health
and fallback.

#### Scenario: Unknown or missing field contract
- **WHEN** a configuration contains an unregistered field or a required group lacks a schema
- **THEN** validation MUST fail with `invalid_configuration`
- **AND** the field MUST NOT be ignored or passed to a runtime

### Requirement: REQ-10 Deterministic precedence

Effective configuration MUST resolve each field using
`platform > standard > runtime > explicitly delegable task`.

#### Scenario: Lower layer widens policy
- **WHEN** runtime or task configuration attempts to widen a platform/standard limit, permission,
eligibility, fallback or exposure boundary
- **THEN** resolution MUST reject the version or task before launch
- **AND** omission MUST inherit rather than erase higher policy

### Requirement: REQ-11 Capability, delegability and apply class

Every configurable leaf MUST declare type, capability predicate, redaction, delegability and
`hot` or `restart` apply class.

#### Scenario: Unsupported reasoning or model
- **WHEN** a model/reasoning/limit combination is absent from the pinned capability catalog
- **THEN** activation MUST fail with a field-addressed `capability_unsupported`
- **AND** validation MUST NOT make an inference request

#### Scenario: Non-delegable task override
- **WHEN** a task overrides a non-delegable field
- **THEN** claim or launch MUST fail before process creation

#### Scenario: Restart-class activation
- **WHEN** a restart-required version is activated
- **THEN** it MUST remain pending until the existing runtime process restarts and acknowledges it
- **AND** running tasks MUST retain their pinned prior version

### Requirement: REQ-12 Activation, audit, redaction, drift and rollback

Configuration activation and rollback MUST be compare-and-swap, redacted and auditable; daemon
application MUST be acknowledged and continuously compared for drift.

#### Scenario: Digest drift
- **WHEN** daemon-applied version/digest differs from desired state
- **THEN** `runtime_configuration.drift_detected` MUST be emitted
- **AND** new claims on that binding MUST fail closed until reconciled
- **AND** audit/events MUST contain no secret, raw path, argv/env value or provider response

### Requirement: REQ-13 Immutable task snapshots

Atomic claim MUST pin session/runtime/agent/workspace IDs, standard and runtime configuration
version IDs, effective digest, binding ID/generation, capability digest, and for native tasks the
opaque home reference/catalog generation.

#### Scenario: Configuration or assignment changes after claim
- **WHEN** active configuration or home assignment changes after a task is claimed
- **THEN** that attempt and its subagents MUST retain the original snapshot
- **AND** reclaim/reuse MUST NOT perform a live lookup that rewrites history

### Requirement: REQ-14 Subagent inheritance

Subagents MUST inherit their parent's task snapshot and MAY receive only explicitly delegable,
bounded task overrides.

#### Scenario: Spawn subagent
- **WHEN** a task spawns a subagent
- **THEN** no persistent runtime/session/account-home row MUST be created
- **AND** it MUST share the parent binding/home and count against configured concurrency

### Requirement: REQ-15 Concurrency independent of accounts

Session, task and subagent concurrency MUST be explicit configuration and MUST NOT derive from
the count of catalog homes.

#### Scenario: Concurrency exceeds home inventory
- **WHEN** configured native concurrency cannot be served without sharing an exclusive home
- **THEN** excess work MUST queue or fail with `capacity_exhausted`
- **AND** no home or ORQ2-dev assignment may be shared

### Requirement: REQ-16 Controlled fallback

Fallback MUST be ordered, bounded, capability-validated, health-gated and frozen in the task's
effective configuration. Every retry route MUST preserve `native_credential_home` and the pinned
exclusive `home_ref`.

#### Scenario: Preferred route is unhealthy
- **WHEN** the preferred route is unhealthy
- **THEN** only a prevalidated eligible provider/model attempt on the same pinned native home MAY run
- **AND** fallback, translation, rotation, or retry through OmniRouter/OmniRoute, another binding,
  or another native home MUST be forbidden
- **AND** global HOME, cross-workspace, stale-generation, and ORQ2-dev fallback MUST be forbidden

### Requirement: REQ-17 Frozen migration identities

Implementation MUST reserve migrations 130-134 as named in the design, subject to an immediate
registrar recheck.

#### Scenario: Registrar detects collision
- **WHEN** any identity 130-134 is occupied before implementation
- **THEN** schema work MUST stop
- **AND** K1 MUST amend the canonical contract before any replacement name is used

### Requirement: REQ-18 Frozen REST contracts

Implementations MUST use the methods, routes, body fields, pagination, idempotency, response
status and error envelope frozen in the design.

#### Scenario: Administrative mutation
- **WHEN** a Runtime Standard, session, enrollment, binding, assignment or configuration is mutated
- **THEN** an authenticated authorized human owner/admin and `Idempotency-Key` MUST be required
- **AND** expected version/generation MUST be checked where specified
- **AND** task actors and cross-workspace actors MUST receive `forbidden`

#### Scenario: Error response
- **WHEN** a request fails
- **THEN** it MUST return `{error:{code,message,field?,request_id,retryable}}`
- **AND** message/field data MUST NOT reveal a path, account identity, credential, env value or
  provider response body

### Requirement: REQ-19 Frozen event contracts

Runtime-manager events MUST use the frozen envelope and type names, at-least-once delivery and
per-resource monotonic generations.

#### Scenario: Duplicate or gap
- **WHEN** a consumer receives a duplicate event
- **THEN** it MUST deduplicate by `event_id`
- **WHEN** it observes a generation gap
- **THEN** it MUST GET/reconcile complete state rather than infer missing changes

### Requirement: REQ-20 Authorization and path secrecy

Owner-global administration MUST be owner-only; workspace administrators MUST be confined to
their workspace; daemons MUST report only authorized daemon/binding state.

#### Scenario: Read catalog or evidence
- **WHEN** an authorized actor reads catalog, event, audit or task evidence
- **THEN** only opaque refs, states, versions, generations, digests, reason codes and counters MAY
  be returned
- **AND** raw paths, filesystem identity, account identity, credential names/data, prompts and
  provider response bodies MUST be absent

### Requirement: REQ-21 Health, revoke and retention

Eligibility MUST require fresh health and catalog data. Revocation MUST drain by default; hard
revoke requires explicit policy and audit. Audit/task snapshots and catalog tombstones MUST be
retained through all active references and evidence windows.

#### Scenario: Stale health or active reference
- **WHEN** health TTL expires or retirement is requested for a referenced home
- **THEN** new claims MUST fail with `health_stale` or `active_reference`
- **AND** every active, draining, referenced, reserved, admissible, unproven, or quarantined source
  credential home MUST remain untouched regardless of TTL or retention age
- **AND** only task-local non-source material MAY be cleaned after active-reference checks

#### Scenario: Inactivity or authority is uncertain
- **WHEN** inactivity, mapping, database state, fencing, process references, or filesystem boundary
  cannot be proven authoritatively
- **THEN** the home MUST remain `QUARANTINED_FAIL_CLOSED`, preserved, and non-admissible
- **AND** age, mtime, local registry data, alert state, or force semantics MUST NOT authorize deletion

### Requirement: REQ-22 Existing ORQ2 topology and release boundary

The solution MUST reuse the existing ORQ2 daemon and existing rows, preserve all ORQ2-dev
bindings, and remain fail closed until approved implementation and assurance gates pass.

#### Scenario: Documentation completion
- **WHEN** SPE-5 documentation validates
- **THEN** it MUST NOT imply executable implementation, production authorization or SharePoint
  authorization
- **AND** K3 review, integrated gates, Council acceptance and separate rollout authorization MUST
  remain mandatory

### Requirement: REQ-23 Repository/OpenSpec synchronization gate

Before any future handoff or deployment, every affected OpenSpec MUST pass strict validation,
the cross-authority residual scan MUST pass, and the Git index/worktree MUST have zero staged,
modified, or untracked owned files. The exact local commit MUST be published to its configured
non-main remote branch; local HEAD and upstream SHA MUST match and ahead/behind MUST be `0/0`.
Deployment evidence MUST pin that exact SHA. This requirement documents a future gate and does
not itself authorize publishing or deployment.

#### Scenario: Pending file or repository divergence
- **WHEN** an owned file is staged, modified, or untracked, an affected OpenSpec or authority
  scan fails, the exact commit is unpublished, upstream is absent or points to main, local and
  upstream SHAs differ, or ahead/behind is not `0/0`
- **THEN** handoff and deployment MUST fail closed
- **AND** no deployment evidence may claim a different or unresolved SHA
### Requirement: REQ-24 Resolucao por provider

Accepted REQ-05 MUST have precedence: discovery is dynamic and opaque, with no static slot grammar, number, or allowlist. The later `<slot>` notation denotes only the already-selected opaque physical home for the active binding.

O daemon MUST resolver a raiz de credencial especifica de cada provider obrigatorio.

#### Scenario: Preparar uma task obrigatoria

- **WHEN** o daemon prepara ambiente de execucao para uma task
- **THEN** `CredentialAccountHome` MUST ser resolvido pela raiz especifica do vendor dentro do home opaco selecionado dinamicamente, conforme: antigravity/agy `<slot>/home`, kiro `<slot>/xdg-data` e
  codex `<slot>/codex`
- **AND** MUST NOT usar um caminho unico para todos os providers

### Requirement: REQ-25 Validacao fail-closed do caminho

This requirement MUST specialize accepted REQ-01, REQ-05, REQ-07 and REQ-20 and MUST NOT create a second path/discovery authority.

O daemon MUST rejeitar todo AccountHome ausente, invalido ou fora do root autorizado.

#### Scenario: Validar AccountHome

- **WHEN** um caminho de `AccountHome` e resolvido
- **THEN** MUST ser absoluto, MUST estar sob o root permitido, MUST passar por `EvalSymlinks` e
  `Stat`
- **AND** caminho ou artefato nativo ausente MUST retornar erro e bloquear a task
- **AND** MUST NOT converter erro em HOME global

### Requirement: REQ-26 Codex honra AccountHome

This requirement MUST specialize the accepted native-only transport binding and MUST NOT create a second transport authority.

O daemon MUST usar AccountHome nativo pelo Codex e MUST rejeitar transporte credentialless.

#### Scenario: Executar por gateway

- **WHEN** existe plano gateway
- **THEN** a configuracao e a admissao MUST falhar antes do launch
- **AND** o gateway MUST NOT virar fallback, autoridade de credencial ou fonte de aceite

#### Scenario: Executar nativamente

- **WHEN** a execucao e nativa
- **THEN** `CredentiallessGateway` MUST ser falso e `AccountHome` MUST ser valido

### Requirement: REQ-27 Cobertura de Prepare e Reuse

O daemon MUST aplicar as mesmas garantias de isolamento aos caminhos Prepare e Reuse.

#### Scenario: Preparar ou reutilizar ambiente

- **WHEN** o patch e aplicado
- **THEN** MUST cobrir o caminho Prepare e o caminho Reuse, que duplicam as mesmas condicoes

### Requirement: REQ-28 Identidade estavel e atribuicao deterministica

This MUST be treated as validated source-v2 behavior for a future separately authorized rollout. It MUST NOT describe installed legacy registry v1 or authorize deployment, restart, registry conversion, or home mutation.

A alocacao MUST usar identidade explicita e estavel composta por UUID canonico nao-zero do
agente mais fingerprint SHA-256 da subscription do provider em 64 caracteres hex minusculos.
Os portadores canonicos sao `AGENT_CRED_ISOLATION_AGENT_ID` e
`AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT`. Ausencia ou invalidade de qualquer componente
MUST falhar antes de qualquer alocacao de raiz.

#### Scenario: Repetir tasks do mesmo agente

- **WHEN** um agente executa tasks repetidas sob a mesma subscription
- **THEN** a selecao MUST ser deterministica a partir de
  `AGENT_CRED_ISOLATION_AGENT_ID + AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT`
- **AND** MUST persistir a escolha
- **AND** MUST NOT usar round-robin

#### Scenario: Identidade ausente ou invalida

- **WHEN** o UUID do agente e zero ou ausente, ou o fingerprint nao e exatamente 64 hex
  minusculos
- **THEN** a alocacao MUST falhar fechada antes de criar ou reservar qualquer raiz
- **AND** MUST NOT derivar identidade de Herdr, pane, TTY, PID, processo ou UUID aleatorio
- **AND** MUST NOT incrementar contador de slot

#### Scenario: Slot persistido inelegivel

- **WHEN** um slot persistido deixa de ser elegivel
- **THEN** a task MUST falhar em vez de remapear silenciosamente
- **AND** o remapeamento MUST exigir acao operacional explicita

### Requirement: REQ-29 Runtimes obrigatorios

Accepted REQ-05 dynamic catalog authority MUST have precedence. No static slot allowlist or slot-number grammar MAY be reintroduced.

O T2 MUST manter antigravity/agy, Codex e Kiro operacionais em todo cenario suportado.

#### Scenario: Avaliar o escopo T2

- **WHEN** qualquer cenario e avaliado
- **THEN** antigravity/agy, codex e kiro MUST estar operacionais
- **AND** cline, opencode e demais providers estao fora de escopo

#### Scenario: Descobrir modelos Antigravity

- **WHEN** a UI solicita o catalogo do runtime antigravity
- **THEN** `agy models` MUST executar com HOME de uma entrada saudavel e elegivel do catalogo dinamico opaco
- **AND** o HOME global do daemon MUST NOT ser usado
- **AND** discovery que retorna erro MAY tentar a proxima entrada elegivel apenas para catalog discovery; task binding MUST NOT remapear ou fazer fallback

#### Scenario: Preparar task-home Antigravity

- **WHEN** o daemon prepara uma task com AccountHome AGY elegivel
- **THEN** MUST copiar somente `.gemini/antigravity-cli/antigravity-oauth-token`
- **AND** o token de destino MUST ser arquivo fisico regular com modo `0600`
- **AND** logs, symlinks, caches, bancos e demais artefatos irmaos MUST NOT ser copiados
- **AND** token ausente, symlink ou nao regular MUST falhar explicitamente

### Requirement: REQ-30 Topologia T2

This MUST remain a future cutover condition gated by fresh explicit owner approval. It MUST NOT record a completed replacement, authorize stopping ORQ1, or perform deployment or restart.

Quando o cutover for separadamente autorizado, o executor credential-isolated MUST rodar no ORQ2 e substituir o executor do ORQ1.

#### Scenario: Executar com conta isolada

- **WHEN** o daemon executa uma task com conta isolada
- **THEN** ele MUST estar no ORQ2 e consumir somente slots locais sob o root 0700
- **AND** o daemon ORQ1 MUST deixar de ser elegivel antes da primeira task T2

### Requirement: REQ-31 Durabilidade

O tunel e o daemon T2 MUST reiniciar automaticamente em ordem apos reboot do ORQ2.

#### Scenario: Reiniciar o host ORQ2

- **WHEN** o T2 entra em operacao
- **THEN** binario e tunel MUST ser geridos fora de `/tmp`
- **AND** reinicio do host MUST restaurar tunel antes do daemon

### Requirement: REQ-32 Configuracao de reasoning

This requirement MUST specialize accepted configuration/capability authority and MUST NOT create a competing precedence model.

A UI MUST preservar o formato de reasoning anunciado pelo runtime e MUST persistir a escolha
explicita do owner sem nivel chumbado.

#### Scenario: Modelo com niveis estruturados

- **WHEN** o modelo selecionado anuncia `thinking.supported_levels`
- **THEN** criar e duplicar agente MUST mostrar um seletor separado com os tokens anunciados
- **AND** a escolha MUST ser enviada e persistida em `thinking_level`
- **AND** a ausencia de escolha MUST manter o comportamento nativo do CLI

#### Scenario: Tier embutido no ID AGY

- **WHEN** o modelo selecionado nao anuncia `thinking.supported_levels` e seu tier ja faz
  parte do ID
- **THEN** a UI MUST manter o ID literal como escolha de modelo
- **AND** MUST NOT mostrar um segundo seletor de reasoning

#### Scenario: Trocar para catalogo incompativel

- **WHEN** runtime ou modelo muda e o `thinking_level` atual nao existe no novo catalogo
- **THEN** a UI MUST limpar o override obsoleto antes de criar o agente

### Requirement: REQ-33 Snapshot imutavel da conta produtora

This requirement MUST specialize accepted REQ-13 immutable task snapshots; accepted snapshot/reclaim authority MUST remain controlling.

O backend MUST congelar a conta aprovada que produz cada tentativa no claim atomico e MUST
copiar somente esse snapshot para `task_usage`. O daemon MUST NOT enviar `account_id`.

#### Scenario: Assignment muda depois do claim

- **WHEN** a task e reivindicada com conta A e o agente e depois reatribuido para conta B
- **THEN** a task e todos os reports dessa tentativa MUST permanecer atribuidos a A
- **AND** um report posterior MUST NOT reescrever a historia usando a assignment corrente

#### Scenario: Reclaim de linha com snapshot

- **WHEN** uma task dispatched ja possui `credential_account_id`
- **THEN** reclaim MUST preservar exatamente esse snapshot

#### Scenario: Reclaim de linha legada sem snapshot

- **WHEN** uma task de provider coberto ja esta dispatched com snapshot NULL durante o cutover
- **THEN** ela MUST continuar visivel ao claim/reclaim gate
- **AND** MUST ser cancelada fail-closed pelo contrato ORQ-21
- **AND** MUST NOT receber uma conta por lookup vivo ou backfill retroativo

#### Scenario: Uso nao atribuivel

- **WHEN** nenhuma conta aprovada foi congelada
- **THEN** `task_usage.account_id` MUST permanecer NULL
- **AND** relatorios MUST expor o bucket NULL em vez de omiti-lo dos totais

### Requirement: REQ-34 Cardinalidade fisica das raizes de credencial

Accepted REQ-05/07/21 lifecycle authority MUST have precedence. "Zero historicos" is a gated
convergence objective after lawful cleanup, not an instantaneous invariant or count-only deletion
authority. One admissible physical home per active stable binding remains exact. Non-admissible
homes MAY exist only while protected by retention hold, pending policy/gates, or fail-closed
quarantine.

O numero de homes admissiveis MUST ser exatamente igual ao numero de bindings ativos estaveis.
Um home `INACTIVE_PROVEN` MUST permanecer retido ate completar mais de 24 horas. Idade e necessaria
para `POLICY_NOT_RETAINED`, mas nunca e autoridade suficiente de remocao.

#### Scenario: Reconciliacao do conjunto ativo

- **WHEN** a reconciliacao executa
- **THEN** MUST existir exatamente um home admissivel por binding ativo estavel
- **AND** homes adicionais MUST estar nao-admissiveis em `RETENTION_HOLD`,
  `POLICY_NOT_RETAINED` pendente, ou `QUARANTINED_FAIL_CLOSED`
- **AND** diretorios historicos MUST convergir para zero somente depois que cada home passar por
  todos os gates de cleanup e pela autorizacao destrutiva separada
- **AND** a reconciliacao de producao MUST exigir auditoria de todos os processos via `/proc`
  com privilegio de root, falhando fechada e sem remover nada quando indisponivel
- **AND** homes referenciados por processo vivo MUST ser preservados
- **AND** idade maior que 24 horas MUST ser um criterio necessario somente depois de
  `INACTIVE_PROVEN`, e MUST NOT ser criterio suficiente de remocao

#### Scenario: Slot legado unico preexistente

This scenario is historical conditional chronology, not a statement that one global slot is the current invariant.

- **WHEN** historical migration starts from exactly one legacy active slot and that condition is freshly proven
- **THEN** ele MUST ser adotado no lugar, sem mover, sobrescrever ou recriar
- **AND** seu conteudo de credencial MUST NOT ser lido, copiado ou impresso

### Requirement: REQ-35 Tombstone de metadados e nao-reuso

This requirement MUST specialize accepted REQ-07 and REQ-21; their catalog lifecycle, retention,
protected-home preservation, and gated-cleanup clauses MUST remain controlling.

A camada de metadados MUST preservar tombstones para garantir nao-reuso, independentemente da
reconciliacao fisica.

#### Scenario: Referencia liberada ou revalidacao falha

- **WHEN** a ultima referencia cai para zero ou uma revalidacao falha
- **THEN** um tombstone completo MUST ser persistido imediatamente, com rollback e retry em
  falha de persistencia
- **AND** o nao-reuso MUST sobreviver a restart, restaurado por geracao duravel com fence de
  admissao no startup

#### Scenario: Prazo de retencao expira

- **WHEN** um prazo de retencao de metadados expira
- **THEN** o prazo MUST ser tratado como evidencia apenas
- **AND** o tombstone MUST NOT ser apagado por expiracao de prazo
- **AND** a camada de catalogo MUST NOT executar copia, remocao ou historico de pasta fisica
- **AND** eventual remocao fisica por executor autorizado MUST preservar o tombstone duravel e o
  nao-reuso

#### Scenario: Correlacao entre metadados e pasta fisica

- **WHEN** um `home_ref` UUID canonico e emitido
- **THEN** ele MUST ser derivado deterministicamente de
  `AGENT_CRED_ISOLATION_AGENT_ID + AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT`, exatamente
  os mesmos insumos usados pelo alocador fisico
- **AND** `name_ref` MUST ser um valor canonico `name_<43>` distinto do `home_ref`
- **AND** a camada de catalogo MUST NOT criar diretorio fisico
