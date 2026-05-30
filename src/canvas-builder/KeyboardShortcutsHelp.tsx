import { useEffect } from 'react';
import './keyboard-shortcuts-help.css';

export interface KeyboardShortcutsHelpProps {
  isOpen: boolean;
  onClose: () => void;
}

const SHORTCUTS = [
  ['Ctrl+Shift+F', 'Toggle fullscreen'],
  ['Ctrl+0', 'Fit view'],
  ['Ctrl+=', 'Zoom in'],
  ['Ctrl+-', 'Zoom out'],
  ['?', 'Show this help'],
  ['Delete', 'Remove selected node'],
  ['Escape', 'Close panel / exit fullscreen'],
] as const;

export function KeyboardShortcutsHelp({ isOpen, onClose }: KeyboardShortcutsHelpProps) {
  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div className="keyboard-shortcuts-overlay" role="presentation" onMouseDown={onClose}>
      <section
        className="keyboard-shortcuts-help"
        role="dialog"
        aria-modal="true"
        aria-labelledby="keyboard-shortcuts-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header className="keyboard-shortcuts-header">
          <div>
            <span>Canvas controls</span>
            <h2 id="keyboard-shortcuts-title">Keyboard Shortcuts</h2>
          </div>
          <button type="button" aria-label="Close keyboard shortcuts" onClick={onClose}>
            x
          </button>
        </header>

        <div className="keyboard-shortcuts-grid">
          {SHORTCUTS.map(([shortcut, action]) => (
            <div className="keyboard-shortcuts-row" key={shortcut}>
              <kbd>{shortcut}</kbd>
              <span>{action}</span>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}

export default KeyboardShortcutsHelp;
