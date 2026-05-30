import React from 'react';
import { Handle, NodeProps, Position } from '@xyflow/react';
import { Card, StatusBadge } from '@/design-system';
import { CanvasNode } from '@/shared/canvas-types';
import { useSessionStore } from '@/api/session-store';

type AgentNodeData = CanvasNode['data'];

export const AgentNode: React.FC<NodeProps> = ({ data, selected }) => {
  const agentData = data as unknown as AgentNodeData;
  const badgeStatus = agentData.is_entry_point ? 'completed' : 'idle';
  const agentColor = agentData.color || 'var(--cyan)';
  const session = useSessionStore((state) => (
    agentData.session_id ? state.getSession(agentData.session_id) : undefined
  ));
  const oauthTitle = session ? `OAuth: ${session.account_email}` : 'OAuth session bound';

  return (
    <Card
      className={`canvas-agent-node ${selected ? 'is-selected' : ''}`}
      glow={selected ? 'cyan' : 'none'}
      style={{
        width: 230,
        padding: 'var(--space-4)',
        borderColor: agentColor,
        boxShadow: selected ? `0 0 24px ${agentColor}40` : undefined,
      }}
    >
      <Handle type="target" position={Position.Left} className="canvas-node-handle" />
      <div className="canvas-agent-node-header">
        <strong>{agentData.display_name}</strong>
        <StatusBadge
          status={badgeStatus}
          label={agentData.is_entry_point ? 'Entry' : agentData.role}
          style={{ flexShrink: 0 }}
        />
      </div>
      <div className="canvas-agent-node-meta">
        <span>{agentData.provider ?? 'No provider'}</span>
        <span>{agentData.model ?? 'No model'}</span>
      </div>
      {agentData.session_id ? (
        <span
          title={oauthTitle}
          aria-label={oauthTitle}
          style={{
            position: 'absolute',
            right: 8,
            bottom: 8,
            padding: '2px 6px',
            border: '1px solid var(--cyan-edge)',
            borderRadius: 999,
            background: 'var(--surface-overlay)',
            color: 'var(--cyan)',
            fontFamily: 'var(--font-mono)',
            fontSize: '0.62rem',
            fontWeight: 700,
            letterSpacing: 0,
            opacity: 0.82,
            pointerEvents: 'auto',
          }}
        >
          OAuth
        </span>
      ) : null}
      <Handle type="source" position={Position.Right} className="canvas-node-handle" />
    </Card>
  );
};

export type { AgentNodeData };
export default AgentNode;
