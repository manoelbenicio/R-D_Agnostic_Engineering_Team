import React, { useEffect, useMemo } from 'react';
import Editor from '@monaco-editor/react';
import { FormField, StatusBadge } from '@/design-system';
import { PROVIDERS_REGISTRY, ProviderType as KeyProviderType } from '@/api/key-store/registry';
import { useKeyStore } from '@/api/key-store/store';
import { useSessionStore } from '@/api/session-store';
import { CanvasNode, ProviderType } from '@/shared/canvas-types';
import { CanvasProviderOption, findSourceProvider } from './provider-options';
import { ROLE_TEMPLATES, StarterRole, roleFromValue, AGENT_COLOR_PALETTE } from './role-templates';
import { ALLOWED_TOOLS, HIGH_PRIVILEGE_TOOLS, unknownTools } from './allowed-tools';

export interface BlockConfigurationPanelProps {
  node?: CanvasNode;
  providerOptions: CanvasProviderOption[];
  onUpdateNode: (nodeId: string, updater: (node: CanvasNode) => CanvasNode) => void;
}

export const BlockConfigurationPanel: React.FC<BlockConfigurationPanelProps> = ({
  node,
  providerOptions,
  onUpdateNode,
}) => {
  const cachedModels = useKeyStore((state) => state.cachedModels);
  const { sessions, refresh: refreshSessions } = useSessionStore();
  const providerSessions = sessions.filter((session) => session.cli_provider === (node?.data.provider ?? ''));
  const modelOptions = useMemo(() => {
    if (!node?.data.provider) return [];
    const sourceProvider = findSourceProvider(node.data.provider, providerOptions);
    if (!sourceProvider) return [];
    return cachedModels[sourceProvider] ?? [];
  }, [cachedModels, node?.data.provider, providerOptions]);

  useEffect(() => {
    if (sessions.length === 0) void refreshSessions();
  }, [refreshSessions, sessions.length]);

  if (!node) {
    return (
      <aside className="canvas-config-panel">
        <div className="canvas-panel-title">Block Configuration</div>
        <div className="canvas-empty-config">Select an agent block to edit its runtime profile.</div>
      </aside>
    );
  }

  const patchData = (patch: Partial<CanvasNode['data']>) => {
    onUpdateNode(node.id, (current) => ({
      ...current,
      data: {
        ...current.data,
        ...patch,
      },
    }));
  };

  const toggleTool = (toolId: string) => {
    const current = node.data.allowedTools ?? [];
    const next = current.includes(toolId)
      ? current.filter((tool) => tool !== toolId)
      : [...current, toolId];
    patchData({ allowedTools: next });
  };

  const unknownSelected = unknownTools(node.data.allowedTools ?? []);

  const handleRoleChange = (roleValue: string) => {
    const role = roleFromValue(roleValue);
    const template = ROLE_TEMPLATES[role];
    patchData({
      role,
      display_name: template.display_name,
      profile_name: `${template.profile_prefix}-${node.id.slice(0, 8)}`,
      system_prompt: template.system_prompt,
      allowedTools: [...template.allowedTools],
    });
  };

  const handleProviderChange = (providerValue: string) => {
    patchData({
      provider: providerValue ? (providerValue as ProviderType) : undefined,
      model: undefined,
      session_id: undefined,
    });
  };

  const handleEntryPointChange = (checked: boolean) => {
    onUpdateNode(node.id, (current) => ({
      ...current,
      data: {
        ...current.data,
        is_entry_point: checked,
      },
    }));
  };

  const currentSourceProvider = findSourceProvider(node.data.provider, providerOptions);

  return (
    <aside className="canvas-config-panel" aria-label="Block configuration">
      <div className="canvas-panel-title">Block Configuration</div>
      <div className="canvas-config-id">
        <span>{node.id}</span>
        <StatusBadge
          status={node.data.is_entry_point ? 'completed' : 'idle'}
          label={node.data.is_entry_point ? 'Entry point' : 'Draft'}
        />
      </div>

      <FormField label="Display name" id="node-display-name">
        <input
          id="node-display-name"
          value={node.data.display_name}
          onChange={(event) => patchData({ display_name: event.target.value })}
        />
      </FormField>

      <FormField label="Role" id="node-role">
        <select id="node-role" value={roleFromValue(node.data.role)} onChange={(event) => handleRoleChange(event.target.value)}>
          {Object.keys(ROLE_TEMPLATES).map((role) => (
            <option key={role} value={role}>
              {ROLE_TEMPLATES[role as StarterRole].display_name}
            </option>
          ))}
        </select>
      </FormField>

      <label className="canvas-checkbox-row">
        <input
          type="checkbox"
          checked={node.data.is_entry_point}
          onChange={(event) => handleEntryPointChange(event.target.checked)}
        />
        <span>Entry point</span>
      </label>

      <FormField id="block-color" label="Agent Color">
        <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap', alignItems: 'center' }}>
          {AGENT_COLOR_PALETTE.map((c) => (
            <button
              key={c}
              type="button"
              onClick={() => patchData({ color: c })}
              title={c}
              aria-label={`Set agent color to ${c}`}
              style={{
                width: 24, height: 24,
                borderRadius: '50%',
                border: node.data.color === c ? '2px solid #fff' : '2px solid transparent',
                background: c,
                cursor: 'pointer',
                transition: 'border-color 0.15s ease, transform 0.1s ease',
                transform: node.data.color === c ? 'scale(1.2)' : 'scale(1)',
              }}
            />
          ))}
          <input
            type="color"
            value={node.data.color || '#00b0bd'}
            onChange={(e) => patchData({ color: e.target.value })}
            title="Custom color"
            aria-label="Custom agent color"
            style={{ width: 24, height: 24, padding: 0, border: 'none', cursor: 'pointer', borderRadius: '50%' }}
          />
        </div>
      </FormField>

      <FormField label="Provider" id="node-provider">
        <select id="node-provider" value={node.data.provider ?? ''} onChange={(event) => handleProviderChange(event.target.value)}>
          <option value="">No provider selected</option>
          {providerOptions.map((option) => (
            <option key={`${option.sourceProvider}:${option.provider}`} value={option.provider}>
              {option.label}
            </option>
          ))}
        </select>
      </FormField>

      <FormField
        label="Auth Session"
        id="block-session"
        helperText={providerSessions.length === 0 && node.data.provider ? 'No sessions - visit Sessions page' : undefined}
      >
        <select
          id="block-session"
          value={node.data.session_id ?? ''}
          disabled={!node.data.provider}
          onChange={(event) => patchData({ session_id: event.target.value || undefined })}
        >
          <option value="">Auto (default session)</option>
          {providerSessions.map((session) => (
            <option key={session.id} value={session.id}>
              {sessionOptionLabel(session.status)} {session.account_email}
            </option>
          ))}
        </select>
      </FormField>

      <FormField label="Model" id="node-model" helperText={modelHelperText(currentSourceProvider)}>
        <select
          id="node-model"
          value={node.data.model ?? ''}
          disabled={!node.data.provider}
          onChange={(event) => patchData({ model: event.target.value || undefined })}
        >
          <option value="">Select a model</option>
          {modelOptions.map((model) => (
            <option key={model} value={model}>
              {model}
            </option>
          ))}
        </select>
      </FormField>

      <FormField label="Allowed tools" id="node-allowed-tools">
        <fieldset id="node-allowed-tools" className="canvas-tools-fieldset">
          {ALLOWED_TOOLS.map((tool) => {
            const selected = (node.data.allowedTools ?? []).includes(tool.id);
            const highPrivilege = HIGH_PRIVILEGE_TOOLS.includes(tool.id);
            return (
              <label
                key={tool.id}
                className={`canvas-tool-option${highPrivilege ? ' canvas-tool-option--privileged' : ''}`}
                title={tool.description}
              >
                <input
                  type="checkbox"
                  checked={selected}
                  onChange={() => toggleTool(tool.id)}
                />
                <span className="canvas-tool-name">{tool.label}</span>
                <span className="canvas-tool-desc">{tool.description}</span>
              </label>
            );
          })}
          {unknownSelected.length > 0 ? (
            <div className="canvas-tools-warning" role="alert">
              <span>Unknown tools on this node: {unknownSelected.join(', ')}. Uncheck to remove.</span>
              {unknownSelected.map((tool) => (
                <label key={tool} className="canvas-tool-option canvas-tool-option--unknown">
                  <input type="checkbox" checked onChange={() => toggleTool(tool)} />
                  <span className="canvas-tool-name">{tool}</span>
                </label>
              ))}
            </div>
          ) : null}
        </fieldset>
      </FormField>

      <FormField label="System prompt" id="node-system-prompt">
        <div className="canvas-monaco-shell">
          <Editor
            height="240px"
            defaultLanguage="markdown"
            theme="vs-dark"
            value={node.data.system_prompt}
            onChange={(value) => patchData({ system_prompt: value ?? '' })}
            options={{
              minimap: { enabled: false },
              fontFamily: 'JetBrains Mono, monospace',
              fontSize: 13,
              wordWrap: 'on',
              scrollBeyondLastLine: false,
            }}
          />
        </div>
      </FormField>
    </aside>
  );
};

function modelHelperText(sourceProvider?: KeyProviderType): string {
  if (!sourceProvider) return 'Choose a validated provider before selecting a model.';
  const label = PROVIDERS_REGISTRY.find((provider) => provider.id === sourceProvider)?.label ?? sourceProvider;
  return `Models listed from the validated ${label} response. No default is selected.`;
}

function sessionOptionLabel(status: 'active' | 'expiring' | 'expired'): string {
  if (status === 'active') return '[active]';
  if (status === 'expiring') return '[expiring]';
  return '[expired]';
}

export default BlockConfigurationPanel;
