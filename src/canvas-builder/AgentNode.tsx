import React from 'react';
import { Handle, NodeProps, Position } from '@xyflow/react';
import { Card, StatusBadge } from '@/design-system';
import { CanvasNode } from '@/shared/canvas-types';

type AgentNodeData = CanvasNode['data'];

export const AgentNode: React.FC<NodeProps> = ({ data, selected }) => {
  const agentData = data as unknown as AgentNodeData;
  const badgeStatus = agentData.is_entry_point ? 'completed' : 'idle';

  return (
    <Card
      className={`canvas-agent-node ${selected ? 'is-selected' : ''}`}
      glow={selected ? 'cyan' : 'none'}
      style={{
        width: 230,
        padding: 'var(--space-4)',
        borderColor: selected ? 'var(--cyan)' : 'var(--border)',
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
      <Handle type="source" position={Position.Right} className="canvas-node-handle" />
    </Card>
  );
};

export type { AgentNodeData };
export default AgentNode;
