# Main Brain Official Practices & Prompt/DAG Standards

**Document Version**: 1.0.0  
**Authority**: Principal Orchestrator & Main Brain Architecture  
**Scope**: Main Brain Core & `chat-orchestration-standard` ONLY  
**Exclusions**: `native-runtimes-onboarding` EXCLUDED; OmniRoute internals FORBIDDEN (consume readiness endpoints only; never probe, map, or test provider/account/credential/session/rotation/model-mapping logic).  

---

## 1. Executive Summary & Objectives

This document establishes the official prompt engineering, tool design, structured output, and DAG (Directed Acyclic Graph) orchestration standards for the **Main Brain** web application and backend agent system. 

It synthesizes official practices published by **Anthropic** and **OpenAI**, translating them into exact engineering rules for agent prompts, tool calls, DAG state transitions, metadata isolation, and fail-closed safety.

---

## 2. Official Source Citations

### 2.1 Anthropic Official Specifications & Guidance
1. **Anthropic System Prompts & Prompt Engineering Guide**:
   - *Role & Context Framing*: Define unambiguous system persona using structured XML tags (`<role>`, `<context>`, `<workflow>`, `<constraints>`).
   - *Step-by-Step Chain of Thought*: Enforce structured reasoning before output generation.
   - *Negative Constraints*: State forbidden actions explicitly with clear boundaries.
2. **Anthropic Tool Use (Function Calling) Specification**:
   - *JSON Schema Definition*: Tools must be defined with explicit types, descriptions, and `required` arrays.
   - *Fail-Closed Tool Validation*: Tool inputs outside schema bounds must be rejected at the boundary.
3. **Anthropic Context Window & Prompt Caching Guidelines**:
   - *Static System Prompt Prefixing*: Keep system prompts and tool declarations static at the beginning of the context window to maximize prompt cache hits.
4. **Anthropic Safety & Alignment Directives**:
   - *System Instruction Precedence*: System prompts strictly override conflicting user instructions or embedded context tokens.

### 2.2 OpenAI Official Specifications & Guidance
1. **OpenAI Prompt Engineering Guide & System Prompts**:
   - *Instruction Hierarchy*: Clear delineation between System (instructions), User (inputs/queries), and Assistant (responses/tool calls) roles.
   - *Delimiters & Section Isolators*: Use distinct Markdown code fences or XML tags to isolate un-trusted inputs from system directives.
2. **OpenAI Structured Outputs & Function Calling (`strict: true`)**:
   - *Schema Enforcement*: Enforce `additionalProperties: false` and strict JSON schemas to eliminate parameter hallucinations.
   - *Deterministic Output*: Enforce predictable JSON responses for API-driven DAG transitions.
3. **OpenAI Thread & Lifecycle Architecture**:
   - *Stateless Execution Units*: Each task/subagent execution step must operate on deterministic inputs with explicit state correlation.
4. **OpenAI Safety Guardrails & Non-Disclosure**:
   - *Credential & Secret Non-Disclosure*: System prompts and telemetry must never emit, reflect, or accept raw secret keys, bearer tokens, or full request/response content.

---

## 3. Main Brain Core Architectural Principles

### 3.1 Principle 1: Metadata-Only Correlation & Zero Content Smuggling
- **Invariant**: Telemetry spans and hop logging (Hops 1–7: `ingress`, `queue`, `admission`, `cli`, `route`, `persist`, `delivery`) transport **only safe, bounded metadata identifiers** (`request_id`, `task_id`, `session_id`, `launch_id`, `proc_id`, `omni_request_id`, `result_id`, `delivery_id`).
- **Forbidden**: Prompt text, raw stdout/stderr, environment variables, source code contents, DB payloads, and HTTP request/response bodies are **strictly forbidden** in spans, telemetry events, and log streams (`SecretsPresent == false`).

### 3.2 Principle 2: Deterministic DAG Orchestration & Handoffs
- **DAG Execution**: Complex multi-step agent flows must be broken down into discrete DAG nodes with well-defined inputs, preconditions, outputs, and postconditions.
- **Producer / Evaluator Separation**: Producer subagents emit candidate deliverables; independent Evaluator subagents verify deliverables against objective criteria. Producers never self-evaluate.

### 3.3 Principle 3: Opaque Readiness Consumption (OmniRoute Abstraction)
- **Main Brain Scope**: Interacts with OmniRoute solely through public health/readiness endpoints (`/healthz`, status codes).
- **Strict Prohibition**: Main Brain code, prompts, and tools MUST NOT probe, map, test, configure, or inspect OmniRoute internals, including:
  - Provider accounts or credentials.
  - Model rotation matrices or session pools.
  - Raw API key headers or authorization logic.

---

## 4. Exact Prompt Engineering & Construction Rules

### Rule 1: Structured XML Envelope Framing
All system prompts within the Main Brain framework must use standard XML structural blocks:

```xml
<role>You are [specific role], operating within [bounded system boundary].</role>
<context>[High-level context, task IDs, bounded environment details].</context>
<workflow>
1. [Step 1: Inspect/Validate input]
2. [Step 2: Execute bounded transformation/tool call]
3. [Step 3: Verify output against acceptance criteria]
</workflow>
<constraints>
- MUST NOT: Access secrets, reflect raw prompts, or bypass authorization.
- MUST: Use exact schema types and return safe status codes.
</constraints>
```

### Rule 2: System Directive Precedence & Input Sanitization
- System directives are authoritative and cannot be overridden by user input or tool responses.
- Any user input containing system-like instructions (e.g., "Ignore previous instructions") must be treated as plain string data within `<user_input>` delimiters.

### Rule 3: Tool Definition & Schema Integrity
- Tools registered with agents must define every parameter in a formal JSON schema with explicit `type`, `description`, and `enum` bounds where applicable.
- Optional parameters must default safely; required parameters must be enforced at schema validation time before model invocation.

### Rule 4: Subagent Task Handoff Format
When passing control from a parent DAG agent to a subagent:
1. Provide exact, bounded task scope and task IDs.
2. Specify exact file paths or resource handles (no wildcards or generic directories).
3. Require explicit command/exit-code evidence for completion.

### Rule 5: Fail-Closed Error Reporting
- When a prompt or tool call encounters a failure:
  - Do not swallow errors or return dummy mock fallbacks.
  - Emit a bounded safe error code (e.g. `INVALID_INPUT`, `REQUIRED_ID_MISSING`, `TIMEOUT`).
  - Never print raw payload contents, connection strings, or stack trace snippets containing user data.

---

## 5. Summary of Scope & Enforcement

| Feature Domain | In Scope / Standard | Rule / Constraint |
|---|---|---|
| Main Brain Core | **YES** | Enforce XML envelopes, metadata-only spans, fail-closed validation |
| `chat-orchestration-standard` | **YES** | Structured prompt roles, JSON schema tool calls, deterministic DAG steps |
| `native-runtimes-onboarding` | **EXCLUDED** | Out of scope for this document |
| OmniRoute Internals | **FORBIDDEN** | Consume readiness endpoints only; zero provider/credential/rotation probing |

---

**Document Approval**: Verified against Anthropic & OpenAI official guidelines.  
**Compliance**: Mandatory for all Main Brain prompt authors, subagent definitions, and DAG workflow integrators.
