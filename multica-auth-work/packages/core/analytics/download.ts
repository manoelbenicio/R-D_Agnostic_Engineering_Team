/**
 * Download funnel instrumentation.
 *
 * Complements the onboarding path-selection events by distinguishing
 * the Welcome and runtime-choice entry points to the public releases page.
 *
 * Event names and property shapes are governed by docs/analytics.md;
 * keep the two in sync when adding a new source or field.
 */

import { captureEvent, setPersonProperties } from "./index";

/**
 * Where the user clicked a CTA that points at the public releases page.
 */
export type DownloadIntentSource = "welcome" | "step3";

/**
 * Fires when a user clicks an onboarding CTA for the public releases page. We
 * also write `platform_preference` to person properties so the backend
 * can segment subsequent events — same convention the Step 3 handler
 * already uses (see `step-platform-fork.tsx`).
 */
export function captureDownloadIntent(source: DownloadIntentSource): void {
  captureEvent("download_intent_expressed", {
    source,
  });
  setPersonProperties({ platform_preference: "desktop" });
}
