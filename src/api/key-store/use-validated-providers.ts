import { useEffect } from 'react';
import { useKeyStore } from './store';
import { ProviderType } from './registry';

export function useValidatedProviders(): ProviderType[] {
  const init = useKeyStore((s) => s.init);
  const validated = useKeyStore((s) => s.validated);
  const initialized = useKeyStore((s) => s.initialized);

  useEffect(() => {
    if (!initialized) {
      void init();
    }
  }, [init, initialized]);

  return validated;
}

export default useValidatedProviders;
