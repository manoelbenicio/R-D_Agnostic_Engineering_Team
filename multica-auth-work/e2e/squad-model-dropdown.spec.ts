import "./env";
import { test, expect } from "@playwright/test";
import pg from "pg";
import { createTestApi, loginAsDefault } from "./helpers";
import type { TestApiClient } from "./fixtures";

const DATABASE_URL =
  process.env.DATABASE_URL ??
  "postgres://multica:multica@localhost:5432/multica?sslmode=disable";

type SquadFixture = {
  runtimeId: string;
  runtimeName: string;
  agentIds: string[];
  leaderName: string;
  memberName: string;
};

async function seedSquadAgents(api: TestApiClient): Promise<SquadFixture> {
  const workspace = (await api.getWorkspaces())[0];
  if (!workspace) throw new Error("E2E workspace missing");
  api.setWorkspaceId(workspace.id);
  api.setWorkspaceSlug(workspace.slug);

  const client = new pg.Client(DATABASE_URL);
  await client.connect();
  await client.query("BEGIN");
  try {
    const user = await client.query<{ id: string }>(
      `SELECT id::text FROM "user" WHERE email = $1`,
      [api.getEmail()],
    );
    const userId = user.rows[0]?.id;
    if (!userId) throw new Error("E2E user missing");
    const suffix = Date.now();
    const runtimeName = `E2E Squad Runtime ${suffix}`;
    const leaderName = `E2E Squad Leader ${suffix}`;
    const memberName = `E2E Squad Member ${suffix}`;
    const runtime = await client.query<{ id: string }>(
      `INSERT INTO agent_runtime (
         workspace_id, daemon_id, name, runtime_mode, provider, status,
         device_info, metadata, last_seen_at, owner_id
       ) VALUES ($1, NULL, $2, 'cloud', 'codex', 'online', $2, '{}'::jsonb, now(), $3)
       RETURNING id::text`,
      [workspace.id, runtimeName, userId],
    );
    const runtimeId = runtime.rows[0]!.id;
    const agentIds: string[] = [];
    for (const name of [leaderName, memberName]) {
      const agent = await client.query<{ id: string }>(
        `INSERT INTO agent (
           workspace_id, name, description, runtime_mode, runtime_config,
           runtime_id, visibility, max_concurrent_tasks, owner_id
         ) VALUES ($1, $2, '', 'cloud', '{}'::jsonb, $3, 'workspace', 1, $4)
         RETURNING id::text`,
        [workspace.id, name, runtimeId, userId],
      );
      agentIds.push(agent.rows[0]!.id);
    }
    const fixture = { runtimeId, runtimeName, agentIds, leaderName, memberName };
    await client.query("COMMIT");
    return fixture;
  } catch (error) {
    await client.query("ROLLBACK");
    throw error;
  } finally {
    await client.end();
  }
}

async function removeSquadFixture(fixture: SquadFixture | null) {
  if (!fixture) return;
  const client = new pg.Client(DATABASE_URL);
  await client.connect();
  try {
    await client.query(`DELETE FROM squad WHERE leader_id = ANY($1::uuid[])`, [
      fixture.agentIds,
    ]);
    await client.query(`DELETE FROM agent WHERE id = ANY($1::uuid[])`, [
      fixture.agentIds,
    ]);
    await client.query(`DELETE FROM agent_runtime WHERE id = $1`, [fixture.runtimeId]);
  } finally {
    await client.end();
  }
}

test.describe("Squad model dropdown", () => {
  let api: TestApiClient;
  let fixture: SquadFixture | null = null;

  test.beforeEach(async ({ page }) => {
    api = await createTestApi();
    fixture = await seedSquadAgents(api);
    await loginAsDefault(page);
  });

  test.afterEach(async () => {
    await removeSquadFixture(fixture);
    fixture = null;
    await api.cleanup();
  });

  test("creates a squad with a member and exposes a non-empty runtime model catalog", async ({
    page,
  }) => {
    const now = new Date().toISOString();
    const corsHeaders = {
      "Access-Control-Allow-Origin": new URL(page.url()).origin,
      "Access-Control-Allow-Credentials": "true",
    };
    await page.route(`**/api/runtimes/${fixture!.runtimeId}/models`, async (route) => {
      if (route.request().method() === "OPTIONS") {
        await route.continue();
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        headers: corsHeaders,
        body: JSON.stringify({
          id: "44444444-4444-4444-8444-444444444444",
          runtime_id: fixture!.runtimeId,
          status: "completed",
          supported: true,
          models: [
            { id: "e2e-fast", label: "E2E Fast", provider: "codex", default: true },
            { id: "e2e-deep", label: "E2E Deep", provider: "codex" },
          ],
          created_at: now,
          updated_at: now,
        }),
      });
    });

    const workspace = (await api.getWorkspaces())[0]!;
    await page.goto(`/${workspace.slug}/squads`, { waitUntil: "domcontentloaded" });
    await expect(page.getByRole("heading", { name: "Squads" })).toBeVisible();
    await page.getByRole("button", { name: "New Squad" }).click();

    const squadName = `E2E Browser Squad ${Date.now()}`;
    await page.getByPlaceholder("e.g. Frontend Team").fill(squadName);
    await page.getByText("Select a leader agent", { exact: true }).click();
    await page.getByText(fixture!.leaderName, { exact: true }).last().click();

    await page.getByRole("combobox").click();
    await page.getByText(fixture!.memberName, { exact: true }).last().click();
    await page.keyboard.press("Escape");

    await page.getByRole("button", { name: "Create Squad", exact: true }).click();
    await page.waitForURL(/\/squads\/[\w-]+/);
    await expect(page.getByText(squadName, { exact: true })).toBeVisible();
    await expect(page.getByText(fixture!.leaderName, { exact: true })).toBeVisible();
    await expect(page.getByText(fixture!.memberName, { exact: true })).toBeVisible();

    await page.getByRole("button", { name: "Create Agent" }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expect(page.getByText(fixture!.runtimeName, { exact: true })).toBeVisible();
    await page.getByText("Default (provider)", { exact: true }).click();

    const modelOptions = page
      .locator("button")
      .filter({ hasText: /^E2E (Fast|Deep)/ });
    await expect.poll(async () => modelOptions.count()).toBeGreaterThan(0);
  });
});
