# API Key Storage Security Model (v1)

This document outlines the design decisions, residual risks (R9), and telemetry safety policies for BYOK (Bring Your Own Key) credential storage in AgentVerse v1.

## 1. plaintext IndexedDB Persistence (Local Dev Only)
For the v1 milestone, API keys are persisted as plaintext values within the browser's `IndexedDB` under the `provider_keys` object store.
- **Scope**: Designed strictly for local development environments (`localhost`).
- **Rationale**: Reduces architectural overhead for bootstrap verification before cloud deployment layers are introduced.
- **Access Bounds**: Access is isolated via browser sandboxing and the Same-Origin Policy (SOP). Only scripts served from the same origin can query the local IndexedDB.

## 2. Telemetry and Console Redaction Policy
To prevent credential leaks, the system enforces a zero-exposure policy for raw credentials:
- **No Console Logs**: Plainttext keys are never outputted to console logs.
- **Sanitized Errors**: API exceptions caught during validation redact raw key values before reporting error bodies to user alerts or console objects.
- **Telemetry Protection**: Telemetry logs and telemetry payloads are strictly validated to exclude auth headers or raw input values.

## 3. Residual Exposure (Risk R9 - Dev-Tools Exposure)
Because keys reside in plaintext in client memory and IndexedDB, any user or script with local console access (such as browser extensions with broad permissions or local developer tools) has residual exposure to these credentials. 
- **Mitigation**: Users are advised to only run AgentVerse in secure browser profiles free from untrusted extensions.

## 4. Post-Launch Roadmap (Encrypted Cloud Syncing)
In post-launch milestones (Milestone 2+), the storage backend will transition to a secure remote persistence model:
- **Encrypted Firestore**: Credentials will be stored in an encrypted Firebase Firestore instance.
- **Key Storage Proxy**: Future calls will route through a validation proxy running in a gated cloud environment, preventing raw credentials from ever landing in client-side state.
