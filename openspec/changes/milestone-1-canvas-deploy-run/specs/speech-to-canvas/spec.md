## ADDED Requirements

### Requirement: Tiered STT Engine Selection

The system SHALL support a tiered Speech-to-Text strategy per master spec §5.3. Tier 1 (default) SHALL be the browser's native Web Speech API where available (Chrome, Edge). Tier 2 SHALL be the OpenAI Whisper API (audio/webm capture via `MediaRecorder`, POST to `https://api.openai.com/v1/audio/transcriptions`), used when (a) Web Speech API is unavailable, or (b) the user has selected Tier 2 in Settings → STT Engine. Tier 3 (Deepgram streaming) is post-launch.

The selected engine SHALL be configurable in Settings → STT Engine (master spec §8.10). Changing the engine SHALL take effect immediately for subsequent voice activations.

#### Scenario: Web Speech API is the default in Chrome

- **WHEN** a user on Chrome opens the voice input for the first time
- **THEN** Tier 1 (Web Speech API) is used and no Whisper API call is issued

#### Scenario: Whisper fallback when Web Speech is unavailable

- **WHEN** the browser does not expose `SpeechRecognition` and the user activates voice
- **THEN** the system uses Tier 2 (Whisper API) automatically, captures via `MediaRecorder` at 16 kHz mono, and posts the WebM to OpenAI

#### Scenario: User-forced Tier 2

- **WHEN** the user has selected "Whisper API" in Settings → STT Engine and Web Speech is available
- **THEN** the system still uses Whisper for voice input

### Requirement: Audio Capture Privacy

The system SHALL request microphone permission via the browser's native `getUserMedia` prompt at the moment of activation, not at app load. Audio data SHALL NOT be transmitted to any AgentVerse-managed backend. Tier 1 audio is processed by the browser's STT vendor (typically Google or Microsoft); Tier 2 audio is sent to OpenAI using the user's own API key. Transcripts SHALL be held only in browser session memory and SHALL NOT be persisted.

#### Scenario: No persistent mic access

- **WHEN** the user grants microphone permission for one voice activation
- **THEN** stopping the recording immediately releases the microphone (no `MediaStream` is retained between activations)

#### Scenario: Transcripts not persisted

- **WHEN** a voice session ends
- **THEN** no transcript is written to IndexedDB or any other persistent store

### Requirement: NLU Intent Extraction via BYOK LLM

The system SHALL parse the final transcript into a structured `VoiceIntent` (per master spec §5.6) using a structured-extraction call against the user's own LLM key. The system SHALL select the cheapest available validated provider in this order: Gemini Flash → GPT-4o-mini → Anthropic Haiku. If no LLM key is validated, voice input SHALL be disabled with an inline message directing the user to Settings → Providers.

The NLU prompt SHALL be the bilingual prompt template in master spec §5.6, customized at runtime with the user's transcript. The latency budget for the parse call SHALL be 3 seconds; if exceeded, the user SHALL see a soft warning but the parse SHALL continue. Cost per utterance SHALL be approximately $0.001–$0.005 (master spec §16.1).

#### Scenario: NLU extracts a CreateCanvasIntent

- **WHEN** the user says "I need a code review pipeline with a supervisor on Kiro and two developers on Claude"
- **THEN** the parsed `VoiceIntent` has `type: "create_canvas"` and a `parsed.nodes` array containing entries for supervisor (kiro_cli, count=1) and developer (claude_code, count=2)

#### Scenario: No LLM key disables voice input

- **WHEN** the user has zero validated LLM providers and clicks the mic button
- **THEN** an inline message reads "Voice input requires a validated LLM provider — visit Settings → Providers" and the mic does not activate

### Requirement: Canvas Generation from Voice Intent

When the parsed intent is `type: "create_canvas"`, the system SHALL produce a draft `CanvasDocument` per master spec §5.7. Auto-layout SHALL place nodes in left-to-right hierarchy. Provider names from the transcript SHALL be resolved via the master-spec §5.6 mapping (e.g., "Kiro" → `kiro_cli`). The supervisor role SHALL be set as the entry-point. The generated canvas SHALL be presented to the user through the Voice UI's intent-preview confirm step (see below) before being handed off to the Canvas Builder.

#### Scenario: Voice generates a 4-node canvas

- **WHEN** the canonical pt-BR example transcript from master spec §5.1 is parsed and generated
- **THEN** the resulting `CanvasDocument` has 4 nodes (1 supervisor + 2 developers + 1 reviewer), 3 handoff edges (supervisor→developer×2, developer→reviewer), and `deploy_state: { status: "draft" }`

### Requirement: Voice UI With Intent-Preview Confirm Step

The system SHALL render a Voice UI panel per master spec §5.9 with five states: `idle`, `listening`, `processing`, `confirming`, `error`. In `confirming`, the panel SHALL display the parsed intent summary (node counts by role + provider, edge counts by type, confidence percentage) and SHALL offer three actions: **Cancel**, **Edit Before Deploy** (loads the draft canvas into the Builder), and **Generate** (loads the draft and immediately attempts deploy via the Reconciler).

The panel SHALL be activatable by clicking the mic button or by the keyboard shortcut `Ctrl+Shift+V` / `Cmd+Shift+V`.

#### Scenario: Confirm step shows parsed summary

- **WHEN** parsing returns a CreateCanvasIntent
- **THEN** the panel transitions to `confirming` state and displays the node/edge summary plus a confidence percentage
- **AND** all three action buttons (Cancel, Edit Before Deploy, Generate) are present

#### Scenario: Edit Before Deploy hands off to Builder

- **WHEN** the user clicks "Edit Before Deploy" in `confirming` state
- **THEN** the Voice UI closes and the Canvas Builder opens with the generated `CanvasDocument` loaded for editing

### Requirement: Bilingual Support pt-BR + en-US

The system SHALL accept voice input in pt-BR (Brazilian Portuguese) and en-US (US English). The default language for Web Speech API SHALL be pt-BR per master spec §5.4; users SHALL be able to switch to en-US in Settings. The NLU prompt SHALL include bilingual keyword mapping per master spec §5.6 so users SHALL be able to mix pt-BR and en-US within a single utterance.

#### Scenario: Mixed-language utterance is parsed

- **WHEN** the user says "Cria um pipeline com um supervisor e dois developers usando handoff"
- **THEN** the parsed intent contains the supervisor + developer nodes plus a handoff edge type, despite the mixed pt-BR/en wording
