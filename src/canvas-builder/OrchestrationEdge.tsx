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
};

const EDGE_OPTIONS: OrchestrationType[] = ['handoff', 'assign', 'send_message'];

const EDGE_STROKE: Record<OrchestrationType, string> = {
  handoff: '',
  assign: '8 6',
  send_message: '2 6',
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
}) => {
  const edgeData = data as OrchestrationEdgeData | undefined;
  const edgeType = edgeData?.orchestrationType ?? 'handoff';
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  return (
    <>
      <BaseEdge
        path={edgePath}
        markerEnd={markerEnd}
        style={{
          stroke: 'var(--cyan)',
          strokeWidth: 2,
          strokeDasharray: EDGE_STROKE[edgeType],
          transition: 'stroke-dasharray 80ms ease, opacity 80ms ease',
        }}
      />
      <EdgeLabelRenderer>
        <select
          className="canvas-edge-label"
          value={edgeType}
          onChange={(event) =>
            edgeData?.onTypeChange?.(id, event.target.value as OrchestrationType)
          }
          style={{
            transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
          }}
          aria-label="Edge type"
        >
          {EDGE_OPTIONS.map((option) => (
            <option key={option} value={option}>
              {option}
            </option>
          ))}
        </select>
      </EdgeLabelRenderer>
    </>
  );
};

export default OrchestrationEdge;
