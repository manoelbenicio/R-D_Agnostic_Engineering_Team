import { useEffect } from 'react';
import { useKeyStore } from './store';
import { ProviderType, PROVIDERS_REGISTRY } from './registry';

export function useProviderModels(provider: ProviderType): string[] {
  const init = useKeyStore((s) => s.init);
  const initialized = useKeyStore((s) => s.initialized);
  const cachedModels = useKeyStore((s) => s.cachedModels);

  useEffect(() => {
    if (!initialized) {
      void init();
    }
  }, [init, initialized]);

  return cachedModels[provider] || PROVIDERS_REGISTRY.find((p) => p.id === provider)?.defaultModels || [];
}

export default useProviderModels;
