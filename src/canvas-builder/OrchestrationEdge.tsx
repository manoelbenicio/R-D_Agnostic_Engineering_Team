import React from 'react';
import {
  BaseEdge,
  EdgeLabelRenderer,
  EdgeProps,
  getBezierPath,
} from '@xyflow/react';
import { OrchestrationType } from '@/shared/canvas-types';

export type OrchestrationEdgeData = {
  [key: string]: unknown;
  orchestrationType: OrchestrationType;
  onTypeChange?: (edgeId: string, type: OrchestrationType) => void;
  onDelete?: (edgeId: string) => void;
  sourceColor?: string;
};

const EDGE_OPTIONS: OrchestrationType[] = ['handoff', 'assign', 'send_message'];

const EDGE_STROKE: Record<OrchestrationType, string> = {
  handoff: '',
  assign: '8 6',
  send_message: '2 6',
};

const EDGE_COLOR: Record<OrchestrationType, string> = {
  handoff: 'var(--cyan)',
  assign: '#f59e0b',
  send_message: '#a78bfa',
};

export const OrchestrationEdge: React.FC<EdgeProps> = ({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  markerEnd,
  data,
  selected,
}) => {
  const edgeData = data as OrchestrationEdgeData | undefined;
  const edgeType = edgeData?.orchestrationType ?? 'handoff';
  const edgeColor = edgeData?.sourceColor || EDGE_COLOR[edgeType];
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const handleDelete = (event: React.MouseEvent) => {
    event.stopPropagation();
    edgeData?.onDelete?.(id);
  };

  return (
    <>
      {/* Invisible wider hit area for easier interaction */}
      <path
        d={edgePath}
        fill="none"
        stroke="transparent"
        strokeWidth={20}
        style={{ pointerEvents: 'stroke', cursor: 'pointer' }}
      />
      <BaseEdge
        path={edgePath}
        markerEnd={markerEnd}
        style={{
          stroke: selected ? '#fff' : edgeColor,
          strokeWidth: selected ? 3 : 2,
          strokeDasharray: EDGE_STROKE[edgeType],
          transition: 'stroke-dasharray 80ms ease, opacity 80ms ease, stroke 120ms ease',
        }}
      />
      <EdgeLabelRenderer>
        <div
          className="canvas-edge-controls"
          style={{
            transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
          }}
        >
          <select
            className="canvas-edge-label"
            value={edgeType}
            onChange={(event) =>
              edgeData?.onTypeChange?.(id, event.target.value as OrchestrationType)
            }
            style={{ color: edgeColor, borderColor: edgeColor }}
            aria-label="Edge type"
          >
            {EDGE_OPTIONS.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
          <button
            className="canvas-edge-delete"
            onClick={handleDelete}
            title="Delete connection"
            aria-label="Delete edge"
          >
            ✕
          </button>
        </div>
      </EdgeLabelRenderer>
    </>
  );
};

export default OrchestrationEdge;
