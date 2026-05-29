import React, { useMemo } from 'react';
import Editor from '@monaco-editor/react';
import { FormField, StatusBadge } from '@/design-system';
import { PROVIDERS_REGISTRY, ProviderType as KeyProviderType } from '@/api/key-store/registry';
import { useKeyStore } from '@/api/key-store/store';
import { CanvasNode, ProviderType } from '@/shared/canvas-types';
import { CanvasProviderOption, findSourceProvider } from './provider-options';
import { ROLE_TEMPLATES, StarterRole, roleFromValue } from './role-templates';

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
  const modelOptions = useMemo(() => {
    if (!node?.data.provider) return [];
    const sourceProvider = findSourceProvider(node.data.provider, providerOptions);
    if (!sourceProvider) return [];
    return cachedModels[sourceProvider] ?? [];
  }, [cachedModels, node?.data.provider, providerOptions]);

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
        <input
          id="node-allowed-tools"
          value={(node.data.allowedTools ?? []).join(', ')}
          onChange={(event) =>
            patchData({
              allowedTools: event.target.value
                .split(',')
                .map((item) => item.trim())
                .filter(Boolean),
            })
          }
        />
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

export default BlockConfigurationPanel;
