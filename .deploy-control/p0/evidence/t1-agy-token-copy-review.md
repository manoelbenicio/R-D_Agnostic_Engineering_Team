# T1 — Review do patch AGY token-only (`prepareAntigravityHome`)

- Revisor: Kiro-Opus5 (leitura, análise e recomendação apenas — sem poder de decisão, AA-001 §0.0)
- Solicitante: Codex56#B (w7:p4); relatório também entregue ao Codex56-TL (w5:pC)
- Data: 2026-07-27T10:15Z
- Escopo autorizado: `server/internal/daemon/execenv/antigravity_home.go` e
  `antigravity_home_test.go`. Helpers compartilhados e call sites foram lidos apenas
  como contexto necessário.
- Ações executadas: somente leitura. Nenhuma edição, deploy, restart ou rerun.
- Status de rollout (informado pelo solicitante): **BLOQUEADO** até correção de H1/H2 e
  nova execução dos gates.

## Veredito

O patch é direcionalmente correto e resolve a falha reportada. `prepareAntigravityHome`
passou a mirar o arquivo `.gemini/antigravity-cli/antigravity-oauth-token`
(`antigravity_home.go:11-12,42-43`), então o symlink `cli.log` não pode mais acionar o
erro `unsupported credential path type` de `copyCredentialDir`. O teste planta exatamente
esse symlink na origem e comprova que o preparo conclui sem erro
(`antigravity_home_test.go:20-27,45-47`).

Dois itens HIGH devem ser corrigidos antes do rollout; quatro itens MED precisam de
decisão do owner ou do TL.

## HIGH

### H1 — Janela de permissão frouxa durante a cópia

`copyCredentialFile` cria o destino com o modo **da origem**:

```go
// kiro_home.go:95
out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, srcInfo.Mode().Perm())
```

O próprio teste faz `os.Chmod(sourceTokenA, 0o644)` (`antigravity_home_test.go:16-18`),
provando que a origem pode ser 0644. Nesse caso o token é criado 0644 e só é restringido
pelo `os.Chmod(dst, 0o600)` posterior (`antigravity_home.go:52-54`). Consequências:

1. os bytes do token ficam legíveis por grupo/outros durante todo o `io.Copy`;
2. se `io.Copy` ou `Close` falhar, a função retorna **antes** do chmod, deixando um token
   parcial 0644 no env root até o próximo preparo removê-lo.

Recomendação: criar arquivos de credencial com modo explícito 0600 no helper (por exemplo
`syncCredentialFileMode(src, dst, 0o600)`), mantendo o chmod posterior como redundância.
Observação de escopo: `prepareKiroHome` não tem chmod algum no destino, logo
`kiro-cli/data.sqlite3` herda permanentemente o modo da origem. É pré-existente e fora do
T1, mas o ponto de correção é o mesmo helper.

### H2 — Modos dos diretórios intermediários não são garantidos

`prepareAntigravityHome` faz chmod 0700 apenas em `home` (`antigravity_home.go:34-40`).
Os diretórios `.gemini` e `.gemini/antigravity-cli` são criados por
`os.MkdirAll(filepath.Dir(dst), 0o700)` dentro de `copyCredentialFile`
(`kiro_home.go:85`). `MkdirAll` aplica o modo somente aos diretórios que ele cria e está
sujeito a umask; diretórios pré-existentes com modo frouxo permanecem frouxos.
`prepareKiroHome` faz chmod explícito em `home` e em `home/kiro-cli`
(`kiro_home.go:43-48`).

Recomendação: paridade — `MkdirAll` mais `Chmod(0700)` explícito para `.gemini` e
`.gemini/antigravity-cli`.

## MED

### M1 — TOCTOU entre o guard e a cópia

O caminho de origem é resolvido três vezes: `os.Lstat(src)` no guard
(`antigravity_home.go:44-48`), `os.Stat(src)` em `syncCredentialFile`
(`kiro_home.go:63`, que **segue** symlink) e `os.Open(src)` em `copyCredentialFile`
(`kiro_home.go:89`). Se o account home não for de propriedade exclusiva do daemon com
0700, a origem pode ser trocada por um symlink entre as verificações e a cópia o seguirá,
trazendo um arquivo arbitrário para o HOME da task. Probabilidade baixa, impacto alto.

Recomendação: abrir uma única vez com `O_NOFOLLOW` e validar via `fstat` do handle, ou
documentar formalmente a fronteira de confiança do account home.

### M2 — Sem write-back do token renovado (risco funcional principal)

A cópia é unidirecional e ocorre no preparo. O caminho de reuso repete o preparo
(`execenv.go:527-535`), sobrescrevendo o que o CLI tiver escrito no HOME da task. O teste
afirma ambos os lados: a escrita feita dentro do HOME não chega à origem, e o reuso
substitui o HOME pelo conteúdo mais novo da origem
(`antigravity_home_test.go:49-60`). Consequências:

1. qualquer refresh que o CLI AGY faça dentro do HOME da task é descartado, então o token
   do account home nunca é renovado pelo tráfego das tasks e precisa ser renovado fora de
   banda antes de expirar;
2. se o Antigravity rotacionar o refresh token no uso (uso único), a primeira task que
   renovar invalida a cópia do account home e todas as tasks seguintes partem de um token
   morto; tasks concorrentes na mesma conta amplificam o efeito.

Não foi possível confirmar o comportamento de rotação do Antigravity a partir do
repositório. É necessária decisão escrita sobre política de write-back, ou declaração
escrita de que o token é de longa duração e renovado fora de banda.

### M3 — Reuso falha aberto enquanto o preparo falha fechado

`Prepare` retorna erro e impede o lançamento (`execenv.go:313-317`). O caminho de reuso
apenas registra `logger.Warn("execenv: refresh antigravity-home failed", ...)`
(`execenv.go:532`) e mantém o `env.AntigravityHome` anterior, de modo que um token
ausente, symlink ou não regular **não** bloqueia o lançamento: a task roda sobre a cópia
antiga. Padrão pré-existente compartilhado com kiro e cline, não introduzido por este
patch, mas enfraquece o novo guard exatamente no caso de reuso.

Recomendação: decidir se falha de guard de credencial deve ser terminal também no reuso.

### M4 — Alcance da evidência token-only

A prova E2E registrada é `agy models` com exit 0 em mount namespace efêmero com apenas o
token montado. Isso valida a resolução de autenticação, não um turno completo de prompt
(settings, configuração MCP, installation id, cache passaram a estar ausentes). O CLI
recria no HOME isolado o que precisar, que é a intenção do patch; o risco residual é um
artefato exigido só em turno real.

Recomendação: um E2E `agy` de prompt não mutante, autorizado pelo owner, antes do rollout.

## Verificado como correto

- Symlink plantado no **destino** não é seguido: `syncCredentialFile` faz `os.Lstat(dst)`
  e `os.Remove(dst)`, removendo o link em si, e a recriação usa `O_EXCL`
  (`kiro_home.go:71-76,95`).
- Nenhum conteúdo de token é registrado em log: `logCredentialFileState` emite apenas
  path, size, mtime e tipo (`kiro_home.go:112-131`).
- Isolamento por conta A/B confirmado por teste, incluindo modo final 0600 mesmo com
  origem 0644 (`antigravity_home_test.go:40-48`).
- Caminhos de rejeição não deixam destino parcial (`antigravity_home_test.go:120-123`).
- Apenas o token é copiado: `state.db`, `cache/` e `cli.log` ficam ausentes no HOME da
  task (`antigravity_home_test.go:45-47`).

## Lacunas de cobertura

1. modo de `.gemini` e `.gemini/antigravity-cli` no HOME da task não é verificado;
2. destino pré-existente como symlink ou como diretório não é testado (sobrescrita de
   arquivo regular obsoleto é coberta indiretamente pelo reuso);
3. `home == ""` com `AccountHome` não vazio não é testado;
4. a tabela de rejeição cobre symlink e diretório, mas não FIFO, socket ou device;
5. não há teste de que uma falha no meio da cópia não deixa nada legível por outros;
6. não há teste de independência de umask.

## Limite de verificação

Não existe toolchain Go neste host: `go` não está no PATH e não foram encontrados
`/usr/local/go`, `/opt/go` nem qualquer `*/bin/go`. Portanto **não** foi possível
re-executar `go test ./... -count=1`, `go vet ./...` ou `go build ./...` de forma
independente. Instalar runtime está na classe STOP-AND-WAIT (AA-001 §0.1) e não foi feito.
A afirmação de gates verdes em `2026-07-27T10:09:31Z` permanece evidência do escritor
único, não verificada por este review.

## Itens que exigem decisão escrita do owner

| # | Item | Classe AA-001 | Estado |
|---|---|---|---|
| T1-a | Corrigir H1 e H2 no helper de credencial (refactor de código) | §0.1 refactor | pendente |
| T1-b | Rerodar gates após correção (requer toolchain Go no host) | §0.1 instalação | pendente |
| T1-c | Política de write-back do token AGY (M2) | §0.1 credenciais | pendente |
| T1-d | Falha de guard terminal também no reuso (M3) | §0.1 refactor | pendente |
| T1-e | E2E `agy` de prompt não mutante antes do rollout (M4) | §0.1 blast radius | pendente |

---

# POS-FIX RE-REVIEW T1 — 2026-07-27T10:23Z (patch congelado, somente leitura)

Veredito: **PASS** nos quatro itens verificados (H1, H2, M1, M3). Nenhuma edição, deploy,
restart ou rerun. Ressalva de rollout: os gates não foram re-executados por mim (sem
toolchain Go neste host), logo o PASS é de revisão de código, não de execução de gates.

## H1 — 0600 no primeiro byte + limpeza de parcial: PASS

- `kiro_home.go:108` — `os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)`; o modo
  da origem não é mais herdado.
- `kiro_home.go:103-107` — comentário fixa a invariante (privado desde o primeiro byte).
- `kiro_home.go:111-118` — flag `success` com `defer` que faz `out.Close()` e
  `os.Remove(dst)` em qualquer saída de erro, então falha de `io.Copy`, `Close` ou
  `Chtimes` não deixa token parcial. `success = true` só em `kiro_home.go:128`, após
  `Chtimes` (`kiro_home.go:125`).
- `antigravity_home.go:56-61` — chmod 0600 mantido como pós-condição, agora documentado
  como redundância.
- Teste: `antigravity_home_test.go:16-18` mantém origem 0644 e
  `antigravity_home_test.go:54-55` exige 0600 no destino.

## H2 — diretórios 0700: PASS

- `antigravity_home.go:34-45` — laço com `MkdirAll(0700)` **e** `Chmod(0700)` explícito
  para `home`, `home/.gemini` e `home/.gemini/antigravity-cli`; corrige diretório
  pré-existente frouxo e elimina a dependência de umask.
- Teste: `antigravity_home_test.go:28-39` pré-cria os três diretórios com 0755 e
  `antigravity_home_test.go:56-58` exige 0700 nos três após o preparo. Este é o teste que
  faltava.

## M1 — O_NOFOLLOW + fstat: PASS

- `credential_copy_source_unix.go:13-14` — `os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)`.
- `kiro_home.go:89` — a cópia passou a usar `openCredentialCopySource`, não `os.Open`.
- `kiro_home.go:94-99` — `in.Stat()` (fstat do handle) mais `IsRegular` no descritor já
  aberto.
- `kiro_home.go:101-103` — `os.SameFile(srcInfo, openedInfo)` rejeita troca de caminho
  entre o guard e a abertura; como `srcInfo` vem de `os.Lstat`, uma troca de componente
  pai também resulta em inode diferente e é detectada.
- `kiro_home.go:63-70` — `syncCredentialFile` trocou `os.Stat` por `os.Lstat` e rejeita
  symlink ou não regular, fechando a resolução que seguia link.
- Não testável de forma determinística: os ramos `O_NOFOLLOW` e `SameFile` só são
  alcançáveis sob corrida real. Aceitável.

## M3 — reuso falha fechado: PASS

- `execenv.go:531-537` — o caminho de reuso AGY agora faz `return nil` após o `Warn`.
- `daemon.go:3495-3509` — o único chamador trata `env == nil` caindo no bloco `Prepare`,
  que falha fechado com erro explícito (`execenv.go:313-317`). O comentário do patch
  confere com o comportamento real do chamador.
- Teste novo: `TestReuseAntigravityFailsClosedOnInvalidSource`
  (`antigravity_home_test.go:129-144`) exige `Reuse(...) == nil` com token ausente.

## Testes novos confirmados

- `TestPrepareAntigravityHomeRejectsEmptyDestination` (`antigravity_home_test.go:78-85`) —
  fecha a lacuna 3 do review anterior.
- `TestPrepareAntigravityHomeReplacesDestinationSymlink`
  (`antigravity_home_test.go:87-109`) — símlink plantado no destino é substituído, alvo
  externo intacto, modo final 0600; fecha a lacuna 2.
- Modos de diretório (`antigravity_home_test.go:56-58`) — fecha a lacuna 1.
- As lacunas 5 e 6 ficam fechadas por construção: `0600` no `O_CREATE` torna o umask
  irrelevante para o arquivo, o `Chmod` explícito torna o umask irrelevante para os
  diretórios, e o `defer` de limpeza impede parcial legível.

## Residuais (nenhum bloqueia H1/H2/M1/M3)

- R1: `copyCredentialFile` agora força 0600 para **todos** os chamadores, incluindo
  `copyCredentialDir` usado por opencode (`opencode_home.go:57-73`) e cline
  (`cline_home.go:55`). Arquivos copiados nesses diretórios perdem o modo de origem e
  qualquer bit de execução. Mesmo uid, então leitura segue OK; só importa se algo copiado
  precisar ser executado. Não verifiquei se existe executável nesses caminhos. Codex usa
  copiador próprio (`codex_home.go:400`, 0644) e não é afetado.
- R2: kiro (`execenv.go:518-526`) e cline seguem falhando abertos no reuso, enquanto AGY
  passou a falhar fechado. Assimetria dentro da mesma função; recomenda-se decisão.
- R3: `credential_copy_source_windows.go:13-14` sempre retorna erro, logo qualquer HOME
  com isolamento de credencial falha fechado no Windows. Intencional pelo comentário;
  confirmar que não há alvo Windows.
- R4: estilo — `} else {` após `return nil` em `execenv.go:536-538` é `else` supérfluo;
  `gofmt` e `go vet` não reclamam, `revive`/`golangci` (indent-error-flow) reclamariam.
  Não bloqueante.
- R5: tabela de rejeição segue sem FIFO, socket ou device na origem; o guard `IsRegular`
  cobre, o teste não exercita.
- M2 (write-back do token renovado) permanece **aberto e sem decisão**: não faz parte de
  H1/H2/M1/M3 e o diff final não o altera.

## Limite de verificação (inalterado)

Sem toolchain Go neste host (`go` fora do PATH, sem `/usr/local/go`, `/opt/go` ou
`*/bin/go`), portanto `go test ./... -count=1`, `go vet ./...` e `go build ./...` não foram
re-executados por mim. Instalar runtime é STOP-AND-WAIT (AA-001 §0.1). O rollout continua
dependendo dessa evidência de gates mais das decisões escritas do owner na tabela
T1-a..T1-e.
