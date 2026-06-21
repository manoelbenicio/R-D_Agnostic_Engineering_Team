import { useQuery } from '@tanstack/react-query';
import { goCoreClient } from './go-core-client';  // CRIT-003.8

/**
 * Installed (and OAuth-authenticated) CLI providers reported by the GO Core server
 * runtime via `GET /agents/providers`. These are the `name`s whose CLI is
 * present on the server (e.g. `codex`, `kiro_cli`) — the OAuth path that does
 * NOT require a BYOK API key.
 *
 * Exposes `refetch` so the UI can offer a "re-sync" button: after the user
 * re-logs a CLI on the host (rotating its OAuth token), re-polling confirms
 * the CLI is still authenticated. The token itself is read by the CLI at
 * launch time from the bind-mounted credential dir — relaunching a session
 * (redeploy) is what actually picks up a rotated token.
 */
export function useInstalledCliProviders() {
  const query = useQuery({
    queryKey: ['goCore', 'installed-providers'],
    queryFn: async () => {
      const providers = await goCoreClient.listProviders();
      return providers.filter((p) => p.installed).map((p) => String(p.name));
    },
    staleTime: 10_000,
  });

  return {
    installed: query.data ?? [],
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    refetch: query.refetch,
  };
}

export default useInstalledCliProviders;
