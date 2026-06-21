import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Badge, Button, Card, FormField, Prose } from '@/design-system';
import { goCoreClient, sessionsQueryKeys, type ProviderType, type Terminal } from '@/api';
import { subscribeTerminalSocket } from '@/api/terminal-socket-fanout';
import { segmentByAgentBoundary } from './ansi-parser';
import type { ChatBubble, ChatViewMode, ParsedSegment } from './types';

export interface ChatViewProps {
  sessionName: string;
}

interface PendingTerminalBuffer {
  content: string;
  bubbleId?: string;
  timerId?: number;
}

const PARTIAL_FLUSH_MS = 500;
const SWIPE_THRESHOLD_PX = 60;

export const ChatView: React.FC<ChatViewProps> = ({ sessionName }) => {
  const [bubbles, setBubbles] = useState<ChatBubble[]>([]);
  const [composerValue, setComposerValue] = useState('');
  const [quotePrefix, setQuotePrefix] = useState('');
  const [openActionsBubbleId, setOpenActionsBubbleId] = useState<string | null>(null);
  const [statusMessage, setStatusMessage] = useState('');
  const [viewMode, setViewMode] = useState<ChatViewMode>('chat');
  const pendingByTerminal = useRef(new Map<string, PendingTerminalBuffer>());
  const latestTerminalId = useRef<string | null>(null);
  const touchStartX = useRef(new Map<string, number>());
  const textDecoder = useMemo(() => new TextDecoder(), []);

  const terminalsQuery = useQuery({
    queryKey: sessionsQueryKeys.terminals(sessionName),
    queryFn: () => goCoreClient.listTerminalsInSession(sessionName),
    enabled: sessionName.length > 0,
  });

  const terminals = useMemo(() => terminalsQuery.data ?? [], [terminalsQuery.data]);
  const terminalsById = useMemo(() => {
    return new Map(terminals.map((terminal) => [terminal.id, terminal]));
  }, [terminals]);

  const focusedTerminalId = latestTerminalId.current ?? terminals[0]?.id ?? null;
  const focusedTerminal = focusedTerminalId ? terminalsById.get(focusedTerminalId) : undefined;

  useEffect(() => {
    const isSmallViewport = window.matchMedia('(max-width: 768px)').matches;
    setViewMode(isSmallViewport ? 'chat' : 'grid');
  }, []);

  const persistViewMode = useCallback((_canvasId: string, nextMode: ChatViewMode) => {
    // TODO task 12.5: replace this local fallback with @/settings/settings-store.
    // settings.viewMode[canvasId] = 'grid' | 'chat'
    setViewMode(nextMode);
  }, []);

  const appendSegments = useCallback(
    (terminalId: string, segments: ParsedSegment[], isTyping: boolean, existingBubbleId?: string) => {
      const terminal = terminalsById.get(terminalId);
      const timestamp = new Date().toISOString();
      latestTerminalId.current = terminalId;

      setBubbles((current) => {
        const firstSegment = segments[0];
        if (isTyping && existingBubbleId && firstSegment) {
          return current.map((bubble) =>
            bubble.id === existingBubbleId
              ? toBubble(firstSegment, terminal, timestamp, true, existingBubbleId)
              : bubble
          );
        }

        const nextBubbles = segments.map((segment, index) =>
          toBubble(segment, terminal, timestamp, isTyping && index === segments.length - 1, createBubbleId(terminalId))
        );

        const firstBubble = nextBubbles[0];
        if (isTyping && existingBubbleId && firstBubble) {
          return current.map((bubble) => (bubble.id === existingBubbleId ? firstBubble : bubble));
        }

        return [...current, ...nextBubbles];
      });
    },
    [terminalsById]
  );

  const finalizePending = useCallback(
    (terminalId: string) => {
      const pending = pendingByTerminal.current.get(terminalId);
      if (!pending || pending.content.length === 0) return;

      if (pending.timerId !== undefined) window.clearTimeout(pending.timerId);
      const segments = segmentByAgentBoundary(pending.content, terminalId);
      appendSegments(terminalId, segments, false, pending.bubbleId);
      pendingByTerminal.current.delete(terminalId);
    },
    [appendSegments]
  );

  const handleTerminalFrame = useCallback(
    (terminalId: string, frame: ArrayBuffer) => {
      const text = textDecoder.decode(frame, { stream: true });
      if (text.length === 0) return;

      const pending = pendingByTerminal.current.get(terminalId) ?? { content: '' };
      pending.content += text;
      if (pending.timerId !== undefined) window.clearTimeout(pending.timerId);

      const segments = segmentByAgentBoundary(pending.content, terminalId);
      if (segments.length === 0) return;

      if (hasTerminator(pending.content) || segments.length > 1) {
        appendSegments(terminalId, segments, false, pending.bubbleId);
        pendingByTerminal.current.delete(terminalId);
        return;
      }

      pending.bubbleId ??= createBubbleId(terminalId);
      appendSegments(terminalId, segments, true, pending.bubbleId);
      pending.timerId = window.setTimeout(() => finalizePending(terminalId), PARTIAL_FLUSH_MS);
      pendingByTerminal.current.set(terminalId, pending);
    },
    [appendSegments, finalizePending, textDecoder]
  );

  useEffect(() => {
    const pendingBuffers = pendingByTerminal.current;
    const subscriptions = terminals.map((terminal) =>
      subscribeTerminalSocket(terminal.id, {
        onBinary: (frame) => handleTerminalFrame(terminal.id, frame),
      })
    );

    return () => {
      subscriptions.forEach((subscription) => subscription.unsubscribe());
      for (const pending of pendingBuffers.values()) {
        if (pending.timerId !== undefined) window.clearTimeout(pending.timerId);
      }
      pendingBuffers.clear();
    };
  }, [handleTerminalFrame, terminals]);

  const submitComposer = async () => {
    const message = composerValue.trim();
    if (!focusedTerminalId || message.length === 0) return;

    try {
      await goCoreClient.sendTerminalInput(focusedTerminalId, message);
      setComposerValue('');
      setQuotePrefix('');
      setStatusMessage('Mensagem enviada.');
    } catch {
      setStatusMessage('Nao foi possivel enviar a mensagem.');
    }
  };

  const handleComposerKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key !== 'Enter' || event.shiftKey) return;
    event.preventDefault();
    void submitComposer();
  };

  const handleQuoteReply = (bubble: ChatBubble) => {
    const quoted = bubble.content
      .split('\n')
      .map((line) => `> ${line}`)
      .join('\n');
    setQuotePrefix(quoted);
    setComposerValue(`${quoted}\n\n`);
    setOpenActionsBubbleId(null);
  };

  const handleCopy = async (bubble: ChatBubble) => {
    await navigator.clipboard?.writeText(bubble.content);
    setStatusMessage('Mensagem copiada.');
    setOpenActionsBubbleId(null);
  };

  const toggleViewMode = () => {
    const nextMode: ChatViewMode = viewMode === 'chat' ? 'grid' : 'chat';
    persistViewMode(sessionName, nextMode);
  };

  return (
    <section
      aria-label="Chat View"
      data-view-mode={viewMode}
      style={{
        minHeight: '100dvh',
        display: 'flex',
        flexDirection: 'column',
        background: 'var(--void)',
        color: 'var(--text-primary)',
      }}
    >
      <div
        role="toolbar"
        aria-label="Terminal surface"
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 'var(--space-4)',
          padding: 'var(--space-4) var(--space-5)',
          borderBottom: '1px solid var(--border)',
          background: 'var(--panel)',
        }}
      >
        <div>
          <div style={{ fontFamily: 'var(--font-display)', fontWeight: 700 }}>Chat View</div>
          <div style={{ color: 'var(--text-dim)', fontSize: '0.8rem' }}>{sessionName}</div>
        </div>
        <Button type="button" variant="secondary" aria-pressed={viewMode === 'chat'} onClick={toggleViewMode}>
          {viewMode === 'chat' ? 'Chat View' : 'Grid View'}
        </Button>
      </div>

      <div
        aria-live="polite"
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: 'var(--space-5)',
          display: 'flex',
          flexDirection: 'column',
          gap: 'var(--space-4)',
        }}
      >
        {terminalsQuery.isLoading && <Card>Carregando terminais...</Card>}
        {terminalsQuery.isError && <Card glow="red">Nao foi possivel carregar os terminais da sessao.</Card>}
        {!terminalsQuery.isLoading && bubbles.length === 0 && (
          <Card>Nenhuma mensagem recebida ainda.</Card>
        )}
        {bubbles.map((bubble) => (
          <BubbleCard
            key={bubble.id}
            bubble={bubble}
            actionsOpen={openActionsBubbleId === bubble.id}
            onCopy={() => void handleCopy(bubble)}
            onQuote={() => handleQuoteReply(bubble)}
            onTouchStart={(clientX) => touchStartX.current.set(bubble.id, clientX)}
            onTouchEnd={(clientX) => {
              const startX = touchStartX.current.get(bubble.id);
              if (startX !== undefined && startX - clientX >= SWIPE_THRESHOLD_PX) {
                setOpenActionsBubbleId(bubble.id);
              }
              touchStartX.current.delete(bubble.id);
            }}
          />
        ))}
      </div>

      <div
        style={{
          position: 'sticky',
          bottom: 0,
          padding: 'var(--space-4) var(--space-5)',
          paddingBottom: 'calc(var(--space-4) + env(keyboard-inset-height, 0px))',
          borderTop: '1px solid var(--border)',
          background: 'var(--panel)',
          backdropFilter: 'blur(14px)',
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 'var(--space-3)',
            marginBottom: 'var(--space-2)',
          }}
        >
          <span style={{ fontFamily: 'var(--font-display)', fontSize: '0.875rem' }}>
            {focusedTerminal?.display_name ?? focusedTerminal?.profile ?? 'Terminal'}
          </span>
          {focusedTerminal?.provider && <ProviderBadge provider={focusedTerminal.provider} />}
        </div>

        <FormField label="Mensagem" id="chat-view-composer">
          <textarea
            rows={3}
            value={composerValue}
            onChange={(event) => setComposerValue(event.target.value)}
            onKeyDown={handleComposerKeyDown}
            placeholder={quotePrefix ? 'Responder citacao...' : 'Digite e pressione Enter'}
            disabled={!focusedTerminalId}
          />
        </FormField>
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: 'var(--space-3)' }}>
          <span aria-live="polite" style={{ color: 'var(--text-dim)', fontSize: '0.75rem' }}>
            {statusMessage}
          </span>
          <Button type="button" disabled={!focusedTerminalId || composerValue.trim().length === 0} onClick={() => void submitComposer()}>
            Enviar
          </Button>
        </div>
      </div>
    </section>
  );
};

interface BubbleCardProps {
  bubble: ChatBubble;
  actionsOpen: boolean;
  onCopy: () => void;
  onQuote: () => void;
  onTouchStart: (clientX: number) => void;
  onTouchEnd: (clientX: number) => void;
}

const BubbleCard: React.FC<BubbleCardProps> = ({
  bubble,
  actionsOpen,
  onCopy,
  onQuote,
  onTouchStart,
  onTouchEnd,
}) => {
  return (
    <div
      onTouchStart={(event) => onTouchStart(event.changedTouches[0]?.clientX ?? 0)}
      onTouchEnd={(event) => onTouchEnd(event.changedTouches[0]?.clientX ?? 0)}
      style={{ position: 'relative' }}
    >
      <Card
        glow={bubble.kind === 'system' ? 'red' : 'cyan'}
        style={{
          maxWidth: '760px',
          marginLeft: bubble.kind === 'tool_result' ? 'var(--space-7)' : 0,
        }}
      >
        <header
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 'var(--space-3)',
            marginBottom: 'var(--space-3)',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)', flexWrap: 'wrap' }}>
            <strong>{bubble.displayName}</strong>
            {bubble.provider && <ProviderBadge provider={bubble.provider} />}
            <Badge variant={kindToBadgeVariant(bubble.kind)}>{bubble.kind}</Badge>
          </div>
          <time dateTime={bubble.timestamp} style={{ color: 'var(--text-dim)', fontSize: '0.75rem' }}>
            {new Date(bubble.timestamp).toLocaleTimeString()}
          </time>
        </header>
        <Prose style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{bubble.content}</Prose>
        {bubble.isTyping && (
          <div style={{ color: 'var(--cyan)', fontSize: '0.8rem', marginTop: 'var(--space-3)' }}>
            digitando...
          </div>
        )}
      </Card>
      {actionsOpen && (
        <div
          role="menu"
          style={{
            position: 'absolute',
            right: 'var(--space-3)',
            top: 'var(--space-3)',
            display: 'flex',
            gap: 'var(--space-2)',
            padding: 'var(--space-2)',
            background: 'var(--panel)',
            border: '1px solid var(--border-accent)',
            borderRadius: 'var(--radius-button)',
          }}
        >
          <Button type="button" variant="ghost" role="menuitem" onClick={onCopy}>
            Copiar
          </Button>
          <Button type="button" variant="ghost" role="menuitem" onClick={onQuote}>
            Citar
          </Button>
        </div>
      )}
    </div>
  );
};

const ProviderBadge: React.FC<{ provider: ProviderType }> = ({ provider }) => {
  return <Badge variant="processing">{provider}</Badge>;
};

function toBubble(
  segment: ParsedSegment,
  terminal: Terminal | undefined,
  timestamp: string,
  isTyping: boolean,
  id: string
): ChatBubble {
  return {
    id,
    terminalId: segment.terminalId,
    content: segment.content,
    kind: segment.kind,
    displayName: terminal?.display_name ?? terminal?.profile ?? segment.terminalId,
    provider: terminal?.provider,
    timestamp,
    isTyping,
  };
}

function createBubbleId(terminalId: string): string {
  return `${terminalId}-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function hasTerminator(buffer: string): boolean {
  return /(?:\r?\n|\r)$/.test(buffer);
}

function kindToBadgeVariant(kind: ParsedSegment['kind']): React.ComponentProps<typeof Badge>['variant'] {
  if (kind === 'tool_call') return 'waiting_user_answer';
  if (kind === 'tool_result') return 'completed';
  if (kind === 'system') return 'error';
  return 'idle';
}

export default ChatView;
