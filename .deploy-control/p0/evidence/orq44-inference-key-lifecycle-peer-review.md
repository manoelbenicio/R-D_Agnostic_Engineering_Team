# ORQ-44 — Peer review independente do desenho de ciclo de vida da inference key (READ-ONLY)

- Card: **ORQ-44** · UUID `abd12d6a-16a5-439b-bc52-74ec4bb6b231` · number 44
- Documento revisado: `.deploy-control/p0/evidence/orq44-omniroute-inference-key-lifecycle.md`
  (autor Opus48#B, 17756 bytes, mtime 15:20)
- Revisor: Codex56#A (`w7:p3`) · UTC 2026-07-27T15:38Z
- **Skill carregada integralmente antes da revisão:** `.agents/skills/aws-secrets-manager/SKILL.md`
  (regras R1 `get-secret-value`/`batch-get-secret-value` proibidos; R2 SMA `localhost:2773` proibida;
  R3 `{{resolve:secretsmanager:...}}` resolvido por `asm-exec` no processo filho; aviso explícito
  "best-effort defense, not a security boundary")
- Modo: **READ-ONLY**. Nenhum segredo lido; **nenhum `stat` do arquivo-alvo** do segredo; nenhuma
  chamada a provider; zero mutação de AWS, arquivo, serviço, código ou board.

## VEREDITO: **PASS com 2 emendas obrigatórias e 4 perguntas de owner ainda abertas**

O desenho é tecnicamente correto onde eu consegui verificar, e é o primeiro runbook desta série que
**mede o consumidor antes de propor a operação**. As duas emendas não invalidam a sequência; corrigem
um gate que falharia silenciosamente e uma premissa de plataforma que o próprio autor declarou não ter
verificado.

## 1. Verificação item por item do meu escopo de review

| item | afirmação do doc | minha verificação independente | resultado |
|---|---|---|---|
| leitura **por chamada, sem cache** | `credential_file_source.go:26-39` | comentário verbatim confirmado: *"reads a restricted secret file at call time … The value is never cached, logged, or returned outside the callback scope"*; `FileCredentialSource struct{}` sem campo de cache; nenhum `sync.Once`/TTL no arquivo | **CONFIRMADO** |
| `O_NOFOLLOW` | ":32-34", "platform helpers" | `credential_file_source_unix.go:13-14`: `os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)`; `:24` `isELOOP`; build tag `//go:build !windows` | **CONFIRMADO (Unix)** |
| `Fstat` no mesmo descritor | ":61-62" | fluxo medido: `openCredentialFile` → `f.Stat()` → checagens → leitura do mesmo `f` | **CONFIRMADO** |
| modo sem bits de grupo/mundo | "`&0o077 != 0` rejeita" | `credentialAllowedModeMask = os.FileMode(0o077)` com comentário *"Accepts 0600 (owner rw only) or stricter (0400)"*; teste `info.Mode().Perm()&credentialAllowedModeMask != 0` | **CONFIRMADO** |
| dono = UID do processo | `checkCredentialOwner` | presente em `credential_file_source_unix.go:34` | **CONFIRMADO (Unix)** |
| limites 4096 / mínimo 8 / UTF-8 / sem espaço / sem controle | ":17-20, :77-106" | `credentialFileMaxBytes = 4096`, `credentialMinTrimmedLen = 8`, `utf8.ValidString`, `strings.ContainsAny(value, " \t\n\r")`, `unicode.IsControl` | **CONFIRMADO** |
| erros content-free | ":111-119" | razões simbólicas (`secret_file_content_not_utf8`, etc.), sem valor nem erro de OS | **CONFIRMADO** |
| `SecretFileRef` exige absoluto | `config.go:84-92` | verbatim: `"gateway secret file reference is required"` e `"… must be absolute"` + `filepath.Clean` | **CONFIRMADO** |
| gateway-required exige secret + strict fail-closed | `config.go:163-181` | verbatim: `if c.Required && c.SecretFile.Path == ""` → erro; `if c.Required && (!c.Readiness.FailClosed || c.Readiness.Name != ReadinessStrict)` → erro | **CONFIRMADO** |
| gate de fila com 4 estados de 7 | `109_…up.sql:13-15` | `CHECK (status IN ('queued','dispatched','running','waiting_local_directory','completed','failed','cancelled'))` — 7 estados, 3 terminais, os 4 ativos citados corretamente | **CONFIRMADO** |
| contraste com `JWT_SECRET` preso em `sync.Once` | `jwt.go:31` | `jwtSecretOnce sync.Once` + `JWTSecret()` com `Do(...)` lendo `os.Getenv("JWT_SECRET")` | **CONFIRMADO** |
| swap atômico por `rename(2)` no mesmo filesystem | §4 | raciocínio correto e consistente com leitura por chamada: `>` truncaria e produziria `secret_file_content_too_short`; `mv` no mesmo diretório é `rename` | **CONFIRMADO por análise** (não executei) |
| sem restart, exceto se o **caminho** mudar | §7 | coerente: o valor é lido por chamada; o caminho vem do ambiente da unit, logo mudar caminho exige novo processo | **CONFIRMADO por análise** |
| validação só por metadado | §4/§6.2 | `stat -c` e `grep -q` (exit code) não imprimem valor; **eu não executei nenhum** | **CONFIRMADO como forma** |
| sobreposição no provider | §3 declarada **não verificável** | correto: `SecretFileRef` é **um** caminho, o daemon envia **uma** credencial por requisição; overlap é capacidade do OmniRoute | **CONFIRMADO como incerteza honesta** |
| janela de recuperação de 7 dias | §8 | `delete-secret` com recovery window e proibição de `--force-delete-without-recovery`: consistente com R1/R3 da skill (nenhuma leitura de valor envolvida) | **CONFIRMADO como política** |

## 2. Emenda 1 (obrigatória) — o gate V2 aponta para a porta errada em perfil nomeado

O doc usa, em §6.2: `curl … http://127.0.0.1:19514/health`. Medi:

```text
internal/daemon/config.go:58   DefaultHealthPort = 19514
cmd/multica/cmd_daemon.go:160-168  healthPortForProfile: perfil "" → DefaultHealthPort;
                                    perfil NOMEADO → base+1 + (soma de bytes do nome % 1000)
CONTRIBUTING.md:501            | Health port | 19514 | 19514 + 1 + (name_hash % 1000) |
probe read-only agora:         127.0.0.1:19514/health → 200   ·   19515 → 000
```

Hoje o daemon do ORQ2 (`--daemon-id orq2-credential-runtime-v1`) responde em **19514**, portanto o
comando do doc funciona **neste** host. Mas o gate está escrito com porta **fixa**: se a operação rodar
num daemon com **perfil nomeado**, `19514` responderá de outro daemon ou de nada, e o V2 dá
**falso verde** (200 de outro processo) ou **falso vermelho** (000) sem relação com a chave trocada.

**Emenda:** V2 deve derivar a porta do perfil em uso (ou aceitar a porta como parâmetro obrigatório do
runbook) e **provar que o processo que responde é o daemon-alvo** — por exemplo casando `daemon_id`
na resposta ou o PID que detém o socket, antes de aceitar o 200 como evidência.

## 3. Emenda 2 (obrigatória) — a premissa de plataforma é **fail-closed total no Windows**

O próprio doc declara em §12 que **não** verificou os helpers com build tag. Eu verifiquei:

```text
credential_file_source_windows.go:1   //go:build windows
:11  func openCredentialFile(_ string) (*os.File, error) { return nil, &credentialFileError{reason:"platform_unsupported"} }
:16  func checkCredentialOwner(_ os.FileInfo) error       { return &credentialFileError{reason:"platform_unsupported"} }
```

Ou seja: em Windows, `FileCredentialSource` **sempre** falha com `platform_unsupported`, logo um daemon
gateway-required **não funciona** naquele SO — independentemente da rotação. Isso não é defeito do
ORQ-44, mas muda duas coisas no runbook: (a) o escopo deve declarar **Unix-only**; (b) qualquer plano
de canário/rollout que inclua host Windows da frota (há vários no tailnet) está fora de escopo e deve
ser escalado, não absorvido.

**Emenda:** acrescentar em §11 (condições de parada) o item "host-alvo não é Unix" e, em §10, registrar
`platform_unsupported` como comportamento esperado no Windows.

## 4. Observações menores (não bloqueiam)

1. **`.prev` duplica o segredo em disco.** O doc reconhece como trade-off. Alternativa que evita a
   segunda cópia: no rollback, re-resolver `AWSPREVIOUS` via `asm-exec` para regravar o arquivo, em vez
   de manter cópia. Fica a escolha do owner; se `.prev` for mantido, a remoção ao fim da janela precisa
   ser **passo obrigatório com verificação**, não recomendação.
2. **L1 (canário) é chamada a provider** e pode consumir cota — o próprio doc pergunta isso em §3.3.
   Autorização 2 do doc já cobre, mas o gate deveria declarar explicitamente "custo esperado da
   requisição de canário" antes de executar.
3. **V4 "task de smoke não-inferência"**: sob o freeze atual, qualquer task exige card/orçamento
   próprio e o card não deve receber assignee. Recomendo reformular V4 como *probe de readiness*, sem
   task, para não colidir com o freeze de atribuição.
4. **V5 depende de leitura de log do daemon**; os `credentialFileError` são content-free por
   construção (confirmado), então anexar o log é seguro — vale registrar isso explicitamente na
   evidência para o revisor seguinte não hesitar.

## 5. Perguntas de owner/operador ainda **abertas** (bloqueantes para execução)

1. **Sobreposição:** o OmniRoute aceita duas chaves de inferência ativas simultaneamente? Por quanto
   tempo, e como se revoga a antiga? (Se **não**, L4 vira janela disruptiva curta e o gate de fila zero
   passa a ser obrigatório também **imediatamente antes** de L4.)
2. **Escopo da chave:** a chave é por conta ou global? Rotacioná-la afeta as 4 contas Antigravity, 4
   ClinePass e 2 OpenAI Codex, ou apenas o par de credencial do cliente?
3. **Introspecção sem cota:** existe endpoint que valide a chave sem consumir cota? Sem isso, L1 tem
   custo e precisa de orçamento declarado.
4. **Identidade do segredo:** o `secret-id` `prod/multica/omniroute-inference-key` é **proposta** do
   autor; o owner precisa confirmar o ID real (ou declarar que a fonte não é Secrets Manager, caso em
   que a sintaxe `{{resolve:...}}` da skill não se aplica e é preciso mecanismo handle-only equivalente).

## 6. Não-alegações

- Não li segredo, não chamei `GetSecretValue`/`BatchGetSecretValue`, não toquei a SMA, **não executei
  `asm-exec`**, não chamei o provider, não troquei arquivo, não reiniciei nada, não mutei AWS nem board.
- **Não fiz `stat` do arquivo-alvo do segredo**: as exigências que confirmei vêm do código do
  consumidor, não de medição do arquivo. Portanto **não sei** modo, dono, tamanho ou existência atual
  de `/etc/agent-brain/secrets/omniroute-inference-key`.
- O probe que fiz foi em `127.0.0.1:19514/health` e `:19515/health` — endpoint de saúde local, sem
  credencial e sem inferência; não é chamada ao gateway nem ao provider.
- Não validei empiricamente atomicidade de `rename(2)` neste filesystem nem o comportamento do provider.
- Não avaliei ORQ-37 (MCP) nem ORQ-43 (`mdt_`), corretamente excluídos pelo doc.
