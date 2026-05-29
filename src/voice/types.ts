export interface CreateCanvasIntent {
  name: string;
  nodes: Array<{
    display_name: string;
    role: 'supervisor' | 'developer' | 'reviewer' | 'custom';
    provider: string; // e.g. kiro_cli, claude_code, etc.
  }>;
  edges: Array<{
    from: string; // References display_name of source node
    to: string;   // References display_name of target node
    type: 'handoff' | 'assign' | 'send_message';
  }>;
  confidence?: number;
}

export type VoiceState = 'idle' | 'listening' | 'processing' | 'confirming' | 'error';

export interface VoiceCaptureHandlers {
  onPartial?: (text: string) => void;
  onFinal?: (text: string) => void;
  onError?: (error: unknown) => void;
  onEnd?: () => void;
}

// eslint-disable-next-line agentverse/no-sideways-capability-imports
export type { CanvasDocument, CanvasNode, CanvasEdge, ProviderType } from '@/shared/canvas-types';
