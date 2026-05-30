import React from 'react';

interface ProviderIconProps {
  provider: string;
  size?: number;
  className?: string;
}

/**
 * Renders a colored icon/badge for each CLI provider.
 * Uses Unicode/emoji as zero-dependency icons.
 */
export const ProviderIcon: React.FC<ProviderIconProps> = ({ provider, size = 16, className }) => {
  const iconMap: Record<string, { emoji: string; color: string; label: string }> = {
    claude_code: { emoji: '🟠', color: '#E87B35', label: 'Claude Code' },
    codex: { emoji: '🟢', color: '#10A37F', label: 'Codex' },
    gemini_cli: { emoji: '🔵', color: '#4285F4', label: 'Gemini CLI' },
    kiro_cli: { emoji: '🟣', color: '#9B59B6', label: 'Kiro CLI' },
  };

  const icon = iconMap[provider] || { emoji: '⚪', color: '#888888', label: provider };

  return (
    <span
      className={`provider-icon ${className || ''}`}
      style={{ fontSize: size, color: icon.color }}
      title={icon.label}
      aria-label={icon.label}
    >
      {icon.emoji}
    </span>
  );
};

/** Get the display label for a CLI provider */
export function getProviderLabel(provider: string): string {
  const labels: Record<string, string> = {
    claude_code: 'Claude Code',
    codex: 'Codex',
    gemini_cli: 'Gemini CLI',
    kiro_cli: 'Kiro CLI',
  };
  return labels[provider] || provider;
}

/** Get the brand color for a CLI provider */
export function getProviderColor(provider: string): string {
  const colors: Record<string, string> = {
    claude_code: '#E87B35',
    codex: '#10A37F',
    gemini_cli: '#4285F4',
    kiro_cli: '#9B59B6',
  };
  return colors[provider] || '#888888';
}
