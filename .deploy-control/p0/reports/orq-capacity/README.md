# ORQ Capacity Dashboard

Offline, self-contained capacity dashboard for ORQ1 and ORQ2. No CDN, backend, telemetry beacon or automatic refresh is used.

## Open

Open `orq-capacity-dashboard.html` in a modern browser. The initial verified snapshot is embedded, so `file://` works without a web server.

## Update only when a new photo is wanted

1. Produce a JSON file conforming to `orq-capacity-snapshot-v1.schema.json`.
2. Open the dashboard and click **Upload snapshot**.
3. Choose one or multiple snapshot JSON files. A bundle exported by the dashboard is also accepted.
4. The browser validates the schema marker, deduplicates by `snapshot_id`, stores the history in local storage and selects the newest snapshot.

The HTML never connects to ORQ1, ORQ2 or AWS. Upload is the only data-update path.

## Premium workspaces

- **Visão macro**: executive verdict, portfolio exposure, host capacity and prioritized decision board.
- **Visão micro**: per-host telemetry, provisioned shape, storage consumers, observed processes and engineering actions.
- **Pressure lab**: threshold-normalized radar profiles plus an accessible tabular signal matrix.
- **Histórico**: local trends and selected-versus-baseline deltas.
- **Governança**: quality grade, limitations, lineage, references and the portable data contract.

The left menu is keyboard navigable; `Alt+1` through `Alt+5` switch workspaces. The dashboard includes visible focus states, a skip link, responsive single-column layouts, scroll-safe tables and `prefers-reduced-motion` support. All icons, charts, styles and scripts are embedded—there are no external assets.

## Vendor visual themes

Use the **Tema visual do fornecedor** selector in the header. The choice is saved only in local browser storage and does not change snapshot data.

- **AWS Cloudscape** — default high-visibility light theme with white/grayscale surfaces and AWS action blue.
- **Microsoft Azure** — Fluent-inspired neutral canvas, Segoe UI stack and Azure blue selection.
- **Google Cloud** — Material-inspired tonal surfaces, rounded shapes and Google blue actions.
- **Oracle Cloud** — Redwood-inspired warm neutrals, compact radii, teal interaction and Oracle-red identity.
- **IBM Carbon** — layered white/gray surfaces, IBM blue interaction and square geometry.

The themes are offline adaptations built from official vendor design-system guidance and public token repositories; no vendor CSS, fonts, logos, scripts or network assets are imported.

## Versioning and portability

- Snapshot schema: `orq-capacity-snapshot/v1`
- Bundle schema: `orq-capacity-bundle/v1`
- Each snapshot has an immutable `snapshot_id`, `data_version` and `captured_at`.
- **Export selected** downloads one portable snapshot.
- **Export history** downloads a bundle containing every imported snapshot.
- **Print / PDF** produces an executive report from the current selection.
- Browser history is local to the browser profile. Export a bundle before clearing site data or moving machines.

## Files

- `orq-capacity-dashboard.html`: premium offline viewer and history manager.
- `orq-capacity-snapshot-v1.schema.json`: JSON Schema for future snapshots.
- `snapshots/*.json`: immutable source snapshots.

## Safety

Snapshots must contain aggregate metrics and paths only. Do not include credentials, tokens, environment values, process arguments, cookies or secret contents. Capacity recommendations do not authorize EC2/EBS changes or deletion. Docker/cache cleanup must remain target-specific and separately approved.
