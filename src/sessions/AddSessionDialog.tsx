import React, { useEffect, useState } from 'react';
import { Button, FormField } from '@/design-system';
import { useSessionStore } from '@/api/session-store';
import './add-session-dialog.css';

const PROVIDERS = [
  { id: 'claude_code', label: 'Claude Code' },
  { id: 'codex', label: 'Codex' },
  { id: 'gemini_cli', label: 'Gemini CLI' },
  { id: 'kiro_cli', label: 'Kiro CLI' },
] as const;

export interface AddSessionDialogProps {
  isOpen: boolean;
  onClose: () => void;
  defaultProvider?: string;
}

export const AddSessionDialog: React.FC<AddSessionDialogProps> = ({
  isOpen,
  onClose,
  defaultProvider,
}) => {
  const { addSession, clearError } = useSessionStore();
  const [provider, setProvider] = useState(defaultProvider ?? PROVIDERS[0].id);
  const [configDir, setConfigDir] = useState('');
  const [billingLabel, setBillingLabel] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [feedback, setFeedback] = useState<{ type: 'success' | 'error'; message: string } | null>(
    null,
  );

  useEffect(() => {
    if (!isOpen) return;
    setProvider(defaultProvider ?? PROVIDERS[0].id);
    setConfigDir('');
    setBillingLabel('');
    setSubmitting(false);
    setFeedback(null);
    clearError();
  }, [clearError, defaultProvider, isOpen]);

  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmitting(true);
    setFeedback(null);
    clearError();

    await addSession(provider, configDir.trim() || undefined);
    const storeError = useSessionStore.getState().error;

    if (storeError) {
      setFeedback({ type: 'error', message: storeError });
    } else {
      const labelSuffix = billingLabel.trim() ? ` for ${billingLabel.trim()}` : '';
      setFeedback({ type: 'success', message: `OAuth login started${labelSuffix}.` });
    }
    setSubmitting(false);
  };

  return (
    <div className="add-session-overlay" role="presentation" onMouseDown={onClose}>
      <section
        className="add-session-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="add-session-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header className="add-session-dialog-header">
          <div>
            <span>Runtime identity</span>
            <h2 id="add-session-title">Add Auth Session</h2>
          </div>
          <button
            type="button"
            className="add-session-close"
            onClick={onClose}
            aria-label="Close dialog"
          >
            &times;
          </button>
        </header>

        <form className="add-session-form" onSubmit={(event) => void handleSubmit(event)}>
          <FormField label="Provider" id="add-session-provider">
            <select
              value={provider}
              onChange={(event) => setProvider(event.target.value)}
              disabled={submitting}
            >
              {PROVIDERS.map((option) => (
                <option key={option.id} value={option.id}>
                  {option.label}
                </option>
              ))}
            </select>
          </FormField>

          <FormField
            label="Config directory"
            id="add-session-config-dir"
            helperText="Optional custom CLI config directory path."
          >
            <input
              value={configDir}
              onChange={(event) => setConfigDir(event.target.value)}
              placeholder="C:\\Users\\you\\.config\\provider"
              disabled={submitting}
            />
          </FormField>

          <FormField
            label="Billing label"
            id="add-session-billing-label"
            helperText="Optional friendly name for this login flow."
          >
            <input
              value={billingLabel}
              onChange={(event) => setBillingLabel(event.target.value)}
              placeholder="Team A Account"
              disabled={submitting}
            />
          </FormField>

          {feedback ? (
            <div
              className={`add-session-feedback add-session-feedback-${feedback.type}`}
              role="status"
            >
              {feedback.message}
            </div>
          ) : null}

          <footer className="add-session-actions">
            <Button type="button" variant="ghost" onClick={onClose} disabled={submitting}>
              Cancel
            </Button>
            <Button type="submit" variant="cyan" disabled={submitting}>
              {submitting ? 'Starting Login...' : 'Start OAuth Login'}
            </Button>
          </footer>
        </form>
      </section>
    </div>
  );
};

export default AddSessionDialog;
