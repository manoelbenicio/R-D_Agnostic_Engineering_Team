/* eslint-disable agentverse/no-sideways-capability-imports -- HealthPage needs to read key-store status and check CAO-client API health directly for diagnosing capability status. */
import React, { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useKeyStore } from '@/api/key-store/store';
import { PROVIDERS_REGISTRY } from '@/api/key-store/registry';
import { caoClient } from '@/api/cao-client';
import { Button } from '@/design-system/components/Button';
import { StatusBadge } from '@/design-system/components/StatusBadge';
import { useToast } from '@/shell/toasts';
import './health.css';

interface ServerHealthState {
  cao: 'ok' | 'error' | 'loading';
  caoExplanation: string;
  tmux: 'ok' | 'error' | 'loading';
  tmuxExplanation: string;
  providers: Array<{ name: string; installed: boolean }>;
  providersLoading: boolean;
}

export const HealthPage: React.FC = () => {
  const navigate = useNavigate();
  const toast = useToast();

  const keyStoreInit = useKeyStore((s) => s.init);
  const keyStoreInitialized = useKeyStore((s) => s.initialized);
  const providerStatuses = useKeyStore((s) => s.statuses);

  // States
  const [serverHealth, setServerHealth] = useState<ServerHealthState>({
    cao: 'loading',
    caoExplanation: 'Checking runtime connectivity...',
    tmux: 'loading',
    tmuxExplanation: 'Checking tmux status...',
    providers: [],
    providersLoading: true,
  });

  const [browserCapabilities, setBrowserCapabilities] = useState({
    webgl2: 'loading',
    webgl2Explanation: 'Checking WebGL2 context...',
    indexedDb: 'loading',
    indexedDbExplanation: 'Checking IndexedDB support...',
    microphone: 'loading' as 'ok' | 'warning' | 'error' | 'loading',
    microphoneExplanation: 'Checking microphone permission...',
  });

  const [testingMic, setTestingMic] = useState(false);

  // Initialize key store
  useEffect(() => {
    if (!keyStoreInitialized) {
      void keyStoreInit();
    }
  }, [keyStoreInit, keyStoreInitialized]);

  // Check servers
  const checkServers = useCallback(async () => {
    setServerHealth((prev) => ({
      ...prev,
      cao: 'loading',
      tmux: 'loading',
      providersLoading: true,
    }));

    let caoStatus: 'ok' | 'error' = 'error';
    let caoExpl = `Cannot reach the runtime at ${caoClient.baseUrl}`;
    let tmuxStatus: 'ok' | 'error' = 'error';
    let tmuxExpl = 'Cannot communicate with tmux service.';
    let providerList: Array<{ name: string; installed: boolean }> = [];

    // 1. Runtime engine check
    try {
      const res = await caoClient.getHealth();
      if (res && res.status === 'ok') {
        caoStatus = 'ok';
        caoExpl = 'Runtime engine is running and responding.';
      } else {
        caoExpl = `Runtime returned an unexpected status: ${JSON.stringify(res)}`;
      }
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : String(err);
      caoExpl = `Cannot reach the runtime at ${caoClient.baseUrl} (${errMsg})`;
    }

    // 2. tmux Check
    try {
      await caoClient.listSessions();
      tmuxStatus = 'ok';
      tmuxExpl = 'tmux Server is active and operational.';
    } catch {
      // Outage or connection issue
    }

    // 3. CAO Managed Providers Check
    try {
      const providers = await caoClient.listProviders();
      providerList = providers.map((p) => ({
        name: p.name,
        installed: p.installed,
      }));
    } catch {
      // Failed to list providers
    }

    setServerHealth({
      cao: caoStatus,
      caoExplanation: caoExpl,
      tmux: tmuxStatus,
      tmuxExplanation: tmuxExpl,
      providers: providerList,
      providersLoading: false,
    });
  }, []);

  const updateMicStatus = useCallback((state: PermissionState) => {
    let status: 'ok' | 'warning' | 'error' = 'warning';
    let explanation = 'Microphone permission prompt required.';

    if (state === 'granted') {
      status = 'ok';
      explanation = 'Microphone permission granted and active.';
    } else if (state === 'denied') {
      status = 'error';
      explanation = 'Microphone permission denied. Voice capabilities are disabled.';
    }

    setBrowserCapabilities((prev) => ({
      ...prev,
      microphone: status,
      microphoneExplanation: explanation,
    }));
  }, []);

  // Check browser capabilities
  const checkBrowser = useCallback(() => {
    // WebGL2 support
    const hasWebGL2 = typeof HTMLCanvasElement !== 'undefined' && !!document.createElement('canvas').getContext('webgl2');
    const webgl2Status = hasWebGL2 ? 'ok' : 'error';
    const webgl2Expl = hasWebGL2
      ? 'WebGL2 is enabled for hardware-accelerated terminal rendering.'
      : 'WebGL2 is required for the Terminal View';

    // IndexedDB support
    const hasIDB = typeof window !== 'undefined' && 'indexedDB' in window;
    const idbStatus = hasIDB ? 'ok' : 'error';
    const idbExpl = hasIDB
      ? 'IndexedDB is available for local workspace persistence.'
      : 'IndexedDB is required for persistence';

    setBrowserCapabilities((prev) => ({
      ...prev,
      webgl2: webgl2Status,
      webgl2Explanation: webgl2Expl,
      indexedDb: idbStatus,
      indexedDbExplanation: idbExpl,
    }));

    // Microphone Permission Check
    if (typeof navigator !== 'undefined' && navigator.permissions && navigator.permissions.query) {
      navigator.permissions
        .query({ name: 'microphone' as PermissionName })
        .then((permissionStatus) => {
          updateMicStatus(permissionStatus.state);
          permissionStatus.onchange = () => {
            updateMicStatus(permissionStatus.state);
          };
        })
        .catch(() => {
          setBrowserCapabilities((prev) => ({
            ...prev,
            microphone: 'warning',
            microphoneExplanation: 'Microphone permission status could not be queried. Click Test Microphone to verify.',
          }));
        });
    } else {
      setBrowserCapabilities((prev) => ({
        ...prev,
        microphone: 'warning',
        microphoneExplanation: 'Permissions API not supported in this browser. Click Test Microphone to verify.',
      }));
    }
  }, [updateMicStatus]);

  const handleTestMicrophone = async () => {
    if (testingMic) return;
    setTestingMic(true);
    toast.info('Requesting microphone access...');

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      // Stop tracks immediately
      stream.getTracks().forEach((track) => track.stop());

      setBrowserCapabilities((prev) => ({
        ...prev,
        microphone: 'ok',
        microphoneExplanation: 'Microphone permission granted and active.',
      }));
      toast.success('Microphone verified successfully!');
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : String(err);
      setBrowserCapabilities((prev) => ({
        ...prev,
        microphone: 'error',
        microphoneExplanation: `Microphone access denied: ${errMsg}`,
      }));
      toast.error('Microphone access was denied or failed.');
    } finally {
      setTestingMic(false);
    }
  };

  useEffect(() => {
    void checkServers();
    checkBrowser();
  }, [checkServers, checkBrowser]);

  return (
    <div className="health-page">
      <header className="health-header">
        <h1>System Health & Diagnostics</h1>
        <p>Verify Central Orchestrator status, validation of API keys, and browser compatibility flags.</p>
        <Button
          variant="secondary"
          disabled={serverHealth.providersLoading}
          onClick={() => {
            void checkServers();
            toast.info('Re-syncing runtime & CLI auth…');
          }}
        >
          {serverHealth.providersLoading ? 'Re-syncing…' : '↻ Re-sync Runtime & CLIs'}
        </Button>
      </header>

      <div className="health-sections">
        {/* 1. Server Health */}
        <section>
          <h2 className="health-section-title">Server Health</h2>
          <div className="health-grid">
            {/* Runtime Engine Row */}
            <div className={`health-row ${serverHealth.cao === 'error' ? 'health-row-error' : ''}`}>
              <div className="health-info">
                <span className="health-component-name">Runtime Engine</span>
                <span className="health-explanation">{serverHealth.caoExplanation}</span>
              </div>
              <div className="health-actions">
                <StatusBadge
                  status={serverHealth.cao === 'ok' ? 'completed' : serverHealth.cao === 'loading' ? 'processing' : 'error'}
                  label={serverHealth.cao === 'ok' ? 'Operational' : serverHealth.cao === 'loading' ? 'Checking...' : 'Outage'}
                />
                {serverHealth.cao === 'error' && (
                  <Button variant="secondary" onClick={() => navigate('/settings/general')}>
                    Fix
                  </Button>
                )}
              </div>
            </div>

            {/* tmux Server Row */}
            <div className={`health-row ${serverHealth.tmux === 'error' ? 'health-row-error' : ''}`}>
              <div className="health-info">
                <span className="health-component-name">tmux Server Connection</span>
                <span className="health-explanation">{serverHealth.tmuxExplanation}</span>
              </div>
              <div className="health-actions">
                <StatusBadge
                  status={serverHealth.tmux === 'ok' ? 'completed' : serverHealth.tmux === 'loading' ? 'processing' : 'error'}
                  label={serverHealth.tmux === 'ok' ? 'Operational' : serverHealth.tmux === 'loading' ? 'Checking...' : 'Outage'}
                />
              </div>
            </div>

            {/* CAO Managed Providers */}
            {serverHealth.providersLoading ? (
              <div className="health-row">
                <div className="health-info">
                  <span className="health-component-name">CAO Model Providers</span>
                  <span className="health-explanation">Retrieving installed provider modules...</span>
                </div>
                <StatusBadge status="processing" label="Loading..." />
              </div>
            ) : (
              serverHealth.providers.map((p) => (
                <div key={p.name} className={`health-row ${!p.installed ? 'health-row-warning' : ''}`}>
                  <div className="health-info">
                    <span className="health-component-name">Provider: {p.name}</span>
                    <span className="health-explanation">
                      {p.installed ? 'Provider engine is installed on CAO server.' : 'Provider engine is missing on CAO server.'}
                    </span>
                  </div>
                  <div className="health-actions">
                    <StatusBadge
                      status={p.installed ? 'completed' : 'idle'}
                      label={p.installed ? 'Installed' : 'Missing'}
                    />
                  </div>
                </div>
              ))
            )}
          </div>
        </section>

        {/* 2. Provider Validations */}
        <section>
          <h2 className="health-section-title">Provider Validations (BYOK)</h2>
          <div className="health-grid">
            {PROVIDERS_REGISTRY.map((provider) => {
              const status = providerStatuses[provider.id] || 'unset';
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
                  ? 'Invalid Key'
                  : 'Unconfigured';
              const explanation =
                status === 'set'
                  ? `API configuration set and validated for ${provider.label}.`
                  : status === 'invalid'
                  ? `Validation failed for ${provider.label}. Key might be invalid or expired.`
                  : `API keys for ${provider.label} are unconfigured.`;

              return (
                <div
                  key={provider.id}
                  className={`health-row ${status === 'invalid' ? 'health-row-error' : status === 'unset' ? 'health-row-warning' : ''}`}
                >
                  <div className="health-info">
                    <span className="health-component-name">{provider.label} Key Status</span>
                    <span className="health-explanation">{explanation}</span>
                  </div>
                  <div className="health-actions">
                    <StatusBadge status={badgeStatus} label={badgeLabel} />
                    {status !== 'set' && (
                      <Button variant="secondary" onClick={() => navigate('/settings/providers')}>
                        Configure
                      </Button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </section>

        {/* 3. Browser Capabilities */}
        <section>
          <h2 className="health-section-title">Browser Capabilities</h2>
          <div className="health-grid">
            {/* WebGL2 Row */}
            <div className={`health-row ${browserCapabilities.webgl2 === 'error' ? 'health-row-error' : ''}`}>
              <div className="health-info">
                <span className="health-component-name">WebGL2 Hardware Acceleration</span>
                <span className="health-explanation">{browserCapabilities.webgl2Explanation}</span>
              </div>
              <div className="health-actions">
                <StatusBadge
                  status={browserCapabilities.webgl2 === 'ok' ? 'completed' : 'error'}
                  label={browserCapabilities.webgl2 === 'ok' ? 'Supported' : 'Unsupported'}
                />
                {browserCapabilities.webgl2 === 'error' && (
                  <Button
                    variant="secondary"
                    onClick={() =>
                      window.open('https://get.webgl.org/webgl2/enable.html', '_blank', 'noopener,noreferrer')
                    }
                  >
                    Fix
                  </Button>
                )}
              </div>
            </div>

            {/* IndexedDB Row */}
            <div className={`health-row ${browserCapabilities.indexedDb === 'error' ? 'health-row-error' : ''}`}>
              <div className="health-info">
                <span className="health-component-name">IndexedDB Storage</span>
                <span className="health-explanation">{browserCapabilities.indexedDbExplanation}</span>
              </div>
              <div className="health-actions">
                <StatusBadge
                  status={browserCapabilities.indexedDb === 'ok' ? 'completed' : 'error'}
                  label={browserCapabilities.indexedDb === 'ok' ? 'Supported' : 'Unsupported'}
                />
              </div>
            </div>

            {/* Microphone Permission Row */}
            <div
              className={`health-row ${
                browserCapabilities.microphone === 'error'
                  ? 'health-row-error'
                  : browserCapabilities.microphone === 'warning'
                  ? 'health-row-warning'
                  : ''
              }`}
            >
              <div className="health-info">
                <span className="health-component-name">Microphone Permission (Voice Control)</span>
                <span className="health-explanation">{browserCapabilities.microphoneExplanation}</span>
              </div>
              <div className="health-actions">
                <StatusBadge
                  status={
                    browserCapabilities.microphone === 'ok'
                      ? 'completed'
                      : browserCapabilities.microphone === 'warning'
                      ? 'idle'
                      : 'error'
                  }
                  label={
                    browserCapabilities.microphone === 'ok'
                      ? 'Granted'
                      : browserCapabilities.microphone === 'warning'
                      ? 'Prompt Required'
                      : 'Denied'
                  }
                />
                <Button variant="secondary" disabled={testingMic} onClick={handleTestMicrophone}>
                  {testingMic ? 'Testing...' : 'Test Microphone'}
                </Button>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
};

export default HealthPage;
