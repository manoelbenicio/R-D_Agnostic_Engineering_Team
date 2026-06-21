import React, { useEffect, useState, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { Modal } from '@/design-system/components/Modal';
import { Card } from '@/design-system/components/Card';
import { Button } from '@/design-system/components/Button';
import { StatusBadge } from '@/design-system/components/StatusBadge';
import { Badge } from '@/design-system/components/Badge';
import { Prose } from '@/design-system/components/Prose';
import { NoProvidersNotice } from '@/api/key-store/no-providers-notice';
import { useValidatedProviders } from '@/api/key-store/use-validated-providers';
import { useVoiceStore } from './store';
import { getSTTEngine } from './engine';
import { extractIntent } from './nlu';
import { voiceToCanvas } from './voice-to-canvas';
import { dbPut } from '@/shared/storage/idb';
import { CanvasDocument } from './types';
import { matchRuntimeCommand, RuntimeCommand } from './runtime-commands';
import { useToast } from '@/shell/toasts';
import { goCoreClient } from '@/api';
import { canvasCommandBus } from '@/shell/canvas-command-adapter';
import {
  executeRuntimeCommand,
  type CommandExecutorDeps,
  type ConfirmOptions,
} from './command-executor';

export interface VoicePanelProps {
  currentCanvas?: CanvasDocument | null;
  onUpdateCanvas?: (updater: (current: CanvasDocument) => CanvasDocument) => void;
}

interface DestructiveConfirmState {
  isOpen: boolean;
  title: string;
  message: string;
  resolve: (confirmed: boolean) => void;
}

export const VoicePanel: React.FC<VoicePanelProps> = ({ currentCanvas, onUpdateCanvas }) => {
  const navigate = useNavigate();
  const toast = useToast();
  const isOpen = useVoiceStore((s) => s.isOpen);
  const setOpen = useVoiceStore((s) => s.setOpen);
  const voiceState = useVoiceStore((s) => s.voiceState);
  const setState = useVoiceStore((s) => s.setState);
  const interimTranscript = useVoiceStore((s) => s.interimTranscript);
  const finalTranscript = useVoiceStore((s) => s.finalTranscript);
  const setInterimTranscript = useVoiceStore((s) => s.setInterimTranscript);
  const setFinalTranscript = useVoiceStore((s) => s.setFinalTranscript);
  const intent = useVoiceStore((s) => s.intent);
  const setIntent = useVoiceStore((s) => s.setIntent);
  const error = useVoiceStore((s) => s.error);
  const setError = useVoiceStore((s) => s.setError);
  const reset = useVoiceStore((s) => s.reset);

  const validatedProviders = useValidatedProviders();
  const [canvasDoc, setCanvasDoc] = useState<CanvasDocument | null>(null);
  const [softWarning, setSoftWarning] = useState<string | null>(null);
  const [confirmDestructive, setConfirmDestructive] =
    useState<DestructiveConfirmState | null>(null);

  // Keep track of active engine to stop it on unmount
  const engineRef = useRef<ReturnType<typeof getSTTEngine> | null>(null);

  const hasProviders = validatedProviders.length > 0;

  const handleClose = () => {
    if (engineRef.current) {
      engineRef.current.stop();
    }
    reset();
    setOpen(false);
    setSoftWarning(null);
  };

  /**
   * Bridge between the executor's `confirm(opts)` Promise contract and the
   * existing destructive-confirmation modal. Resolves with `true` when the
   * user clicks Confirm, `false` when they Cancel or close the modal.
   *
   * Future direction: the destructive-confirmation modal could be lifted
   * into a shell-owned service so that this local closure disappears, but
   * that is out of scope for the current bus refactor.
   */
  const requestConfirm = (opts: ConfirmOptions): Promise<boolean> =>
    new Promise<boolean>((resolve) => {
      setConfirmDestructive({
        isOpen: true,
        title: opts.title,
        message: opts.message,
        resolve,
      });
    });

  /**
   * Build the executor deps from the panel's React-side resources.
   * The shape matches `CommandExecutorDeps` exactly — see command-executor.ts
   * for the full contract. Canvas-builder + canvas-reconciler access goes
   * through the `canvasCommandBus` adapter (see
   * `@/shell/canvas-command-adapter`), so this file no longer takes any
   * sideways capability imports.
   */
  const buildExecutorDeps = (): CommandExecutorDeps => ({
    canvas: currentCanvas ?? null,
    goCore: goCoreClient,
    toast,
    navigate,
    confirm: requestConfirm,
    bus: canvasCommandBus,
    onUpdateCanvas,
    speak: (text, lang) => {
      const utterance = new SpeechSynthesisUtterance(text);
      if (lang) {
        utterance.lang = lang;
      }
      window.speechSynthesis.speak(utterance);
    },
    emit: (event, command) => {
      window.dispatchEvent(
        new CustomEvent(`voice-canvas-${event}`, { detail: command }),
      );
    },
  });

  const handleExecuteCommand = async (command: RuntimeCommand): Promise<void> => {
    const deps = buildExecutorDeps();
    await executeRuntimeCommand(command, deps);
    // Confirm-bearing actions (kill / stop_all) deferred the panel close until
    // the user resolved the modal — close now.
    if (command.action === 'kill' || command.action === 'stop_all') {
      handleClose();
    }
  };

  const startListening = () => {
    reset();
    setState('listening');
    setSoftWarning(null);

    const stt = getSTTEngine('pt-BR');
    engineRef.current = stt;

    stt.start({
      onPartial: (text) => {
        setInterimTranscript(text);
      },
      onFinal: (text) => {
        setFinalTranscript(text);
      },
      onError: (err: unknown) => {
        let errMsg = '';
        if (err instanceof Error) {
          errMsg = err.message;
        } else if (err && typeof err === 'object' && 'message' in err) {
          errMsg = String((err as { message?: unknown }).message);
        } else {
          errMsg = String(err);
        }
        setError(errMsg);
        setState('error');
      },
      onEnd: () => {
        // If the user hasn't finished, end is triggered manually or by silence.
      },
    });
  };

  const stopListening = async () => {
    if (engineRef.current) {
      engineRef.current.stop();
    }
    setState('processing');

    const transcript = (finalTranscript + ' ' + interimTranscript).trim();
    if (!transcript) {
      setError('No speech detected.');
      setState('error');
      return;
    }

    const runtimeCommand = matchRuntimeCommand(transcript);
    if (runtimeCommand) {
      // Non-destructive commands close the panel immediately so toast +
      // navigation feedback is visible. Destructive ones (kill / stop_all)
      // keep the panel open until the confirmation modal resolves.
      if (runtimeCommand.action !== 'kill' && runtimeCommand.action !== 'stop_all') {
        handleClose();
      }
      void handleExecuteCommand(runtimeCommand);
      return;
    }

    // Set a timer for the 3-second soft warning (latency warning)
    const warningTimeout = setTimeout(() => {
      setSoftWarning('Parsing is taking longer than expected... continuing in background');
    }, 3000);

    try {
      const parsedIntent = await extractIntent(transcript);
      clearTimeout(warningTimeout);
      setIntent(parsedIntent);

      const doc = voiceToCanvas(parsedIntent);
      setCanvasDoc(doc);
      setState('confirming');
    } catch (err) {
      clearTimeout(warningTimeout);
      const errMsg = err instanceof Error ? err.message : String(err);
      setError(errMsg);
      setState('error');
    }
  };

  const handleCancel = () => {
    handleClose();
  };

  const handleEditBeforeDeploy = async () => {
    if (canvasDoc) {
      try {
        await dbPut('canvases', canvasDoc);
        handleClose();
        navigate(`/canvas/${canvasDoc.id}`);
      } catch (e) {
        const errMsg = e instanceof Error ? e.message : String(e);
        setError(`Failed to save canvas: ${errMsg}`);
        setState('error');
      }
    }
  };

  const handleGenerate = async () => {
    if (canvasDoc) {
      try {
        canvasDoc.deploy_state = {
          status: 'deploying',
        };
        await dbPut('canvases', canvasDoc);
        handleClose();
        navigate(`/canvas/${canvasDoc.id}`);
      } catch (e) {
        const errMsg = e instanceof Error ? e.message : String(e);
        setError(`Failed to save canvas: ${errMsg}`);
        setState('error');
      }
    }
  };

  useEffect(() => {
    return () => {
      if (engineRef.current) {
        engineRef.current.stop();
      }
    };
  }, []);

  const renderContent = () => {
    if (!hasProviders) {
      return (
        <div style={{ padding: 'var(--space-2)' }}>
          <NoProvidersNotice message="Voice requires at least one validated LLM provider" />
        </div>
      );
    }

    switch (voiceState) {
      case 'idle':
        return (
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 'var(--space-4)', padding: 'var(--space-4) 0' }}>
            <Button
              variant="primary"
              onClick={startListening}
              style={{
                width: '80px',
                height: '80px',
                borderRadius: '50%',
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                fontSize: '2rem',
                background: 'var(--cyan)',
                border: 'none',
                boxShadow: '0 0 15px var(--cyan-edge)',
              }}
            >
              🎤
            </Button>
            <div style={{ fontFamily: 'var(--font-mono)', fontSize: '0.875rem', color: 'var(--text-muted)' }}>
              Click to start speaking
            </div>
            <div style={{ fontFamily: 'var(--font-mono)', fontSize: '0.75rem', color: 'var(--text-dim)' }}>
              Or press <kbd style={{ background: 'var(--surface-elevated-hover)', padding: '2px 6px', borderRadius: '4px' }}>Ctrl+Shift+V</kbd> anywhere
            </div>
          </div>
        );

      case 'listening':
        return (
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 'var(--space-4)', padding: 'var(--space-4) 0' }}>
            <style>{`
              @keyframes pulse {
                0% { transform: scale(1); box-shadow: 0 0 0 0 rgba(0, 176, 189, 0.4); }
                70% { transform: scale(1.08); box-shadow: 0 0 0 15px rgba(0, 176, 189, 0); }
                100% { transform: scale(1); box-shadow: 0 0 0 0 rgba(0, 176, 189, 0); }
              }
              .pulse-mic {
                animation: pulse 1.5s infinite;
              }
            `}</style>
            <Button
              variant="primary"
              onClick={stopListening}
              className="pulse-mic"
              style={{
                width: '80px',
                height: '80px',
                borderRadius: '50%',
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                fontSize: '2rem',
                background: 'var(--threat)',
                border: 'none',
              }}
            >
              🛑
            </Button>
            <div style={{ fontFamily: 'var(--font-mono)', fontSize: '0.875rem', color: 'var(--threat)', fontWeight: 600 }}>
              Listening... Click to process
            </div>
            <Card
              style={{
                width: '100%',
                background: 'rgba(0, 0, 0, 0.3)',
                padding: 'var(--space-3)',
                minHeight: '80px',
                maxHeight: '150px',
                overflowY: 'auto',
                boxSizing: 'border-box',
              }}
            >
              <Prose style={{ fontSize: '0.875rem', margin: 0 }}>
                {finalTranscript || interimTranscript ? (
                  <>
                    <span style={{ color: 'var(--text-primary)' }}>{finalTranscript}</span>
                    <span style={{ color: 'var(--text-muted)', fontStyle: 'italic' }}>
                      {interimTranscript ? ` ${interimTranscript}` : ''}
                    </span>
                  </>
                ) : (
                  <span style={{ color: 'var(--text-dim)' }}>Speak now...</span>
                )}
              </Prose>
            </Card>
            <Button variant="secondary" onClick={handleCancel}>
              Cancel
            </Button>
          </div>
        );

      case 'processing':
        return (
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 'var(--space-4)', padding: 'var(--space-4) 0' }}>
            <style>{`
              @keyframes rotate {
                100% { transform: rotate(360deg); }
              }
              @keyframes dash {
                0% { stroke-dasharray: 1, 150; stroke-dashoffset: 0; }
                50% { stroke-dasharray: 90, 150; stroke-dashoffset: -35; }
                100% { stroke-dasharray: 90, 150; stroke-dashoffset: -124; }
              }
              .spinner-ring {
                animation: rotate 2s linear infinite;
              }
              .spinner-ring-path {
                stroke-dasharray: 1, 150;
                stroke-dashoffset: 0;
                animation: dash 1.5s ease-in-out infinite;
                stroke: var(--cyan);
              }
            `}</style>
            <div style={{ width: '80px', height: '80px', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
              <svg className="spinner-ring" viewBox="0 0 50 50" style={{ width: '50px', height: '50px' }}>
                <circle
                  className="spinner-ring-path"
                  cx="25"
                  cy="25"
                  r="20"
                  fill="none"
                  strokeWidth="4"
                  strokeMiterlimit="10"
                />
              </svg>
            </div>
            <div style={{ fontFamily: 'var(--font-mono)', fontSize: '0.875rem', color: 'var(--text-primary)' }}>
              Extracting intent...
            </div>
            {softWarning && (
              <div
                style={{
                  fontSize: '0.75rem',
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--amber)',
                  background: 'var(--amber-soft)',
                  border: '1px solid var(--amber-edge)',
                  borderRadius: 'var(--radius-button)',
                  padding: 'var(--space-2)',
                  textAlign: 'center',
                  maxWidth: '90%',
                }}
              >
                ⚠️ {softWarning}
              </div>
            )}
            <Button variant="secondary" onClick={handleCancel}>
              Cancel
            </Button>
          </div>
        );

      case 'confirming': {
        const confidenceVal = intent?.confidence ? Math.round(intent.confidence * 100) : null;

        return (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
            <h4 style={{ fontFamily: 'var(--font-display)', margin: '0 0 var(--space-2) 0', fontSize: '1rem', borderBottom: '1px solid var(--border)', paddingBottom: 'var(--space-2)' }}>
              Extracted Canvas Structure: <span style={{ color: 'var(--cyan)' }}>{intent?.name}</span>
            </h4>

            <div style={{ display: 'flex', gap: 'var(--space-2)', flexWrap: 'wrap', marginBottom: 'var(--space-2)' }}>
              <Badge variant="processing">{intent?.nodes.length} Agents</Badge>
              <Badge variant="processing">{intent?.edges.length} Connections</Badge>
              {confidenceVal !== null && (
                <StatusBadge status="completed" label={`${confidenceVal}% Confidence`} />
              )}
            </div>

            <Card style={{ background: 'rgba(0, 0, 0, 0.2)', padding: 'var(--space-3)', maxHeight: '200px', overflowY: 'auto' }}>
              <div style={{ fontFamily: 'var(--font-mono)', fontSize: '0.75rem', color: 'var(--text-muted)', textTransform: 'uppercase', marginBottom: 'var(--space-2)' }}>
                Agents Configuration
              </div>
              <ul style={{ margin: 0, paddingLeft: 'var(--space-4)', fontSize: '0.875rem', color: 'var(--text-primary)', lineHeight: 1.6 }}>
                {intent?.nodes.map((n, i) => (
                  <li key={i}>
                    <strong>{n.display_name}</strong> - <span style={{ color: 'var(--cyan)' }}>{n.role}</span> ({n.provider})
                  </li>
                ))}
              </ul>

              <div style={{ fontFamily: 'var(--font-mono)', fontSize: '0.75rem', color: 'var(--text-muted)', textTransform: 'uppercase', margin: 'var(--space-3) 0 var(--space-2) 0' }}>
                Handoff Flow
              </div>
              {intent?.edges && intent.edges.length > 0 ? (
                <ul style={{ margin: 0, paddingLeft: 'var(--space-4)', fontSize: '0.875rem', color: 'var(--text-primary)', lineHeight: 1.6 }}>
                  {intent.edges.map((e, i) => (
                    <li key={i}>
                      {e.from} ➔ {e.to} <Badge variant="idle" style={{ fontSize: '0.675rem' }}>{e.type}</Badge>
                    </li>
                  ))}
                </ul>
              ) : (
                <div style={{ fontSize: '0.875rem', color: 'var(--text-dim)' }}>No handoff edges declared.</div>
              )}
            </Card>

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 'var(--space-2)', marginTop: 'var(--space-2)' }}>
              <Button variant="ghost" onClick={handleCancel}>
                Cancel
              </Button>
              <Button variant="secondary" onClick={handleEditBeforeDeploy}>
                Edit Before Deploy
              </Button>
              <Button variant="primary" onClick={handleGenerate}>
                Generate & Deploy
              </Button>
            </div>
          </div>
        );
      }

      case 'error':
        return (
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 'var(--space-4)', padding: 'var(--space-4) 0' }}>
            <div
              style={{
                width: '60px',
                height: '60px',
                borderRadius: '50%',
                background: 'rgba(255, 59, 48, 0.1)',
                border: '1px solid var(--threat)',
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                fontSize: '1.75rem',
                color: 'var(--threat)',
              }}
            >
              ✕
            </div>
            <div style={{ fontFamily: 'var(--font-mono)', fontSize: '0.875rem', color: 'var(--threat)', fontWeight: 600 }}>
              An error occurred:
            </div>
            <div
              style={{
                fontSize: '0.875rem',
                color: 'var(--text-primary)',
                textAlign: 'center',
                background: 'rgba(0, 0, 0, 0.2)',
                padding: 'var(--space-3)',
                borderRadius: 'var(--radius-button)',
                border: '1px solid var(--border)',
                maxWidth: '90%',
              }}
            >
              {error}
            </div>
            <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
              <Button variant="secondary" onClick={handleCancel}>
                Cancel
              </Button>
              <Button variant="primary" onClick={startListening}>
                Retry
              </Button>
            </div>
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <>
      <Modal
        isOpen={isOpen}
        onClose={handleClose}
        title="Speech to Canvas"
        actions={null} // custom buttons rendered per state
      >
        {renderContent()}
      </Modal>

      {confirmDestructive && (
        <Modal
          isOpen={confirmDestructive.isOpen}
          onClose={() => {
            confirmDestructive.resolve(false);
            setConfirmDestructive(null);
          }}
          title={confirmDestructive.title}
          actions={
            <div style={{ display: 'flex', gap: '12px' }}>
              <Button
                variant="secondary"
                onClick={() => {
                  confirmDestructive.resolve(false);
                  setConfirmDestructive(null);
                }}
                autoFocus
              >
                Cancel
              </Button>
              <Button
                variant="primary"
                onClick={() => {
                  confirmDestructive.resolve(true);
                  setConfirmDestructive(null);
                }}
              >
                Confirm
              </Button>
            </div>
          }
        >
          <div style={{ fontFamily: 'var(--font-mono)', fontSize: '0.875rem' }}>
            {confirmDestructive.message}
          </div>
        </Modal>
      )}
    </>
  );
};

export default VoicePanel;
