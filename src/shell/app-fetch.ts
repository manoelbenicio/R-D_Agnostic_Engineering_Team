/**
 * Auth-Aware Fetch Boundary wrapper.
 * For Milestone 1, this is a thin pass-through wrapper.
 * Future milestones (M2+) will attach Firebase JWT tokens here.
 */
export async function appFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  // TODO: Attach Firebase JWT token here in Milestone 2
  return fetch(input, init);
}
