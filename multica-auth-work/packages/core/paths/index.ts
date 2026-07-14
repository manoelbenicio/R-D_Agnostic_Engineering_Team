export { paths, isGlobalPath } from "./paths";
export type { WorkspacePaths } from "./paths";
export { DESKTOP_RELEASES_URL } from "./external";
export { RESERVED_SLUGS, isReservedSlug } from "./reserved-slugs";
export { resolvePostAuthDestination, useHasOnboarded } from "./resolve";
export {
  WorkspaceSlugProvider,
  useWorkspaceSlug,
  useRequiredWorkspaceSlug,
  useCurrentWorkspace,
  useWorkspacePaths,
} from "./hooks";
