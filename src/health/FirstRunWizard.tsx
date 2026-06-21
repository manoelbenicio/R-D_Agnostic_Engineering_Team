/* eslint-disable agentverse/no-sideways-capability-imports */
import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { goCoreClient } from '@/api';
import { useKeyStore } from '@/api/key-store/store';
import { PROVIDERS_REGISTRY, ProviderType } from '@/api/key-store/registry';
import { TEMPLATES, instantiateTemplate } from '@/canvas-templates';
import { canvasStore } from '@/canvas-document/store';
import { dbPut } from '@/shared/storage/idb';
import { useToast } from '@/shell/toasts';
import { Card } from '@/design-system/components/Card';
import { Button } from '@/design-system/components/Button';
import { FormField } from '@/design-system/components/FormField';
import { StatusBadge } from '@/design-system/components/StatusBadge';

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

interface FirstRunWizardProps {
  onClose: () => void;
}

export const FirstRunWizard: React.FC<FirstRunWizardProps> = ({ onClose }) => {
  const navigate = useNavigate();
  const toast = useToast();

  const keyStoreInit = useKeyStore((s) => s.init);
  const keyStoreInitialized = useKeyStore((s) => s.initialized);
  const setKeyInStore = useKeyStore((s) => s.setKey);
  const setInvalidInStore = useKeyStore((s) => s.setInvalid);

  const [step, setStep] = useState(1);
  const [loading, setLoading] = useState(false);

  // Step 1: Verify GO Core states
  const [goCoreUrl, setGoCoreUrl] = useState('');
  const [goCoreOk, setGoCoreOk] = useState(false);
  const [goCoreError, setGoCoreError] = useState<string | null>(null);

  // Step 2: Configure Provider states
  const [selectedProvider, setSelectedProvider] = useState<ProviderType>('google');
  const [providerKeys, setProviderKeys] = useState<Record<string, string>>({});
  const [keyValidated, setKeyValidated] = useState(false);
  const [keyError, setKeyError] = useState<string | null>(null);

  useEffect(() => {
    if (!keyStoreInitialized) {
      void keyStoreInit();
    }
  }, [keyStoreInit, keyStoreInitialized]);

  // Step 1: Run connection check on mount or when url changes
  const runGoCoreCheck = async (urlToCheck: string) => {
    setLoading(true);
    setGoCoreError(null);
    const originalUrl = goCoreClient.baseUrl;

    if (urlToCheck) {
      goCoreClient.baseUrl = urlToCheck.replace(/\/$/, '');
    }

    try {
      const res = await goCoreClient.getHealth();
      if (res && res.status === 'ok') {
        setGoCoreOk(true);
        setGoCoreError(null);
      } else {
        setGoCoreOk(false);
        setGoCoreError('Runtime engine returned an unexpected health payload.');
        // Revert URL
        goCoreClient.baseUrl = originalUrl;
      }
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : String(err);
      setGoCoreOk(false);
      setGoCoreError(`Cannot connect to runtime engine: ${errMsg}`);
      // Revert URL
      goCoreClient.baseUrl = originalUrl;
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    setGoCoreUrl(goCoreClient.baseUrl);
    void runGoCoreCheck(goCoreClient.baseUrl);
  }, []);

  const handleRetryGoCore = () => {
    void runGoCoreCheck(goCoreUrl);
  };

  // Step 2: Validate API key
  const handleValidateKey = async () => {
    const providerDef = PROVIDERS_REGISTRY.find((p) => p.id === selectedProvider);
    if (!providerDef) return;

    for (const field of providerDef.fields) {
      if (!providerKeys[field.name]) {
        toast.error(`Please enter ${field.label}`);
        return;
      }
    }

    setLoading(true);
    setKeyError(null);
    setKeyValidated(false);

    try {
      const res = await validateProviderKeys(selectedProvider, providerKeys);
      if (res.ok) {
        await setKeyInStore(selectedProvider, providerKeys, res.models || []);
        setKeyValidated(true);
        toast.success(`Validated provider ${providerDef.label} successfully!`);
      } else {
        setInvalidInStore(selectedProvider);
        setKeyError(res.error || 'Validation failed. Check your API keys.');
        toast.error(res.error || 'Validation failed.');
      }
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : String(err);
      setInvalidInStore(selectedProvider);
      setKeyError(errMsg);
      toast.error(`Unexpected validation failure: ${errMsg}`);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyInputChange = (fieldName: string, value: string) => {
    setProviderKeys((prev) => ({
      ...prev,
      [fieldName]: value,
    }));
    setKeyValidated(false);
    setKeyError(null);
  };

  // Complete Onboarding
  const completeOnboarding = async () => {
    try {
      await dbPut('app_state', { key: 'wizard_completed', value: true });
    } catch (err) {
      console.error('Failed to save wizard completion state:', err);
    }
    onClose();
  };

  const handleUseTemplate = async (templateId: string) => {
    setLoading(true);
    try {
      const doc = instantiateTemplate(templateId);
      const saved = await canvasStore.save(doc);
      await completeOnboarding();
      navigate(`/canvas/${saved.id}`);
      toast.success(`Created canvas from template '${doc.name}'`);
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : String(err);
      toast.error(`Failed to instantiate template: ${errMsg}`);
    } finally {
      setLoading(false);
    }
  };

  const handleStartBlank = async () => {
    setLoading(true);
    try {
      const draft = canvasStore.createDraft();
      const saved = await canvasStore.save(draft);
      await completeOnboarding();
      navigate(`/canvas/${saved.id}`);
      toast.success('Started a blank canvas.');
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : String(err);
      toast.error(`Failed to create blank canvas: ${errMsg}`);
    } finally {
      setLoading(false);
    }
  };

  const handleSkipWizard = async () => {
    await completeOnboarding();
    navigate('/');
    toast.info('Skipped onboarding wizard.');
  };

  const currentProviderDef = PROVIDERS_REGISTRY.find((p) => p.id === selectedProvider);

  return (
    <div className="wizard-overlay">
      <Card className="wizard-card">
        <header className="wizard-header">
          <div>
            <h2>AgentVerse Setup</h2>
            <div className="wizard-step-indicator">Step {step} of 3</div>
          </div>
          <Button variant="ghost" onClick={handleSkipWizard}>
            Skip Setup
          </Button>
        </header>

        <div className="wizard-step-content">
          {/* Step 1: Verify Runtime */}
          {step === 1 && (
            <div>
              <p className="wizard-step-description">
                AgentVerse coordinates with an orchestration runtime. Let&apos;s verify the connection.
              </p>
              {loading ? (
                <div style={{ padding: 'var(--space-4)', textAlign: 'center', fontFamily: 'var(--font-mono)' }}>
                  Verifying runtime engine connection...
                </div>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
                    <span>Connection Status:</span>
                    <StatusBadge
                      status={goCoreOk ? 'completed' : 'error'}
                      label={goCoreOk ? 'Connected' : 'Disconnected'}
                    />
                  </div>

                  {!goCoreOk && (
                    <Card glow="red" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                      <p style={{ margin: 0, fontSize: '0.85rem', color: 'var(--threat)', fontFamily: 'var(--font-mono)' }}>
                        {goCoreError}
                      </p>
                      <FormField label="Edit Runtime Base URL" id="wizard-go-core-url">
                        <input
                          type="text"
                          value={goCoreUrl}
                          onChange={(e) => setGoCoreUrl(e.target.value)}
                          placeholder="e.g. http://127.0.0.1:8080"
                        />
                      </FormField>
                      <div style={{ display: 'flex', gap: 'var(--space-2)', justifyContent: 'flex-end' }}>
                        <Button variant="secondary" onClick={handleRetryGoCore}>
                          Retry Connection
                        </Button>
                        <Button variant="ghost" onClick={() => setStep(2)}>
                          Skip & Continue
                        </Button>
                      </div>
                    </Card>
                  )}

                  {goCoreOk && (
                    <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 'var(--space-4)' }}>
                      <Button variant="primary" onClick={() => setStep(2)}>
                        Next: Configure Provider
                      </Button>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Step 2: Configure Provider */}
          {step === 2 && (
            <div>
              <p className="wizard-step-description">
                Configure at least one validated LLM API key provider to enable deploying and running canvases.
              </p>

              <Card style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                <FormField label="Select Provider" id="wizard-provider-select">
                  <select
                    value={selectedProvider}
                    onChange={(e) => {
                      setSelectedProvider(e.target.value as ProviderType);
                      setProviderKeys({});
                      setKeyValidated(false);
                      setKeyError(null);
                    }}
                  >
                    {PROVIDERS_REGISTRY.map((p) => (
                      <option key={p.id} value={p.id} style={{ background: '#1c1c1e' }}>
                        {p.label}
                      </option>
                    ))}
                  </select>
                </FormField>

                {currentProviderDef?.fields.map((field) => (
                  <FormField key={field.name} label={field.label} id={`wizard-key-${field.name}`}>
                    <input
                      type={field.type}
                      value={providerKeys[field.name] || ''}
                      disabled={loading}
                      onChange={(e) => handleKeyInputChange(field.name, e.target.value)}
                      placeholder={`Enter ${field.label}`}
                    />
                  </FormField>
                ))}

                {keyError && (
                  <div style={{ color: 'var(--threat)', fontFamily: 'var(--font-mono)', fontSize: '0.82rem' }}>
                    Error: {keyError}
                  </div>
                )}

                {keyValidated && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: 'var(--cyan)' }}>
                    <span>✓ validated</span>
                  </div>
                )}

                <div style={{ display: 'flex', gap: 'var(--space-2)', justifyContent: 'flex-end', marginTop: 'var(--space-3)' }}>
                  {!keyValidated ? (
                    <Button variant="secondary" disabled={loading} onClick={handleValidateKey}>
                      {loading ? 'Validating...' : 'Validate & Save'}
                    </Button>
                  ) : (
                    <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
                      <Button variant="secondary" onClick={() => setKeyValidated(false)}>
                        Edit Key
                      </Button>
                    </div>
                  )}
                </div>
              </Card>

              <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 'var(--space-4)' }}>
                <Button variant="ghost" onClick={() => setStep(1)}>
                  Back
                </Button>
                <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
                  <Button variant="secondary" onClick={() => setStep(3)}>
                    Skip & Continue
                  </Button>
                  {keyValidated && (
                    <Button variant="primary" onClick={() => setStep(3)}>
                      Next: Choose Starting Point
                    </Button>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* Step 3: Pick starting point */}
          {step === 3 && (
            <div>
              <p className="wizard-step-description">
                Pick a workspace layout template to start building, or create a completely blank canvas.
              </p>

              {loading ? (
                <div style={{ padding: 'var(--space-5)', textAlign: 'center', fontFamily: 'var(--font-mono)' }}>
                  Instantiating canvas...
                </div>
              ) : (
                <>
                  <div className="wizard-templates-grid">
                    {TEMPLATES.map((tpl) => (
                      <Card key={tpl.id} className="wizard-template-card">
                        <div>
                          <div className="wizard-template-title">{tpl.name}</div>
                          <div className="wizard-template-desc">{tpl.description}</div>
                        </div>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
                          <div className="wizard-template-meta">
                            <span>{tpl.agent_count} agents</span>
                            <span>{tpl.primary_edge_type}</span>
                          </div>
                          <Button variant="secondary" onClick={() => handleUseTemplate(tpl.id)} style={{ padding: '4px 8px', fontSize: '0.8rem' }}>
                            Use Template
                          </Button>
                        </div>
                      </Card>
                    ))}
                  </div>

                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 'var(--space-4)' }}>
                    <Button variant="ghost" onClick={() => setStep(2)}>
                      Back
                    </Button>
                    <div style={{ display: 'flex', gap: 'var(--space-3)', alignItems: 'center' }}>
                      <Button variant="ghost" onClick={handleStartBlank}>
                        Start Blank Canvas
                      </Button>
                    </div>
                  </div>
                </>
              )}
            </div>
          )}
        </div>
      </Card>
    </div>
  );
};

export default FirstRunWizard;
