/* eslint-disable agentverse/no-sideways-capability-imports */
import React, { useEffect, useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useKeyStore } from '@/api/key-store/store';
import { useSettingsStore } from './settings-store';
import { PROVIDERS_REGISTRY, ProviderType } from '@/api/key-store/registry';
import { useToast } from '@/shell/toasts';
import { Card } from '@/design-system/components/Card';
import { Button } from '@/design-system/components/Button';
import { FormField } from '@/design-system/components/FormField';
import { StatusBadge } from '@/design-system/components/StatusBadge';
import { Badge } from '@/design-system/components/Badge';
import { useValidatedProviders } from '@/api/key-store/use-validated-providers';

import {
  validateOpenAI,
  validateAnthropic,
  validateGoogle,
  validateAWS,
  validateAzure,
  validateMoonshot,
  validateCopilot,
  validateOpenCode,
} from '@/api/key-store/validators';

// Helper validator dispatcher
async function validateProviderKeys(providerId: ProviderType, keys: Record<string, string>) {
  switch (providerId) {
    case 'openai':
      return validateOpenAI(keys.apiKey || '');
    case 'anthropic':
      return validateAnthropic(keys.apiKey || '');
    case 'google':
      return validateGoogle(keys.apiKey || '');
    case 'aws':
      return validateAWS(keys.accessKeyId || '', keys.secretAccessKey || '');
    case 'azure':
      return validateAzure(keys.endpoint || '', keys.apiKey || '');
    case 'moonshot':
      return validateMoonshot(keys.apiKey || '');
    case 'copilot':
      return validateCopilot(keys.apiKey || '');
    case 'opencode':
      return validateOpenCode(keys.endpoint || '', keys.apiKey || '');
    default:
      return { ok: false, error: 'Unknown provider' };
  }
}

// Navigation Subheader for Settings
const SettingsNav: React.FC = () => {
  const location = useLocation();
  const currentPath = location.pathname;

  const linkStyle = (path: string): React.CSSProperties => {
    const active = currentPath.endsWith(path);
    return {
      fontFamily: 'var(--font-mono)',
      fontSize: '0.875rem',
      fontWeight: 600,
      color: active ? 'var(--cyan)' : 'var(--text-muted)',
      borderBottom: active ? '2px solid var(--cyan)' : '2px solid transparent',
      padding: 'var(--space-2) var(--space-4)',
      textDecoration: 'none',
      transition: 'all 0.15s ease-in-out',
    };
  };

  return (
    <div
      style={{
        display: 'flex',
        gap: 'var(--space-2)',
        borderBottom: '1px solid var(--border)',
        marginBottom: 'var(--space-5)',
      }}
    >
      <Link to="/settings/providers" style={linkStyle('providers')}>
        Provider Keys
      </Link>
      <Link to="/settings/general" style={linkStyle('general')}>
        General
      </Link>
      <Link to="/settings/appearance" style={linkStyle('appearance')}>
        Appearance
      </Link>
      <Link to="/settings/sharepoint" style={linkStyle('sharepoint')}>
        SharePoint Assessment
      </Link>
    </div>
  );
};

const containerStyle: React.CSSProperties = {
  maxWidth: '1200px',
  margin: '0 auto',
  padding: 'var(--space-5)',
  fontFamily: 'var(--font-body)',
  color: 'var(--text-primary)',
};

const titleStyle: React.CSSProperties = {
  fontFamily: 'var(--font-display)',
  fontSize: '2rem',
  fontWeight: 700,
  marginBottom: 'var(--space-2)',
  color: 'var(--text-primary)',
};

const descStyle: React.CSSProperties = {
  fontSize: '0.875rem',
  color: 'var(--text-muted)',
  marginBottom: 'var(--space-5)',
};

// 1. ProvidersPage component
export const ProvidersPage: React.FC = () => {
  const toast = useToast();
  const initKeys = useKeyStore((s) => s.init);
  const keysInitialized = useKeyStore((s) => s.initialized);
  const statuses = useKeyStore((s) => s.statuses);
  const cachedModels = useKeyStore((s) => s.cachedModels);
  const maskedKeys = useKeyStore((s) => s.maskedKeys);
  const setKey = useKeyStore((s) => s.setKey);
  const removeKey = useKeyStore((s) => s.removeKey);
  const setInvalid = useKeyStore((s) => s.setInvalid);

  const [formKeys, setFormKeys] = useState<Record<string, Record<string, string>>>({});
  const [validating, setValidating] = useState<string | null>(null);

  useEffect(() => {
    if (!keysInitialized) {
      void initKeys();
    }
  }, [initKeys, keysInitialized]);

  const handleInputChange = (providerId: string, fieldName: string, value: string) => {
    setFormKeys((prev) => ({
      ...prev,
      [providerId]: {
        ...(prev[providerId] || {}),
        [fieldName]: value,
      },
    }));
  };

  const handleValidateAndSave = async (providerId: ProviderType) => {
    const keys = formKeys[providerId] || {};
    const providerDef = PROVIDERS_REGISTRY.find((p) => p.id === providerId);
    if (!providerDef) return;

    for (const field of providerDef.fields) {
      if (!keys[field.name]) {
        toast.error(`Please enter ${field.label}`);
        return;
      }
    }

    setValidating(providerId);
    try {
      const res = await validateProviderKeys(providerId, keys);
      if (res.ok) {
        await setKey(providerId, keys, res.models || []);
        toast.success(`Successfully validated and saved ${providerDef.label}!`);
        setFormKeys((prev) => {
          const next = { ...prev };
          delete next[providerId];
          return next;
        });
      } else {
        setInvalid(providerId);
        toast.error(res.error || `Validation failed for ${providerDef.label}`);
      }
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : String(err);
      setInvalid(providerId);
      toast.error(`Unexpected validation failure: ${errMsg}`);
    } finally {
      setValidating(null);
    }
  };

  const handleRemove = async (providerId: ProviderType) => {
    const providerDef = PROVIDERS_REGISTRY.find((p) => p.id === providerId);
    try {
      await removeKey(providerId);
      toast.success(`Removed configuration for ${providerDef?.label || providerId}`);
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : String(err);
      toast.error(`Failed to remove key: ${errMsg}`);
    }
  };

  return (
    <div style={containerStyle}>
      <h1 style={titleStyle}>API Key Management</h1>
      <p style={descStyle}>
        Manage your API keys for model providers. Keys are stored locally in plaintext IndexedDB and never sent to telemetry.
      </p>
      <SettingsNav />

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))',
          gap: 'var(--space-4)',
        }}
      >
        {PROVIDERS_REGISTRY.map((provider) => {
          const status = statuses[provider.id] || 'unset';
          const isSet = status === 'set';
          const isValidating = validating === provider.id;
          const currentValues = formKeys[provider.id] || {};
          const modelsList = cachedModels[provider.id] || [];

          const badgeStatus =
            status === 'set'
              ? 'completed'
              : status === 'invalid'
              ? 'error'
              : 'idle';
          const badgeLabel =
            status === 'set'
              ? 'Validated'
              : status === 'invalid'
              ? 'Invalid'
              : 'Unconfigured';

          return (
            <Card
              key={provider.id}
              glow={status === 'invalid' ? 'red' : 'none'}
              style={{
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'space-between',
                minHeight: '260px',
              }}
            >
              <div>
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    marginBottom: 'var(--space-4)',
                  }}
                >
                  <h3
                    style={{
                      fontFamily: 'var(--font-display)',
                      fontSize: '1.125rem',
                      fontWeight: 600,
                      margin: 0,
                    }}
                  >
                    {provider.label}
                  </h3>
                  <StatusBadge status={badgeStatus} label={badgeLabel} />
                </div>

                {isSet ? (
                  <div style={{ marginBottom: 'var(--space-4)' }}>
                    {provider.fields.map((field) => {
                      const maskedVal = maskedKeys[provider.id]?.[field.name] || '••••••••';
                      return (
                        <div key={field.name} style={{ marginBottom: 'var(--space-3)' }}>
                          <div
                            style={{
                              fontSize: '0.75rem',
                              fontFamily: 'var(--font-mono)',
                              color: 'var(--text-muted)',
                              textTransform: 'uppercase',
                              marginBottom: 'var(--space-1)',
                            }}
                          >
                            {field.label}
                          </div>
                          <div
                            style={{
                              fontFamily: 'var(--font-mono)',
                              fontSize: '0.875rem',
                              background: 'rgba(0, 0, 0, 0.2)',
                              padding: 'var(--space-2) var(--space-3)',
                              borderRadius: 'var(--radius-button)',
                              border: '1px solid var(--border)',
                              color: 'var(--text-dim)',
                            }}
                          >
                            {maskedVal}
                          </div>
                        </div>
                      );
                    })}

                    <div style={{ marginTop: 'var(--space-4)' }}>
                      <div
                        style={{
                          fontSize: '0.75rem',
                          fontFamily: 'var(--font-mono)',
                          color: 'var(--text-muted)',
                          textTransform: 'uppercase',
                          marginBottom: 'var(--space-2)',
                        }}
                      >
                        Available Models
                      </div>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
                        {modelsList.map((model) => (
                          <Badge key={model} variant="idle" style={{ fontSize: '0.675rem' }}>
                            {model}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  </div>
                ) : (
                  <div style={{ marginBottom: 'var(--space-4)' }}>
                    {provider.fields.map((field) => (
                      <FormField
                        key={field.name}
                        label={field.label}
                        id={`${provider.id}-${field.name}`}
                      >
                        <input
                          type={field.type}
                          value={currentValues[field.name] || ''}
                          disabled={isValidating}
                          onChange={(e) =>
                            handleInputChange(provider.id, field.name, e.target.value)
                          }
                          placeholder={`Enter ${field.label}`}
                        />
                      </FormField>
                    ))}
                  </div>
                )}
              </div>

              <div style={{ marginTop: 'auto', display: 'flex', justifyContent: 'flex-end', gap: 'var(--space-2)' }}>
                {isSet ? (
                  <Button
                    variant="secondary"
                    onClick={() => handleRemove(provider.id)}
                    style={{ borderColor: 'var(--threat)', color: 'var(--threat)' }}
                  >
                    Remove Configuration
                  </Button>
                ) : (
                  <Button
                    variant="primary"
                    disabled={isValidating}
                    onClick={() => handleValidateAndSave(provider.id)}
                  >
                    {isValidating ? 'Validating...' : 'Validate & Save'}
                  </Button>
                )}
              </div>
            </Card>
          );
        })}
      </div>
    </div>
  );
};

// 2. GeneralPage component
export const GeneralPage: React.FC = () => {
  const initSettings = useSettingsStore((s) => s.init);
  const settingsInitialized = useSettingsStore((s) => s.initialized);
  const caoBaseUrl = useSettingsStore((s) => s.caoBaseUrl);
  const defaultProvider = useSettingsStore((s) => s.defaultProvider);
  const defaultWorkingDir = useSettingsStore((s) => s.defaultWorkingDir);
  const updateSetting = useSettingsStore((s) => s.updateSetting);

  const validatedProviders = useValidatedProviders();

  useEffect(() => {
    if (!settingsInitialized) {
      void initSettings();
    }
  }, [initSettings, settingsInitialized]);

  const handleProviderChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    void updateSetting('defaultProvider', e.target.value);
  };

  const handleWorkingDirChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    void updateSetting('defaultWorkingDir', e.target.value);
  };

  return (
    <div style={containerStyle}>
      <h1 style={titleStyle}>General Settings</h1>
      <p style={descStyle}>Configure core behaviors and default configurations for the AgentVerse environment.</p>
      <SettingsNav />

      <div style={{ maxWidth: '600px', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
        <Card>
          <FormField
            label="Runtime Mode"
            id="general-runtime-mode"
            helperText="Switch between the local Docker runtime and the deployed cloud runtime. The buttons fill the URL below; you can also override it manually."
          >
            <div style={{ display: 'flex', gap: 'var(--space-2)', flexWrap: 'wrap' }}>
              <button
                type="button"
                className="sentinel-button sentinel-button-secondary"
                onClick={() => void updateSetting('caoBaseUrl', 'http://127.0.0.1:9889')}
              >
                Local (Docker)
              </button>
              <button
                type="button"
                className="sentinel-button sentinel-button-secondary"
                onClick={() => {
                  const cloudUrl = (import.meta.env.VITE_CLOUD_RUNTIME_URL as string | undefined) || '';
                  if (cloudUrl) {
                    void updateSetting('caoBaseUrl', cloudUrl);
                  } else {
                    // No cloud URL baked in yet — leave the field unchanged and rely on the user typing it
                    // eslint-disable-next-line no-alert
                    alert('No cloud runtime URL is configured for this build. Run scripts/deploy-cloud.sh first, or paste the URL into the Runtime Base URL field.');
                  }
                }}
              >
                Cloud (Run)
              </button>
            </div>
          </FormField>

          <FormField
            label="Runtime Base URL"
            id="general-cao-base-url"
            helperText="Base URL of the orchestration runtime. Persisted to IndexedDB. Use http://127.0.0.1:9889 for local Docker, or your Cloud Run URL for cloud."
          >
            <input
              type="text"
              value={caoBaseUrl}
              onChange={(e) => void updateSetting('caoBaseUrl', e.target.value)}
              placeholder="e.g. http://127.0.0.1:9889"
            />
          </FormField>

          <FormField
            label="Default Model Provider"
            id="general-default-provider"
            helperText="Select the default LLM provider for agent nodes. Gated strictly to validated providers."
          >
            <select
              value={defaultProvider}
              onChange={handleProviderChange}
            >
              <option value="">None</option>
              {validatedProviders.map((provId) => {
                const label = PROVIDERS_REGISTRY.find((p) => p.id === provId)?.label || provId;
                return (
                  <option key={provId} value={provId} style={{ background: '#1c1c1e' }}>
                    {label}
                  </option>
                );
              })}
            </select>
          </FormField>

          <FormField
            label="Default Working Directory"
            id="general-default-working-dir"
            helperText="Specify the default working directory path for new agents and local processes."
          >
            <input
              type="text"
              value={defaultWorkingDir}
              onChange={handleWorkingDirChange}
              placeholder="e.g. C:/VMs/Projetos/Workspace"
            />
          </FormField>
        </Card>
      </div>
    </div>
  );
};

// 3. AppearancePage component
export const AppearancePage: React.FC = () => {
  const initSettings = useSettingsStore((s) => s.init);
  const settingsInitialized = useSettingsStore((s) => s.initialized);
  const fonts = useSettingsStore((s) => s.fonts);
  const theme = useSettingsStore((s) => s.theme);
  const updateSetting = useSettingsStore((s) => s.updateSetting);

  useEffect(() => {
    if (!settingsInitialized) {
      void initSettings();
    }
  }, [initSettings, settingsInitialized]);

  const standardFonts = ['JetBrains Mono', 'Inter', 'system-ui'];

  const getFontSelectValue = (val: string) => {
    return standardFonts.includes(val) ? val : 'custom';
  };

  const handleSelectChange = (type: 'body' | 'display' | 'mono', selectVal: string) => {
    if (selectVal === 'custom') {
      const defaultCustom = type === 'mono' ? 'Courier New' : 'Segoe UI';
      const nextFonts = { ...fonts, [type]: defaultCustom };
      void updateSetting('fonts', nextFonts);
    } else {
      const nextFonts = { ...fonts, [type]: selectVal };
      void updateSetting('fonts', nextFonts);
    }
  };

  const handleCustomTextChange = (type: 'body' | 'display' | 'mono', val: string) => {
    const nextFonts = { ...fonts, [type]: val };
    void updateSetting('fonts', nextFonts);
  };

  return (
    <div style={containerStyle}>
      <h1 style={titleStyle}>Appearance Settings</h1>
      <p style={descStyle}>Configure system fonts and visual options for the interface.</p>
      <SettingsNav />

      <div style={{ maxWidth: '600px', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
        <Card>
          <h3 style={{ fontFamily: 'var(--font-display)', marginBottom: 'var(--space-4)', fontSize: '1.25rem' }}>
            Typography Settings
          </h3>

          {/* Display Font */}
          <div style={{ marginBottom: 'var(--space-4)' }}>
            <FormField
              label="Display Font Family (--font-display)"
              id="appearance-font-display"
              helperText="Used for page titles, headings, and primary navigation."
            >
              <select
                value={getFontSelectValue(fonts.display)}
                onChange={(e) => handleSelectChange('display', e.target.value)}
              >
                <option value="Inter" style={{ background: '#1c1c1e' }}>Inter</option>
                <option value="JetBrains Mono" style={{ background: '#1c1c1e' }}>JetBrains Mono</option>
                <option value="system-ui" style={{ background: '#1c1c1e' }}>System UI (system-ui)</option>
                <option value="custom" style={{ background: '#1c1c1e' }}>Custom...</option>
              </select>
            </FormField>

            {!standardFonts.includes(fonts.display) && (
              <div style={{ marginTop: 'calc(-1 * var(--space-2))', marginBottom: 'var(--space-4)' }}>
                <FormField label="Custom Display Font Name" id="appearance-custom-display">
                  <input
                    type="text"
                    value={fonts.display}
                    onChange={(e) => handleCustomTextChange('display', e.target.value)}
                    placeholder="e.g. Montserrat, sans-serif"
                  />
                </FormField>
                <div
                  style={{
                    fontSize: '0.75rem',
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--amber)',
                    marginTop: 'calc(-1 * var(--space-2))',
                    padding: 'var(--space-2)',
                    background: 'rgba(255, 183, 0, 0.05)',
                    borderRadius: 'var(--radius-button)',
                    border: '1px solid rgba(255, 183, 0, 0.2)',
                  }}
                >
                  ⚠ Custom fonts require matching system installation or stylesheet injection to render correctly.
                </div>
              </div>
            )}
          </div>

          {/* Body Font */}
          <div style={{ marginBottom: 'var(--space-4)' }}>
            <FormField
              label="Body Font Family (--font-body)"
              id="appearance-font-body"
              helperText="Used for main content paragraphs, lists, and form descriptions."
            >
              <select
                value={getFontSelectValue(fonts.body)}
                onChange={(e) => handleSelectChange('body', e.target.value)}
              >
                <option value="Inter" style={{ background: '#1c1c1e' }}>Inter</option>
                <option value="JetBrains Mono" style={{ background: '#1c1c1e' }}>JetBrains Mono</option>
                <option value="system-ui" style={{ background: '#1c1c1e' }}>System UI (system-ui)</option>
                <option value="custom" style={{ background: '#1c1c1e' }}>Custom...</option>
              </select>
            </FormField>

            {!standardFonts.includes(fonts.body) && (
              <div style={{ marginTop: 'calc(-1 * var(--space-2))', marginBottom: 'var(--space-4)' }}>
                <FormField label="Custom Body Font Name" id="appearance-custom-body">
                  <input
                    type="text"
                    value={fonts.body}
                    onChange={(e) => handleCustomTextChange('body', e.target.value)}
                    placeholder="e.g. Roboto, sans-serif"
                  />
                </FormField>
                <div
                  style={{
                    fontSize: '0.75rem',
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--amber)',
                    marginTop: 'calc(-1 * var(--space-2))',
                    padding: 'var(--space-2)',
                    background: 'rgba(255, 183, 0, 0.05)',
                    borderRadius: 'var(--radius-button)',
                    border: '1px solid rgba(255, 183, 0, 0.2)',
                  }}
                >
                  ⚠ Custom fonts require matching system installation or stylesheet injection to render correctly.
                </div>
              </div>
            )}
          </div>

          {/* Mono Font */}
          <div style={{ marginBottom: 'var(--space-4)' }}>
            <FormField
              label="Monospace Font Family (--font-mono)"
              id="appearance-font-mono"
              helperText="Used for logs, terminal commands, code editors, and data badges."
            >
              <select
                value={getFontSelectValue(fonts.mono)}
                onChange={(e) => handleSelectChange('mono', e.target.value)}
              >
                <option value="JetBrains Mono" style={{ background: '#1c1c1e' }}>JetBrains Mono</option>
                <option value="Inter" style={{ background: '#1c1c1e' }}>Inter</option>
                <option value="system-ui" style={{ background: '#1c1c1e' }}>System UI (system-ui)</option>
                <option value="custom" style={{ background: '#1c1c1e' }}>Custom...</option>
              </select>
            </FormField>

            {!standardFonts.includes(fonts.mono) && (
              <div style={{ marginTop: 'calc(-1 * var(--space-2))', marginBottom: 'var(--space-4)' }}>
                <FormField label="Custom Monospace Font Name" id="appearance-custom-mono">
                  <input
                    type="text"
                    value={fonts.mono}
                    onChange={(e) => handleCustomTextChange('mono', e.target.value)}
                    placeholder="e.g. Consolas, Monaco, monospace"
                  />
                </FormField>
                <div
                  style={{
                    fontSize: '0.75rem',
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--amber)',
                    marginTop: 'calc(-1 * var(--space-2))',
                    padding: 'var(--space-2)',
                    background: 'rgba(255, 183, 0, 0.05)',
                    borderRadius: 'var(--radius-button)',
                    border: '1px solid rgba(255, 183, 0, 0.2)',
                  }}
                >
                  ⚠ Custom fonts require matching system installation or stylesheet injection to render correctly.
                </div>
              </div>
            )}
          </div>
        </Card>

        {/* Theme Settings */}
        <Card>
          <FormField
            label="Visual Interface Theme"
            id="appearance-theme"
            helperText="The visual theme settings. Gated to SENTINEL system theme for v1."
          >
            <select value={theme} disabled style={{ opacity: 0.7, cursor: 'not-allowed' }}>
              <option value="dark" style={{ background: '#1c1c1e' }}>Sentinel Dark (Default - Locked)</option>
            </select>
          </FormField>
        </Card>
      </div>
    </div>
  );
};

// 4. SharePointAssessmentPage component
export const SharePointAssessmentPage: React.FC = () => {
  const initSettings = useSettingsStore((s) => s.init);
  const settingsInitialized = useSettingsStore((s) => s.initialized);
  const spBaseUrl = useSettingsStore((s) => s.spBaseUrl);
  const updateSetting = useSettingsStore((s) => s.updateSetting);
  const toast = useToast();

  const [inputUrl, setInputUrl] = useState(spBaseUrl);

  useEffect(() => {
    if (!settingsInitialized) {
      void initSettings();
    }
  }, [initSettings, settingsInitialized]);

  useEffect(() => {
    setInputUrl(spBaseUrl);
  }, [spBaseUrl]);

  const handleSave = async () => {
    const trimmed = inputUrl.trim();
    if (!trimmed) {
      toast.error('SharePoint Connection URL cannot be empty');
      return;
    }
    // Validate connection string (accepts http/https URLs, domains, and IP addresses)
    const isValid = /^(https?:\/\/)?([\w.-]+|(\d{1,3}\.){3}\d{1,3})(:\d+)?(\/.*)?$/.test(trimmed);
    if (!isValid) {
      toast.error('Please enter a valid URL, Domain, or IP Address');
      return;
    }

    try {
      await updateSetting('spBaseUrl', trimmed);
      toast.success('SharePoint connection URL saved successfully!');
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : String(err);
      toast.error(`Failed to save configuration: ${errMsg}`);
    }
  };

  return (
    <div style={containerStyle}>
      <h1 style={titleStyle}>SharePoint Assessment</h1>
      <p style={descStyle}>Configure the SharePoint connection endpoint URL used by assessment agents.</p>
      <SettingsNav />

      <div style={{ maxWidth: '600px', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
        <Card>
          <FormField
            label="SharePoint Connection URL"
            id="sharepoint-base-url"
            helperText="The connection URL to Microsoft SharePoint. Supports domain hostnames (e.g. sharepoint.minsait.com) and IP addresses (e.g. 192.168.1.100)."
          >
            <input
              type="text"
              value={inputUrl}
              onChange={(e) => setInputUrl(e.target.value)}
              placeholder="e.g. https://sharepoint.minsait.com or 192.168.1.100"
            />
          </FormField>

          <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 'var(--space-4)' }}>
            <Button variant="primary" onClick={handleSave}>
              Save Connection URL
            </Button>
          </div>
        </Card>
      </div>
    </div>
  );
};
