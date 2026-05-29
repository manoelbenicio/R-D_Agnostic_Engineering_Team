import type { TerminalStatus } from '@/api';

export function mapTerminalStatus(
  status: TerminalStatus
): 'idle' | 'processing' | 'completed' | 'waiting_user_answer' | 'error' {
  switch (status) {
    case 'starting':
    case 'processing':
      return 'processing';
    case 'idle':
      return 'idle';
    case 'exited':
      return 'completed';
    case 'error':
    case 'offline':
      return 'error';
    default:
      return 'idle';
  }
}
