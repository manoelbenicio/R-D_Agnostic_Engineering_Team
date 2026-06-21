import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { goCoreClient, sessionsQueryKeys, terminalQueryKeys } from '@/api';
import type { Terminal, InboxMessage } from '@/api';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { TerminalView } from '@/terminal';
import { TabBar } from './TabBar';
import { mapTerminalStatus } from './utils';
import { Button, Card, Badge, StatusBadge, Modal } from '@/design-system';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { useToast } from '@/shell/toasts';
import type { ViewMode } from './types';
import './terminal-grid.css';

export interface TerminalGridProps {
  sessionName: string;
}

export const TerminalGrid: React.FC<TerminalGridProps> = ({ sessionName }) => {
  const { id: canvasId, terminalId: routeTerminalId } = useParams<{ id: string; terminalId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const toast = useToast();

  const [viewMode, setViewMode] = useState<ViewMode>('tabs');
  const [priorViewMode, setPriorViewMode] = useState<'tabs' | 'grid'>('tabs');
  const [focusedId, setFocusedId] = useState<string | null>(routeTerminalId || null);
  const [menuOpenId, setMenuOpenId] = useState<string | null>(null);
  const [terminalToKill, setTerminalToKill] = useState<Terminal | null>(null);

  // Queries for terminals in the session
  const { data: terminals = [] } = useQuery<Terminal[]>({
    queryKey: sessionsQueryKeys.terminals(sessionName),
    queryFn: () => goCoreClient.listTerminalsInSession(sessionName),
    refetchInterval: 3000,
    refetchIntervalInBackground: false,
  });

  // Automatically set focusedId if not set and terminals are loaded
  useEffect(() => {
    if (!focusedId && terminals.length > 0 && terminals[0]) {
      setFocusedId(terminals[0].id);
    }
  }, [terminals, focusedId]);

  // Handle route terminal ID changes
  useEffect(() => {
    if (routeTerminalId) {
      setFocusedId(routeTerminalId);
    }
  }, [routeTerminalId]);

  // Escape key handler for fullscreen
  useEffect(() => {
    if (viewMode !== 'fullscreen') return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setViewMode(priorViewMode);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [viewMode, priorViewMode]);

  const handleAddTerminal = () => {
    // Adding a terminal means adding a new agent node to the canvas, then
    // re-deploying via the Reconciler's diff-based edit-after-deploy path
    // (canvas-reconciler/spec.md §5). Send the user to the canvas builder
    // for that canvas; if no canvas id is in the URL, surface an error toast
    // since the user reached this view without a parent canvas context.
    if (!canvasId) {
      toast.error('Cannot add terminal: missing canvas context.');
      return;
    }
    navigate(`/canvas/${canvasId}`);
  };

  const handleFocus = (id: string) => {
    setFocusedId(id);
    if (viewMode === 'grid') {
      setViewMode('tabs');
    }
  };

  const handleKillTerminal = (id: string) => {
    const term = terminals.find((t) => t.id === id);
    if (term) {
      setTerminalToKill(term);
    }
  };

  const executeKill = useMutation({
    mutationFn: (id: string) => goCoreClient.deleteTerminal(id),
    onSuccess: (_, id) => {
      toast.success(`Terminal killed successfully`);
      queryClient.invalidateQueries({ queryKey: sessionsQueryKeys.terminals(sessionName) });
      setTerminalToKill(null);
      if (focusedId === id) {
        setFocusedId(null);
      }
    },
    onError: (err: Error) => {
      toast.error(`Failed to kill terminal: ${err.message}`);
      setTerminalToKill(null);
    },
  });

  const handleToggleFullscreen = () => {
    if (viewMode === 'fullscreen') {
      setViewMode(priorViewMode);
    } else {
      setPriorViewMode(viewMode);
      setViewMode('fullscreen');
    }
  };

  return (
    <div className="terminal-grid-container">
      {viewMode !== 'fullscreen' && (
        <div className="terminal-grid-toolbar">
          <div className="toolbar-left">
            <Button
              variant={viewMode === 'tabs' ? 'primary' : 'secondary'}
              onClick={() => setViewMode('tabs')}
              data-testid="toggle-tabs-btn"
            >
              Tabs View
            </Button>
            <Button
              variant={viewMode === 'grid' ? 'primary' : 'secondary'}
              onClick={() => setViewMode('grid')}
              className="ml-2"
              data-testid="toggle-grid-btn"
            >
              Grid View
            </Button>
          </div>
          <div className="toolbar-right">
            {viewMode === 'tabs' && focusedId && (
              <Button variant="secondary" onClick={handleToggleFullscreen} data-testid="fullscreen-btn">
                Fullscreen
              </Button>
            )}
          </div>
        </div>
      )}

      {viewMode === 'fullscreen' && focusedId && (
        <div className="fullscreen-overlay">
          <div className="fullscreen-header">
            <span className="fullscreen-title">
              Fullscreen: {terminals.find((t) => t.id === focusedId)?.display_name || focusedId}
            </span>
            <Button variant="secondary" onClick={() => setViewMode(priorViewMode)} data-testid="exit-fullscreen-btn">
              Exit Fullscreen (Esc)
            </Button>
          </div>
          <div className="fullscreen-terminal-wrapper">
            <TerminalView terminalId={focusedId} readOnly={false} />
          </div>
          <div className="fullscreen-controls-toggle">
            <Button variant="secondary" onClick={() => setMenuOpenId(menuOpenId === focusedId ? null : focusedId)}>
              Controls Menu
            </Button>
            {menuOpenId === focusedId && (
              <div className="fullscreen-controls-popup">
                <TerminalControlsCard
                  terminalId={focusedId}
                  onKill={() => handleKillTerminal(focusedId)}
                  onCloseMenu={() => setMenuOpenId(null)}
                />
              </div>
            )}
          </div>
        </div>
      )}

      {viewMode === 'tabs' && (
        <div className="tabs-view-layout">
          <TabBar
            sessionName={sessionName}
            focusedId={focusedId}
            onFocus={handleFocus}
            onClose={handleKillTerminal}
            onAdd={handleAddTerminal}
          />
          {focusedId ? (
            <div className="focused-tab-content">
              <div className="terminal-panel">
                <TerminalView terminalId={focusedId} readOnly={false} />
              </div>
              <div className="controls-panel">
                <TerminalControlsCard
                  terminalId={focusedId}
                  onKill={() => handleKillTerminal(focusedId)}
                />
              </div>
            </div>
          ) : (
            <Card className="p-8 text-center text-muted">No terminal active. Add one to start.</Card>
          )}
        </div>
      )}

      {viewMode === 'grid' && (
        <div className="grid-view-layout">
          <div className="terminals-grid">
            {terminals.map((terminal) => (
              <Card
                key={terminal.id}
                className="grid-cell-card"
                onClick={() => {
                  setFocusedId(terminal.id);
                  setViewMode('tabs');
                }}
                data-testid={`grid-cell-${terminal.id}`}
              >
                <div className="grid-cell-header" onClick={(e) => e.stopPropagation()}>
                  <div className="header-info">
                    <span className="grid-cell-name">{terminal.display_name || terminal.profile}</span>
                    {terminal.provider && (
                      <Badge className="provider-badge ml-2">
                        {terminal.provider}
                      </Badge>
                    )}
                  </div>
                  <div className="header-actions">
                    <StatusBadge status={mapTerminalStatus(terminal.status)} />
                    <button
                      type="button"
                      className="cell-menu-btn ml-2"
                      onClick={() => setMenuOpenId(menuOpenId === terminal.id ? null : terminal.id)}
                      aria-label="Actions"
                      data-testid={`cell-menu-btn-${terminal.id}`}
                    >
                      &#8943;
                    </button>
                    {menuOpenId === terminal.id && (
                      <div className="cell-menu-dropdown">
                        <TerminalControlsCard
                          terminalId={terminal.id}
                          onKill={() => {
                            handleKillTerminal(terminal.id);
                            setMenuOpenId(null);
                          }}
                          onCloseMenu={() => setMenuOpenId(null)}
                        />
                      </div>
                    )}
                  </div>
                </div>
                <div className="grid-cell-terminal-wrapper">
                  <TerminalView terminalId={terminal.id} readOnly={true} />
                </div>
              </Card>
            ))}
            {terminals.length === 0 && (
              <Card className="p-8 text-center text-muted col-span-full">No terminals in this session.</Card>
            )}
          </div>
        </div>
      )}

      {/* Kill Confirmation Modal */}
      <Modal
        isOpen={!!terminalToKill}
        onClose={() => setTerminalToKill(null)}
        title="Kill Terminal"
        actions={
          <>
            <Button
              variant="secondary"
              onClick={() => setTerminalToKill(null)}
              autoFocus
              data-testid="kill-cancel-btn"
            >
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={() => {
                if (terminalToKill) {
                  executeKill.mutate(terminalToKill.id);
                }
              }}
              className="btn-danger"
              data-testid="kill-confirm-btn"
            >
              Kill
            </Button>
          </>
        }
      >
        <p>
          Are you sure you want to kill terminal{' '}
          <strong>{terminalToKill?.display_name || terminalToKill?.id}</strong>?
        </p>
        <p className="text-muted text-sm mt-2">
          This will terminate the execution and delete the running terminal instance.
        </p>
      </Modal>
    </div>
  );
};

interface TerminalControlsCardProps {
  terminalId: string;
  onKill: () => void;
  onCloseMenu?: () => void;
}

const TerminalControlsCard: React.FC<TerminalControlsCardProps> = ({
  terminalId,
  onKill,
  onCloseMenu,
}) => {
  const [message, setMessage] = useState('');

  // Fetch Working Directory
  const { data: workingDir = 'Loading...' } = useQuery({
    queryKey: terminalQueryKeys.workingDirectory(terminalId),
    queryFn: () => goCoreClient.getTerminalWorkingDirectory(terminalId),
    enabled: !!terminalId,
  });

  // Fetch Inbox Messages
  const { data: inboxMessages = [] } = useQuery<InboxMessage[]>({
    queryKey: terminalQueryKeys.inboxMessages(terminalId),
    queryFn: () => goCoreClient.listInboxMessages(terminalId),
    enabled: !!terminalId,
    refetchInterval: 3000,
  });

  // Send Message Mutation
  const sendMessageMutation = useMutation({
    mutationFn: (msg: string) => goCoreClient.sendTerminalInput(terminalId, msg),
    onSuccess: () => {
      setMessage('');
    },
  });

  const handleSendMessage = (e: React.FormEvent) => {
    e.preventDefault();
    if (!message.trim()) return;
    sendMessageMutation.mutate(message.trim());
  };

  return (
    <Card className="terminal-controls-card p-4">
      {onCloseMenu && (
        <div className="flex justify-end mb-2">
          <button type="button" className="close-menu-x" onClick={onCloseMenu}>
            &times;
          </button>
        </div>
      )}
      <div className="control-section mb-4">
        <label className="control-label">Working Directory</label>
        <div className="working-dir-display" data-testid="working-dir-display">
          {workingDir}
        </div>
      </div>

      <div className="control-section mb-4">
        <label className="control-label">Send Message</label>
        <form onSubmit={handleSendMessage} className="send-message-form">
          <input
            type="text"
            className="send-message-input"
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            placeholder="Type a message..."
            data-testid="send-message-input"
          />
          <Button type="submit" disabled={sendMessageMutation.isPending} className="ml-2">
            Send
          </Button>
        </form>
      </div>

      <div className="control-section mb-4">
        <label className="control-label">Inbox Messages ({inboxMessages.length})</label>
        <div className="inbox-messages-list" data-testid="inbox-messages-list">
          {inboxMessages.map((msg) => (
            <div key={msg.id} className="inbox-message-item">
              <span className="msg-sender">{msg.sender || 'system'}:</span>
              <span className="msg-body">{msg.message}</span>
            </div>
          ))}
          {inboxMessages.length === 0 && <div className="text-muted text-xs">No messages in inbox.</div>}
        </div>
      </div>

      <div className="control-section mt-6">
        <Button variant="secondary" onClick={onKill} className="w-full btn-danger" data-testid="kill-terminal-btn">
          Kill Terminal
        </Button>
      </div>
    </Card>
  );
};

export default TerminalGrid;
