/* eslint-disable agentverse/no-sideways-capability-imports */
import React, { FormEvent, useMemo, useState } from 'react';
import Editor from '@monaco-editor/react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Badge, Button, Card, FormField, Modal, StatusBadge } from '@/design-system';
import { goCoreClient, goCoreQueryKeys, Flow } from '@/api';
import { useToast } from '@/shell/toasts';
import { useValidatedProviders } from '@/api/key-store/use-validated-providers';
import {
  DEFAULT_SCHEDULE_DRAFT,
  ScheduleDraft,
  SchedulePreset,
  scheduleDraftToCron,
  validateCron,
} from './flow-schedule';
import './flows.css';

type EditorMode = 'closed' | 'create' | 'edit';

const emptyFlow: Flow = {
  name: '',
  schedule: scheduleDraftToCron(DEFAULT_SCHEDULE_DRAFT),
  agent_profile: '',
  provider: '',
  prompt_template: '',
  enabled: true,
  gating_script: null,
};

export const FlowsPage: React.FC = () => {
  const toast = useToast();
  const queryClient = useQueryClient();
  const validatedProviders = useValidatedProviders();
  const [search, setSearch] = useState('');
  const [editorMode, setEditorMode] = useState<EditorMode>('closed');
  const [form, setForm] = useState<Flow>(emptyFlow);
  const [scheduleDraft, setScheduleDraft] = useState<ScheduleDraft>(DEFAULT_SCHEDULE_DRAFT);
  const scheduleValidation = useMemo(() => validateCron(form.schedule), [form.schedule]);

  const flowsQuery = useQuery({
    queryKey: goCoreQueryKeys.flows(),
    queryFn: () => goCoreClient.listFlows(),
    refetchInterval: 15_000,
    refetchIntervalInBackground: false,
  });

  const profilesQuery = useQuery({
    queryKey: goCoreQueryKeys.profiles(),
    queryFn: () => goCoreClient.listProfiles(),
  });

  const providerOptions = useMemo(
    () => validatedProviders.map((provider) => ({ provider, label: provider })),
    [validatedProviders]
  );

  const filteredFlows = useMemo(() => {
    const term = search.trim().toLowerCase();
    return (flowsQuery.data ?? []).filter((flow) => {
      if (!term) return true;
      return (
        flow.name.toLowerCase().includes(term) ||
        flow.agent_profile.toLowerCase().includes(term) ||
        String(flow.provider).toLowerCase().includes(term)
      );
    });
  }, [flowsQuery.data, search]);

  const saveMutation = useMutation({
    mutationFn: (flow: Flow) => goCoreClient.createFlow(flow),
    onSuccess: async (flow) => {
      await queryClient.invalidateQueries({ queryKey: goCoreQueryKeys.flows() });
      setEditorMode('closed');
      toast.success(`Saved flow ${flow.name}.`);
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : String(error));
    },
  });

  const runMutation = useMutation({
    mutationFn: (name: string) => goCoreClient.runFlow(name),
    onSuccess: (_, name) => toast.success(`Run started for ${name}.`),
    onError: (error) => toast.error(error instanceof Error ? error.message : String(error)),
  });

  const toggleMutation = useMutation({
    mutationFn: ({ name, enabled }: { name: string; enabled: boolean }) =>
      enabled ? goCoreClient.enableFlow(name) : goCoreClient.disableFlow(name),
    onMutate: async ({ name, enabled }) => {
      await queryClient.cancelQueries({ queryKey: goCoreQueryKeys.flows() });
      const previous = queryClient.getQueryData<Flow[]>(goCoreQueryKeys.flows());
      queryClient.setQueryData<Flow[]>(goCoreQueryKeys.flows(), (current = []) =>
        current.map((flow) => (flow.name === name ? { ...flow, enabled } : flow))
      );
      return { previous };
    },
    onError: (error, _variables, context) => {
      if (context?.previous) {
        queryClient.setQueryData(goCoreQueryKeys.flows(), context.previous);
      }
      toast.error(error instanceof Error ? error.message : String(error));
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: goCoreQueryKeys.flows() }),
  });

  const openCreate = () => {
    setScheduleDraft(DEFAULT_SCHEDULE_DRAFT);
    setForm({
      ...emptyFlow,
      schedule: scheduleDraftToCron(DEFAULT_SCHEDULE_DRAFT),
      agent_profile: profilesQuery.data?.[0]?.name ?? '',
      provider: providerOptions[0]?.provider ?? '',
    });
    setEditorMode('create');
  };

  const openEdit = (flow: Flow) => {
    setForm(flow);
    setScheduleDraft(DEFAULT_SCHEDULE_DRAFT);
    setEditorMode('edit');
  };

  const applyQuickPick = (patch: Partial<ScheduleDraft>) => {
    const nextDraft = { ...scheduleDraft, ...patch };
    setScheduleDraft(nextDraft);
    setForm((current) => ({ ...current, schedule: scheduleDraftToCron(nextDraft) }));
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const validation = validateCron(form.schedule);
    if (!validation.ok) return;
    saveMutation.mutate(form);
  };

  return (
    <main className="flows-page">
      <header className="flows-header">
        <div>
          <h1>Flows</h1>
          <p>Schedule recurring GO Core profile runs with validated providers and cron previews.</p>
        </div>
        <Button variant="primary" onClick={openCreate}>
          New Flow
        </Button>
      </header>

      <Card className="flows-toolbar">
        <FormField label="Search" id="flows-search">
          <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Name, profile, provider" />
        </FormField>
        {flowsQuery.isFetching ? <Badge variant="processing">Refreshing</Badge> : <Badge variant="completed">Live</Badge>}
      </Card>

      <section className="flows-grid">
        {flowsQuery.error ? (
          <Card glow="red">Unable to load flows.</Card>
        ) : flowsQuery.isLoading ? (
          <Card>Loading flows...</Card>
        ) : filteredFlows.length === 0 ? (
          <Card>No flows match the current filters.</Card>
        ) : (
          filteredFlows.map((flow) => {
            const description = validateCron(flow.schedule);
            return (
              <Card key={flow.name} className="flow-card">
                <div className="flow-card-header">
                  <div>
                    <h2>{flow.name}</h2>
                    <p>{description.ok ? description.description : flow.schedule}</p>
                  </div>
                  <StatusBadge status={flow.enabled ? 'completed' : 'idle'} label={flow.enabled ? 'Enabled' : 'Disabled'} />
                </div>
                <div className="flow-meta">
                  <Badge>{flow.agent_profile}</Badge>
                  <Badge>{flow.provider}</Badge>
                  {flow.gating_script ? (
                    <Badge variant="waiting_user_answer" title={flow.gating_script}>
                      Conditional
                    </Badge>
                  ) : null}
                </div>
                <dl className="flow-run-meta">
                  <div>
                    <dt>Last</dt>
                    <dd>{flow.last_run ?? '-'}</dd>
                  </div>
                  <div>
                    <dt>Next</dt>
                    <dd>{flow.next_run ?? '-'}</dd>
                  </div>
                </dl>
                <div className="flow-actions">
                  <Button variant="secondary" onClick={() => openEdit(flow)}>
                    Edit
                  </Button>
                  <Button variant="secondary" onClick={() => runMutation.mutate(flow.name)}>
                    Run Now
                  </Button>
                  <label className="flow-toggle">
                    <input
                      type="checkbox"
                      checked={flow.enabled}
                      onChange={(event) => toggleMutation.mutate({ name: flow.name, enabled: event.target.checked })}
                    />
                    <span>{flow.enabled ? 'Disable' : 'Enable'}</span>
                  </label>
                </div>
              </Card>
            );
          })
        )}
      </section>

      <Modal
        isOpen={editorMode !== 'closed'}
        onClose={() => setEditorMode('closed')}
        title={editorMode === 'edit' ? 'Edit Flow' : 'Create Flow'}
        actions={null}
      >
        <form className="flow-editor" onSubmit={handleSubmit}>
          <div className="flow-editor-grid">
            <FormField label="Name" id="flow-name" required>
              <input
                id="flow-name"
                value={form.name}
                onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
                disabled={editorMode === 'edit'}
              />
            </FormField>
            <FormField label="Agent Profile" id="flow-agent-profile" required>
              <select
                id="flow-agent-profile"
                value={form.agent_profile}
                onChange={(event) => setForm((current) => ({ ...current, agent_profile: event.target.value }))}
              >
                <option value="">Select profile</option>
                {(profilesQuery.data ?? []).map((profile) => (
                  <option key={profile.name} value={profile.name}>
                    {profile.name}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="Provider" id="flow-provider" required helperText="Gated to validated providers.">
              <select
                id="flow-provider"
                value={form.provider}
                onChange={(event) => setForm((current) => ({ ...current, provider: event.target.value }))}
              >
                <option value="">Select provider</option>
                {form.provider && !providerOptions.some((option) => option.provider === form.provider) ? (
                  <option value={form.provider}>{form.provider}</option>
                ) : null}
                {providerOptions.map((option) => (
                  <option key={option.provider} value={option.provider}>
                    {option.label}
                  </option>
                ))}
              </select>
            </FormField>
            <label className="flow-enabled-toggle">
              <input
                type="checkbox"
                checked={form.enabled}
                onChange={(event) => setForm((current) => ({ ...current, enabled: event.target.checked }))}
              />
              Enabled
            </label>
          </div>

          <fieldset className="flow-schedule-picker">
            <legend>Schedule</legend>
            <div className="flow-schedule-row">
              <select
                aria-label="Schedule preset"
                value={scheduleDraft.preset}
                onChange={(event) => applyQuickPick({ preset: event.target.value as SchedulePreset })}
              >
                <option value="every-n-minutes">Every N minutes</option>
                <option value="hourly">Hourly</option>
                <option value="daily-at-time">Daily at time</option>
                <option value="weekdays-at-time">Weekdays at time</option>
                <option value="weekly">Weekly</option>
              </select>
              {scheduleDraft.preset === 'every-n-minutes' ? (
                <input
                  aria-label="Every minutes"
                  type="number"
                  min="1"
                  max="59"
                  value={scheduleDraft.everyMinutes}
                  onChange={(event) => applyQuickPick({ everyMinutes: Number(event.target.value) })}
                />
              ) : null}
              {scheduleDraft.preset !== 'every-n-minutes' ? (
                <>
                  <input
                    aria-label="Schedule hour"
                    type="number"
                    min="0"
                    max="23"
                    value={scheduleDraft.hour}
                    onChange={(event) => applyQuickPick({ hour: event.target.value })}
                  />
                  <input
                    aria-label="Schedule minute"
                    type="number"
                    min="0"
                    max="59"
                    value={scheduleDraft.minute}
                    onChange={(event) => applyQuickPick({ minute: event.target.value })}
                  />
                </>
              ) : null}
              {scheduleDraft.preset === 'weekly' ? (
                <select
                  aria-label="Schedule weekday"
                  value={scheduleDraft.weekday}
                  onChange={(event) => applyQuickPick({ weekday: event.target.value })}
                >
                  <option value="0">Sunday</option>
                  <option value="1">Monday</option>
                  <option value="2">Tuesday</option>
                  <option value="3">Wednesday</option>
                  <option value="4">Thursday</option>
                  <option value="5">Friday</option>
                  <option value="6">Saturday</option>
                </select>
              ) : null}
            </div>
          </fieldset>

          <FormField
            label="Raw cron"
            id="flow-schedule"
            required
            helperText={scheduleValidation.ok ? scheduleValidation.description : scheduleValidation.error}
          >
            <input
              id="flow-schedule"
              value={form.schedule}
              onChange={(event) => setForm((current) => ({ ...current, schedule: event.target.value }))}
              aria-invalid={!scheduleValidation.ok}
            />
          </FormField>
          {!scheduleValidation.ok ? <div className="flow-error">{scheduleValidation.error}</div> : null}

          <FormField label="Gating script" id="flow-gating-script">
            <textarea
              id="flow-gating-script"
              value={form.gating_script ?? ''}
              onChange={(event) => setForm((current) => ({ ...current, gating_script: event.target.value || null }))}
            />
          </FormField>

          <div className="flow-monaco-shell">
            <Editor
              height="260px"
              defaultLanguage="markdown"
              theme="vs-dark"
              value={form.prompt_template}
              onChange={(value) => setForm((current) => ({ ...current, prompt_template: value ?? '' }))}
              options={{
                minimap: { enabled: false },
                fontFamily: 'JetBrains Mono, monospace',
                fontSize: 13,
                wordWrap: 'on',
                scrollBeyondLastLine: false,
              }}
            />
          </div>

          <div className="flow-editor-actions">
            <Button type="button" variant="secondary" onClick={() => setEditorMode('closed')}>
              Cancel
            </Button>
            <Button
              type="submit"
              variant="primary"
              disabled={
                saveMutation.isPending ||
                !form.name ||
                !form.agent_profile ||
                !form.provider ||
                !form.prompt_template ||
                !scheduleValidation.ok
              }
            >
              Save Flow
            </Button>
          </div>
        </form>
      </Modal>
    </main>
  );
};

export default FlowsPage;
