import type { ProviderType } from '@/api';

export type ParsedSegmentKind = 'output' | 'tool_call' | 'tool_result' | 'system';

export interface ParsedSegment {
  terminalId: string;
  content: string;
  kind: ParsedSegmentKind;
}

export interface ChatBubble extends ParsedSegment {
  id: string;
  displayName: string;
  provider?: ProviderType;
  timestamp: string;
  isTyping: boolean;
}

export type ChatViewMode = 'grid' | 'chat';
