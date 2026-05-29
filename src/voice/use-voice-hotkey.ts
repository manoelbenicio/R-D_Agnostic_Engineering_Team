import { useEffect } from 'react';
import { useVoiceStore } from './store';

export function useVoiceHotkey(): void {
  const isOpen = useVoiceStore((s) => s.isOpen);
  const setOpen = useVoiceStore((s) => s.setOpen);
  const reset = useVoiceStore((s) => s.reset);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const isKeyV = e.key === 'v' || e.key === 'V';
      const isModifier = e.ctrlKey || e.metaKey;
      if (isModifier && e.shiftKey && isKeyV) {
        e.preventDefault();
        const nextOpen = !isOpen;
        if (!nextOpen) {
          reset();
        }
        setOpen(nextOpen);
      }
    };

    if (typeof window !== 'undefined') {
      window.addEventListener('keydown', handleKeyDown);
    }
    return () => {
      if (typeof window !== 'undefined') {
        window.removeEventListener('keydown', handleKeyDown);
      }
    };
  }, [isOpen, setOpen, reset]);
}

export default useVoiceHotkey;
