"use client";

import {
  type KeyboardEvent,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@multica/core/api";
import { useWorkspaceId } from "@multica/core/hooks";
import type { RuntimeBindingDetail } from "@multica/core/types";
import { Badge } from "@multica/ui/components/ui/badge";
import { Button } from "@multica/ui/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@multica/ui/components/ui/card";
import { Input } from "@multica/ui/components/ui/input";
import { Textarea } from "@multica/ui/components/ui/textarea";
import { PageHeader } from "../../layout/page-header";
import {
  capabilityModels,
  describeReasonCodeIssue,
  diffRuntimeConfigurations,
  isValidReasonCode,
  safeRuntimeConfiguration,
  safeRuntimeDigest,
  suggestReasonCode,
  withModelAndReasoning,
} from "../configuration";

const selectClass =
  "h-8 w-full rounded-lg border border-input bg-background px-2.5 text-sm outline-none focus:ring-2 focus:ring-ring";

export default function RuntimeManagerPage() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  const queryRoot = ["runtime-manager", workspaceId] as const;
  const [selectedBindingId, setSelectedBindingId] = useState("");
  const [selectedSessionId, setSelectedSessionId] = useState("");
  const [notice, setNotice] = useState<{
    kind: "success" | "error";
    text: string;
  } | null>(null);
  const [busy, setBusy] = useState(false);

  const standardsQuery = useQuery({
    queryKey: [...queryRoot, "standards"],
    queryFn: () => api.listRuntimeStandards(),
  });
  const sessionsQuery = useQuery({
    queryKey: [...queryRoot, "sessions"],
    queryFn: () => api.listRuntimeSessions(),
  });
  const bindingsQuery = useQuery({
    queryKey: [...queryRoot, "bindings"],
    queryFn: () => api.listRuntimeBindings(workspaceId),
    enabled: !!workspaceId,
  });
  const homesQuery = useQuery({
    queryKey: [...queryRoot, "homes"],
    queryFn: () => api.listRuntimeCredentialHomes(workspaceId),
    enabled: !!workspaceId,
  });
  const detailQuery = useQuery({
    queryKey: [...queryRoot, "binding", selectedBindingId],
    queryFn: () => api.getRuntimeBinding(workspaceId, selectedBindingId),
    enabled: !!workspaceId && !!selectedBindingId,
  });

  const standards = standardsQuery.data?.items ?? [];
  const sessions = sessionsQuery.data?.items ?? [];
  const bindings = bindingsQuery.data?.items ?? [];
  const homes = homesQuery.data?.items ?? [];

  useEffect(() => {
    if (!selectedBindingId && bindings[0]) setSelectedBindingId(bindings[0].id);
  }, [bindings, selectedBindingId]);
  useEffect(() => {
    if (!selectedSessionId && sessions[0]) setSelectedSessionId(sessions[0].id);
  }, [sessions, selectedSessionId]);

  async function perform(label: string, operation: () => Promise<unknown>) {
    setBusy(true);
    setNotice(null);
    try {
      await operation();
      await queryClient.invalidateQueries({ queryKey: queryRoot });
      setNotice({ kind: "success", text: label });
    } catch (error) {
      const text =
        error instanceof Error && error.message.startsWith("Runtime Manager request failed")
          ? error.message
          : "Runtime Manager request failed.";
      setNotice({ kind: "error", text });
    } finally {
      setBusy(false);
    }
  }

  const loading =
    standardsQuery.isLoading || sessionsQuery.isLoading || bindingsQuery.isLoading;
  const error = standardsQuery.error ?? sessionsQuery.error ?? bindingsQuery.error ?? homesQuery.error;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader className="justify-between px-5">
        <div>
          <h1 className="text-sm font-medium">Runtime Manager</h1>
          <p className="text-xs text-muted-foreground">
            Reusable sessions, subscriptions, and versioned runtime configuration
          </p>
        </div>
        {notice && (
          <span
            role={notice.kind === "error" ? "alert" : "status"}
            aria-live={notice.kind === "error" ? "assertive" : "polite"}
            className="max-w-md truncate text-xs text-muted-foreground"
          >
            {notice.text}
          </span>
        )}
      </PageHeader>

      <main className="min-h-0 flex-1 overflow-y-auto p-5" aria-busy={busy || loading}>
        {loading ? (
          <p role="status" aria-live="polite" className="text-sm text-muted-foreground">Loading runtime controls…</p>
        ) : error ? (
          <p role="alert" className="text-sm text-destructive">
            Runtime Manager request failed.
          </p>
        ) : (
          <div className="mx-auto grid max-w-6xl gap-5">
            <SessionControls
              standards={standards}
              sessions={sessions}
              selectedSessionId={selectedSessionId}
              setSelectedSessionId={setSelectedSessionId}
              busy={busy}
              perform={perform}
              workspaceId={workspaceId}
            />
            <div className="grid gap-5 lg:grid-cols-[17rem_1fr]">
              <BindingList
                bindings={bindings}
                selectedBindingId={selectedBindingId}
                onSelect={setSelectedBindingId}
              />
              <BindingControls
                detail={detailQuery.data}
                homes={homes}
                loading={detailQuery.isLoading}
                busy={busy}
                workspaceId={workspaceId}
                perform={perform}
              />
            </div>
          </div>
        )}
      </main>
    </div>
  );
}

function SessionControls({
  standards,
  sessions,
  selectedSessionId,
  setSelectedSessionId,
  busy,
  perform,
  workspaceId,
}: {
  standards: Array<{ id: string; name: string }>;
  sessions: Array<{ id: string; name: string; provider: string; runtime_kind: string }>;
  selectedSessionId: string;
  setSelectedSessionId: (id: string) => void;
  busy: boolean;
  perform: (label: string, operation: () => Promise<unknown>) => Promise<void>;
  workspaceId: string;
}) {
  const [name, setName] = useState("");
  const [provider, setProvider] = useState("");
  const [runtimeKind, setRuntimeKind] = useState("");
  const [standardId, setStandardId] = useState("");
  const [runtimeId, setRuntimeId] = useState("");
  const [agentId, setAgentId] = useState("");

  async function createSession() {
    if (!name || !provider || !runtimeKind || !standardId) return;
    await perform("Reusable runtime session created.", async () => {
      const created = await api.createRuntimeSession({
        name,
        provider,
        runtime_kind: runtimeKind,
        standard_id: standardId,
      });
      setSelectedSessionId(created.id);
      setName("");
    });
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Reusable owner-global session</CardTitle>
        <CardDescription>
          Sessions are accountless. Enrollment reuses existing runtime and agent IDs and does not create a runtime, agent, or home.
        </CardDescription>
      </CardHeader>
      <CardContent className="grid gap-4 lg:grid-cols-2">
        <section className="grid gap-2">
          <h3 className="text-sm font-medium">Create session</h3>
          <div className="grid gap-2 sm:grid-cols-2">
            <Input aria-label="Session name" placeholder="Session name" value={name} onChange={(event) => setName(event.target.value)} />
            <Input aria-label="Provider" placeholder="Provider" value={provider} onChange={(event) => setProvider(event.target.value)} />
            <Input aria-label="Runtime kind" placeholder="Runtime kind" value={runtimeKind} onChange={(event) => setRuntimeKind(event.target.value)} />
            <select aria-label="Runtime standard" className={selectClass} value={standardId} onChange={(event) => setStandardId(event.target.value)}>
              <option value="">Select standard</option>
              {standards.map((standard) => <option key={standard.id} value={standard.id}>{standard.name}</option>)}
            </select>
          </div>
          <Button size="sm" disabled={busy || !name || !provider || !runtimeKind || !standardId} onClick={createSession}>Create reusable session</Button>
        </section>
        <section className="grid gap-2">
          <h3 className="text-sm font-medium">Reuse by enrollment</h3>
          <select aria-label="Runtime session" className={selectClass} value={selectedSessionId} onChange={(event) => setSelectedSessionId(event.target.value)}>
            <option value="">Select session</option>
            {sessions.map((session) => <option key={session.id} value={session.id}>{session.name} · {session.provider} · {session.runtime_kind}</option>)}
          </select>
          <div className="grid gap-2 sm:grid-cols-2">
            <Input aria-label="Existing runtime ID" placeholder="Existing runtime ID" value={runtimeId} onChange={(event) => setRuntimeId(event.target.value)} />
            <Input aria-label="Existing agent ID" placeholder="Existing agent ID" value={agentId} onChange={(event) => setAgentId(event.target.value)} />
          </div>
          <Button
            size="sm"
            variant="outline"
            disabled={busy || !selectedSessionId || !runtimeId || !agentId}
            onClick={() => perform("Existing runtime and agent enrolled.", () => api.enrollRuntimeSession(workspaceId, selectedSessionId, { runtime_id: runtimeId, agent_id: agentId }))}
          >
            Enroll existing runtime and agent
          </Button>
        </section>
      </CardContent>
    </Card>
  );
}

export function BindingList({
  bindings,
  selectedBindingId,
  onSelect,
}: {
  bindings: Array<{ id: string; runtime_id: string; provider: string; state: string; home_ref?: string | null }>;
  selectedBindingId: string;
  onSelect: (id: string) => void;
}) {
  const optionRefs = useRef<Array<HTMLButtonElement | null>>([]);

  function focusOption(index: number) {
    const binding = bindings[index];
    if (!binding) return;
    onSelect(binding.id);
    optionRefs.current[index]?.focus();
  }

  function handleOptionKeyDown(event: KeyboardEvent<HTMLButtonElement>, index: number) {
    let nextIndex: number | null = null;
    switch (event.key) {
      case "ArrowDown":
      case "ArrowRight":
        nextIndex = (index + 1) % bindings.length;
        break;
      case "ArrowUp":
      case "ArrowLeft":
        nextIndex = (index - 1 + bindings.length) % bindings.length;
        break;
      case "Home":
        nextIndex = 0;
        break;
      case "End":
        nextIndex = bindings.length - 1;
        break;
      default:
        return;
    }
    event.preventDefault();
    focusOption(nextIndex);
  }

  return (
    <Card className="h-fit">
      <CardHeader>
        <CardTitle>Runtime bindings</CardTitle>
        <CardDescription>Workspace projections only</CardDescription>
      </CardHeader>
      <CardContent
        role="listbox"
        aria-label="Runtime bindings"
        aria-orientation="vertical"
        className="grid gap-2"
      >
        {bindings.length === 0 && <p className="text-xs text-muted-foreground">No runtime bindings.</p>}
        {bindings.map((binding, index) => {
          const selected = selectedBindingId === binding.id;
          return (
            <button
              type="button"
              role="option"
              aria-selected={selected}
              tabIndex={selected || (!selectedBindingId && index === 0) ? 0 : -1}
              ref={(node) => {
                optionRefs.current[index] = node;
              }}
              key={binding.id}
              onClick={() => onSelect(binding.id)}
              onKeyDown={(event) => handleOptionKeyDown(event, index)}
              className={`grid gap-1 rounded-lg border p-3 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${selected ? "border-primary bg-muted" : "hover:bg-muted/50"}`}
            >
              <span className="truncate text-sm font-medium">Runtime {binding.runtime_id}</span>
              <span className="flex flex-wrap gap-1"><Badge variant="outline">{binding.provider}</Badge><Badge variant="secondary">{binding.state}</Badge></span>
              <span className="truncate font-mono text-[11px] text-muted-foreground">{binding.home_ref ? `home ${binding.home_ref}` : "No subscription attached"}</span>
            </button>
          );
        })}
      </CardContent>
    </Card>
  );
}

function BindingControls({
  detail,
  homes,
  loading,
  busy,
  workspaceId,
  perform,
}: {
  detail: RuntimeBindingDetail | undefined;
  homes: Array<{ home_ref: string; generation: number; provider: string; state: string; health: string }>;
  loading: boolean;
  busy: boolean;
  workspaceId: string;
  perform: (label: string, operation: () => Promise<unknown>) => Promise<void>;
}) {
  const [homeRef, setHomeRef] = useState("");
  const [model, setModel] = useState("");
  const [reasoning, setReasoning] = useState("");
  const [reason, setReason] = useState("");
  const [candidateVersionId, setCandidateVersionId] = useState("");
  const [rollbackVersionId, setRollbackVersionId] = useState("");

  const models = useMemo(() => capabilityModels(detail?.capabilities), [detail?.capabilities]);
  const chosenModel = models.find((item) => item.model_id === model);
  const availableHomes = homes.filter((home) => home.health === "healthy" && home.state === "unassigned");
  const active = detail?.active_configuration;
  const activeVersionId = detail?.active_configuration_version_id ?? null;
  const activeVersion = detail?.versions.find((version) => version.id === activeVersionId);
  const candidate = detail?.versions.find((version) => version.id === candidateVersionId);
  const safeConfig = safeRuntimeConfiguration(active);
  const diff = diffRuntimeConfigurations(active, candidate?.configuration);
  const validationLabel = candidate?.validation
    ? candidate.validation.valid
      ? "valid"
      : "invalid"
    : "unvalidated";

  // C3 requires a symbolic reason code and a positive binding generation on
  // every configuration mutation. Both are checked here so an operator sees why
  // a control is unavailable instead of receiving an opaque 400.
  const reasonIssue = describeReasonCodeIssue(reason);
  const reasonReady = isValidReasonCode(reason);
  const reasonSuggestion =
    !reasonReady && reason.trim() ? suggestReasonCode(reason) : "";
  const bindingGeneration = detail?.binding_generation ?? 0;
  const generationReady = bindingGeneration > 0;

  useEffect(() => {
    if (!detail) return;
    setModel(detail.active_configuration?.values.model ?? "");
    setReasoning(detail.active_configuration?.values.reasoning_effort ?? "");
    setCandidateVersionId(detail.versions.find((version) => version.state !== "active")?.id ?? "");
    setRollbackVersionId(detail.versions.find((version) => version.id !== detail.active_configuration_version_id)?.id ?? "");
    setHomeRef("");
  }, [detail?.id]);

  if (loading) return <Card><CardContent>Loading binding…</CardContent></Card>;
  if (!detail) return <Card><CardContent>Select a runtime binding.</CardContent></Card>;

  const createVersion = () => {
    if (!active || !model || !reasonReady) return Promise.resolve();
    return perform("Inactive immutable configuration version created.", async () => {
      const created = await api.createRuntimeBindingConfigurationVersion(
        workspaceId,
        detail.id,
        { configuration: withModelAndReasoning(active, model, reasoning), reason: reason.trim() },
      );
      setCandidateVersionId(created.id);
    });
  };

  return (
    <div className="grid gap-5">
      <Card>
        <CardHeader>
          <CardTitle className="flex flex-wrap items-center gap-2">
            Runtime configuration <Badge variant="outline">generation {detail.binding_generation}</Badge>
          </CardTitle>
          <CardDescription>
            Effective precedence is platform → standard → runtime → explicitly delegable task. Running tasks remain pinned.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4">
          <dl className="grid gap-2 text-xs sm:grid-cols-2 xl:grid-cols-4">
            {Object.entries(safeConfig).map(([field, value]) => (
              <div key={field} className="rounded-lg border p-2">
                <dt className="text-muted-foreground">{field.replaceAll("_", " ")}</dt>
                <dd className="mt-1 truncate font-mono">{value}</dd>
              </div>
            ))}
          </dl>
          <dl aria-label="Pinned runtime digests" className="grid gap-2 text-xs sm:grid-cols-2">
            <div className="min-w-0 rounded-lg border p-2">
              <dt className="text-muted-foreground">Configuration digest</dt>
              <dd className="mt-1 break-all font-mono">
                {safeRuntimeDigest(activeVersion?.configuration_digest)}
              </dd>
            </div>
            <div className="min-w-0 rounded-lg border p-2">
              <dt className="text-muted-foreground">Capability digest</dt>
              <dd className="mt-1 break-all font-mono">
                {safeRuntimeDigest(detail.capabilities?.capability_digest)}
              </dd>
            </div>
          </dl>
          <div className="grid gap-2 sm:grid-cols-2">
            <label className="grid gap-1 text-xs">Pinned capability model
              <select className={selectClass} value={model} onChange={(event) => { setModel(event.target.value); setReasoning(""); }}>
                <option value="">Select model</option>
                {models.map((item) => <option key={item.model_id} value={item.model_id}>{item.display_name ?? item.model_id}</option>)}
              </select>
            </label>
            <label className="grid gap-1 text-xs">Reasoning effort
              <select className={selectClass} value={reasoning} onChange={(event) => setReasoning(event.target.value)}>
                <option value="">Runtime default</option>
                {(chosenModel?.reasoning_efforts ?? []).map((effort) => <option key={effort} value={effort}>{effort}</option>)}
              </select>
            </label>
          </div>
          <div className="grid gap-1">
            <Textarea
              aria-label="Configuration change reason code"
              aria-invalid={reason.trim().length > 0 && !reasonReady}
              aria-describedby="runtime-reason-help"
              placeholder="required_reason_code"
              value={reason}
              onChange={(event) => setReason(event.target.value)}
            />
            <p
              id="runtime-reason-help"
              role={reason.trim().length > 0 && !reasonReady ? "alert" : undefined}
              className={`text-xs ${reason.trim().length > 0 && !reasonReady ? "text-destructive" : "text-muted-foreground"}`}
            >
              {reason.trim().length > 0 && reasonIssue
                ? reasonIssue
                : "Symbolic reason code: lowercase letters, digits, underscore, hyphen and dot, up to 128 characters."}
            </p>
            {reasonSuggestion && (
              <Button
                size="sm"
                variant="ghost"
                className="justify-self-start"
                onClick={() => setReason(reasonSuggestion)}
              >
                Use suggested code {reasonSuggestion}
              </Button>
            )}
          </div>
          <Button size="sm" disabled={busy || !active || !model || !reasonReady} onClick={createVersion}>Create inactive version</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Opaque subscription attachment</CardTitle>
          <CardDescription>No account, filesystem, credential filename, or credential value is exposed.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-2 sm:flex-row">
          <select aria-label="Available credential home" className={selectClass} value={homeRef} onChange={(event) => setHomeRef(event.target.value)}>
            <option value="">Select healthy unassigned home</option>
            {availableHomes.map((home) => <option key={home.home_ref} value={home.home_ref}>{home.provider} · {home.home_ref}</option>)}
          </select>
          <Button
            size="sm"
            variant="outline"
            disabled={busy || !homeRef || !generationReady}
            onClick={() => {
              const home = availableHomes.find((item) => item.home_ref === homeRef);
              if (!home) return;
              void perform("Opaque subscription attached.", () => api.assignRuntimeCredentialHome(workspaceId, detail.id, {
                home_ref: home.home_ref,
                expected_binding_generation: bindingGeneration,
                expected_catalog_generation: home.generation,
              }));
            }}
          >Attach</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Validate, diff, activate, or roll back</CardTitle>
          <CardDescription>Versions are immutable and activation uses compare-and-swap expectations.</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3">
          <select aria-label="Candidate configuration version" className={selectClass} value={candidateVersionId} onChange={(event) => setCandidateVersionId(event.target.value)}>
            <option value="">Select candidate version</option>
            {detail.versions.map((version) => <option key={version.id} value={version.id}>{version.id} · {version.state} · {version.apply_class}</option>)}
          </select>
          {candidate && (
            <div className="grid gap-2 rounded-lg border p-3 text-xs">
              <div className="flex flex-wrap gap-2">
                <Badge variant="outline">{candidate.apply_class}</Badge>
                <Badge variant={validationLabel === "valid" ? "secondary" : "outline"}>{validationLabel}</Badge>
              </div>
              {diff.length === 0 ? <p className="text-muted-foreground">No safe configuration differences.</p> : diff.map((item) => (
                <p key={item.field}><span className="font-medium">{item.field.replaceAll("_", " ")}</span>: <code>{item.before}</code> → <code>{item.after}</code></p>
              ))}
              <p className="text-muted-foreground">Hot/restart class is the server projection. Activation does not claim readiness; wait for runtime acknowledgement.</p>
            </div>
          )}
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="outline" disabled={busy || !candidateVersionId} onClick={() => perform("Version validation requested.", () => api.validateRuntimeBindingConfigurationVersion(workspaceId, detail.id, candidateVersionId))}>Validate</Button>
            <Button size="sm" disabled={busy || !candidateVersionId || !reasonReady || !generationReady} onClick={() => perform("Version activation requested; awaiting runtime acknowledgement.", () => api.activateRuntimeBindingConfigurationVersion(workspaceId, detail.id, candidateVersionId, { expected_active_version_id: activeVersionId, expected_binding_generation: bindingGeneration, reason: reason.trim() }))}>Activate with CAS</Button>
          </div>
          <div className="flex flex-col gap-2 border-t pt-3 sm:flex-row">
            <select aria-label="Rollback target version" className={selectClass} value={rollbackVersionId} onChange={(event) => setRollbackVersionId(event.target.value)}>
              <option value="">Select prior immutable version</option>
              {detail.versions.filter((version) => version.id !== activeVersionId).map((version) => <option key={version.id} value={version.id}>{version.id}</option>)}
            </select>
            <Button size="sm" variant="destructive" disabled={busy || !rollbackVersionId || !reasonReady || !generationReady} onClick={() => perform("Rollback activation requested.", () => api.rollbackRuntimeBindingConfiguration(workspaceId, detail.id, { target_version_id: rollbackVersionId, expected_active_version_id: activeVersionId, expected_binding_generation: bindingGeneration, reason: reason.trim() }))}>Roll back</Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
