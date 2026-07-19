---
agent: Agent-2-NIM-Isolation
model: glm-5.2 (High)
stream: W1-T1.2
started_at: 2026-07-12T06:22:58Z
finished_at: 2026-07-12T06:32:41Z
status: DONE
authority: openspec/changes/native-runtimes-onboarding (task 1.2) + specs/agent-runtimes/spec.md
files_changed:
  - multica-auth-work/server/internal/daemon/execenv/nim_home.go            (NEW)
  - multica-auth-work/server/internal/daemon/execenv/nim_home_test.go       (NEW)
  - multica-auth-work/server/internal/rotation/detector_nim.go             (NEW)
  - multica-auth-work/server/internal/rotation/detector_nim_test.go        (NEW)
  - multica-auth-work/server/internal/daemon/rotation_detector_nim.go       (NEW)
  - multica-auth-work/server/internal/daemon/rotation_detector_nim_test.go  (NEW)
build_result: GREEN (go build ./... exit 0; go vet clean; go test rotation+execenv+daemon ok)
wiring_patch_for_kiro:
  - daemon.go:3910 requiresCredentialIsolation += "nim"  [HEADLINE]
  - rotation/detector.go:50 matchesVendorExhaustion += case "nim"
  - execenv/execenv.go Environment struct += NimDataDir field
  - execenv/execenv.go Prepare += nim branch (mirror cline ~L335)
  - execenv/execenv.go Reuse += nim branch (mirror cline ~L537)
  - execenv/execenv.go CredentialEnv += case "nim" (NIM_HOME)
notes: |
  Task 1.2 concluida. 3 arquivos standalone + 3 testes, espelhando cline/opencode/
  antigravity. NENHUM arquivo compartilhado editado. Todo o wiring neste check-in
  como patch para o Kiro aplicar na Wave 2. Tests verdes sem depender do wiring.
---

# CHECKIN DONE — Agent-2 (NIM-Isolation), Task 1.2

## O que foi feito
Isolamento de credencial + deteccao de exaustao/rotacao do runtime nativo NIM,
espelhando os padroes codex/opencode/cline existentes:

1. execenv/nim_home.go — prepareNimHome / NimHomeOptions / resolveNimSourceDir.
   Isola o dir de credencial NIM (<accountHome>/.nim/) num per-task nimDataDir
   (modo 0700), copiado AS-IS (nao inspeciona conteudo => anti-vazamento).
   Resolve .nim/, nim/ bare, ou markers diretos (credentials.json/api_key/config.json).
   Empty source => dir vazio (fail-closed). Espelha cline_home.go + antigravity_home.go.

2. rotation/detector_nim.go — matchesNimExhaustion matcher (banner OpenAI-compat
   429/rate-limit/quota + NVIDIA/NIM-branded + credits-exhausted, pareado com
   reset/retry). Espelha detector_cline.go / detector_opencode.go. Sem credencial em log.

3. daemon/rotation_detector_nim.go — NimExhaustionDetector struct implementando
   rotation.ExhaustionDetector via o helper compartilhado detectExhaustion, +
   matchesNimScreenExhaustion (matcher daemon-layer, mesmos padroes). Espelha
   rotation_detector_cline.go.

## Arquivos alterados (todos NEW, disjuntos)
- internal/daemon/execenv/nim_home.go (+nim_home_test.go)
- internal/rotation/detector_nim.go (+detector_nim_test.go)
- internal/daemon/rotation_detector_nim.go (+rotation_detector_nim_test.go)

## Evidencia de build/test (verde-em-container)
Toolchain: go1.26.4 (linux/amd64), GOPATH=/home/dataops-lab/go.
- gofmt -l nos 6 arquivos: clean (sem saida).
- go build ./...: exit 0 (servidor inteiro compila).
- go vet ./internal/rotation/ ./internal/daemon/execenv/ ./internal/daemon/: clean.
- go test ./internal/rotation/ ./internal/daemon/execenv/: ok (0.019s / 0.210s).
- go test ./internal/daemon/ (pacote completo): ok (19.263s).
- go test -v -run 'Nim' ...: todos PASS:
  - rotation: TestNimMatcherDetectsExhaustion (14 subtests) PASS
  - execenv: TestPrepareNimHomePerAccountIsolatesCredentialDir,
    TestPrepareNimHomeFallbackWhenNoAccount, TestPrepareNimHomeResolvesBareNimSubdir,
    TestPrepareNimHomeEmptyWhenNoSource, TestPrepareNimHomeRejectsEmptyDataDir — PASS
  - daemon: TestNimExhaustionDetectorMatcher (12 subtests) +
    TestNimExhaustionDetectorViaDetect (8 subtests: screen reset parse, 429,
    503 no-rotate, high-traffic no-rotate, constructor) — PASS

## PATCH DE WIRING (Kiro aplica na Wave 2)
NENHUM aplicado por mim (arquivos compartilhados/dispatchers). Aditivos, espelhando cline.

### W2-1. daemon.go — requiresCredentialIsolation (+nim)  [HEADLINE]
Arquivo: internal/daemon/daemon.go, func requiresCredentialIsolation (~L3910).
```diff
 func requiresCredentialIsolation(provider string) bool {
 	switch strings.ToLower(strings.TrimSpace(provider)) {
-	case "codex", "kiro", "antigravity", "glm", "cline", "opencode":
+	case "codex", "kiro", "antigravity", "glm", "cline", "opencode", "nim":
 		return true
 	default:
 		return false
 	}
 }
```

### W2-2. rotation/detector.go — matchesVendorExhaustion (+nim)
Arquivo: internal/rotation/detector.go, func matchesVendorExhaustion (~L50).
```diff
 	case "opencode":
 		return matchesOpenCodeExhaustion(screenText)
+	case "nim":
+		return matchesNimExhaustion(screenText)
 	default:
 		return false
 	}
```

### W2-3. execenv/execenv.go — Environment struct (+NimDataDir field)
Arquivo: internal/daemon/execenv/execenv.go, struct Environment (apos OpenCodeConfigHome ~L180).
```diff
 	OpenCodeConfigHome string
+	// NimDataDir is the per-task isolated NIM credential directory (set only for
+	// the nim provider when a per-account credential is used). The native nim
+	// runtime (server/pkg/agent/nim.go) reads the account's NVIDIA API key from
+	// here instead of a shared/global store.
+	NimDataDir string
```

### W2-4. execenv/execenv.go — Prepare (+nim branch)
Arquivo: internal/daemon/execenv/execenv.go, func Prepare (apos branch opencode/glm ~L335). Mirror cline (~L308-322).
```diff
+	// NIM native runtime: isola a NVIDIA API key por-conta num dir per-task.
+	// Empty CredentialAccountHome = shared/global behavior.
+	if params.Provider == "nim" && params.CredentialAccountHome != "" {
+		nimDataDir := filepath.Join(envRoot, "nim-data-dir")
+		if err := prepareNimHome(nimDataDir, NimHomeOptions{AccountHome: params.CredentialAccountHome}, logger); err != nil {
+			return nil, fmt.Errorf("execenv: prepare nim-data-dir: %w", err)
+		}
+		env.NimDataDir = nimDataDir
+	}
```

### W2-5. execenv/execenv.go — Reuse (+nim branch)
Arquivo: internal/daemon/execenv/execenv.go, func Reuse (apos branch opencode/glm reuse ~L537). Mirror cline reuse (~L521).
```diff
+	if params.Provider == "nim" && params.CredentialAccountHome != "" {
+		nimDataDir := filepath.Join(envRoot, "nim-data-dir")
+		if err := prepareNimHome(nimDataDir, NimHomeOptions{AccountHome: params.CredentialAccountHome}, logger); err != nil {
+			return nil, fmt.Errorf("execenv: prepare nim-data-dir: %w", err)
+		}
+		env.NimDataDir = nimDataDir
+	}
```

### W2-6. execenv/execenv.go — CredentialEnv (+case "nim")
Arquivo: internal/daemon/execenv/execenv.go, func (*Environment).CredentialEnv (~L617).
Expoe NIM_HOME; o backend nim le a key de $NIM_HOME/credentials.json (execenv nao
inspeciona segredo — anti-vazamento, igual ao cline que expoe CLINE_DATA_DIR).
```diff
 	case "opencode", "glm":
 		if e.OpenCodeDataHome != "" || e.OpenCodeConfigHome != "" {
 			...
 		}
+	case "nim":
+		if e.NimDataDir != "" {
+			return map[string]string{"NIM_HOME": e.NimDataDir}
+		}
 	}
 	return nil
```

## Contrato NIM (para Agent-1 nim.go + Kiro W2)
- Credencial por-conta: NVIDIA API key em <accountHome>/.nim/credentials.json
  (ou .nim/api_key). Preparador copia o dir inteiro AS-IS para NIM_HOME.
- Runtime nim.go (Agent-1) le a key de $NIM_HOME/credentials.json e envia como
  Authorization: Bearer <key> contra https://integrate.api.nvidia.com/v1.
- Deteccao de exaustao: HTTP 429 => rotate (upstream); banner de tela
  (rate/quota/credits + reset) => rotate via matchesNimExhaustion.

## Riscos / follow-ups
1. Padroes de banner NIM sao best-effort OpenAI-compat/NVIDIA (doc 36 §2.1:
   confirmar contra a tela real no deploy). Ajustar se o gateway real surfar frase diferente.
2. Layout .nim/credentials.json deve bater com o que o Agent-1 (nim.go) ler —
   comunicar a Kiro para alinhar A1/A2 antes do W2.
3. Smoke final (criar agente nim, 1 task, ver tokens) e gate do Kiro (Wave 3).

## Status
Task 1.2 DONE. Aguarda Kiro aplicar wiring W2-1..W2-6 e gates W3.
