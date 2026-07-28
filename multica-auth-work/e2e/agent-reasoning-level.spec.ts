import "./env";
import { test, expect } from "@playwright/test";
import pg from "pg";
import { createTestApi, loginAsDefault } from "./helpers";
import type { TestApiClient } from "./fixtures";

const DATABASE_URL =
  process.env.DATABASE_URL ??
  "postgres://multica:multica@localhost:5432/multica?sslmode=disable";

type ReasoningFixture = {
  agentId: string;
  agentName: string;
  runtimeId: string;
};

async function seedReasoningAgent(api: TestApiClient): Promise<ReasoningFixture> {
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
    const agentName = `E2E Reasoning Agent ${suffix}`;
    const runtime = await client.query<{ id: string }>(
      `INSERT INTO agent_runtime (
         workspace_id, daemon_id, name, runtime_mode, provider, status,
         device_info, metadata, last_seen_at, owner_id
       ) VALUES ($1, NULL, $2, 'cloud', 'codex', 'online', $2, '{}'::jsonb, now(), $3)
       RETURNING id::text`,
      [workspace.id, `E2E Reasoning Runtime ${suffix}`, userId],
    );
    const runtimeId = runtime.rows[0]!.id;
    const agent = await client.query<{ id: string }>(
      `INSERT INTO agent (
         workspace_id, name, description, runtime_mode, runtime_config,
         runtime_id, visibility, max_concurrent_tasks, owner_id, model
       ) VALUES ($1, $2, '', 'cloud', '{}'::jsonb, $3, 'workspace', 1, $4, $5)
       RETURNING id::text`,
      [workspace.id, agentName, runtimeId, userId, "e2e-reasoner"],
    );
    const fixture = { agentId: agent.rows[0]!.id, agentName, runtimeId };
    await client.query("COMMIT");
    return fixture;
  } catch (error) {
    await client.query("ROLLBACK");
    throw error;
  } finally {
    await client.end();
  }
}

async function removeReasoningFixture(fixture: ReasoningFixture | null) {
  if (!fixture) return;
  const client = new pg.Client(DATABASE_URL);
  await client.connect();
  try {
    await client.query(`DELETE FROM agent WHERE id = $1`, [fixture.agentId]);
    await client.query(`DELETE FROM agent_runtime WHERE id = $1`, [fixture.runtimeId]);
  } finally {
    await client.end();
  }
}

test.use({ locale: "en-US" });

test.describe("Agent reasoning level", () => {
  let api: TestApiClient;
  let fixture: ReasoningFixture | null = null;

  test.beforeEach(async ({ page }) => {
    api = await createTestApi();
    fixture = await seedReasoningAgent(api);
    await loginAsDefault(page);
  });

  test.afterEach(async () => {
    await removeReasoningFixture(fixture);
    fixture = null;
    await api.cleanup();
  });

  test("renders runtime-supported levels and persists the selected value after reload", async ({
    page,
  }) => {
    const now = new Date().toISOString();
    const corsHeaders = {
      "Access-Control-Allow-Origin": new URL(page.url()).origin,
      "Access-Control-Allow-Credentials": "true",
    };
    // UI-only contract: this deterministic catalog proves rendering and persistence.
    // It does not validate the backend/daemon model-discovery response contract.
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
          id: "33333333-3333-4333-8333-333333333333",
          runtime_id: fixture!.runtimeId,
          status: "completed",
          supported: true,
          models: [
            {
              id: "e2e-reasoner",
              label: "E2E Reasoner",
              provider: "codex",
              default: true,
              thinking: {
                supported_levels: [
                  { value: "low", label: "Low", description: "Light reasoning" },
                  { value: "high", label: "High", description: "Deep reasoning" },
                ],
              },
            },
          ],
          created_at: now,
          updated_at: now,
        }),
      });
    });

    const workspace = (await api.getWorkspaces())[0]!;
    await page.goto(`/${workspace.slug}/agents/${fixture!.agentId}`, {
      waitUntil: "domcontentloaded",
    });
    await expect(page.getByText(fixture!.agentName, { exact: true })).toBeVisible();

    const thinkingTrigger = page.getByRole("button", {
      name: "Thinking · Follow CLI config",
    });
    await expect(thinkingTrigger).toBeVisible();
    await thinkingTrigger.click();

    const updateResponse = page.waitForResponse(
      (response) =>
        response.url().endsWith(`/api/agents/${fixture!.agentId}`) &&
        response.request().method() === "PUT",
    );
    await page.getByText("High", { exact: true }).click();
    const response = await updateResponse;
    expect(response.ok()).toBe(true);
    expect(response.request().postDataJSON()).toMatchObject({ thinking_level: "high" });

    const client = new pg.Client(DATABASE_URL);
    await client.connect();
    try {
      const persisted = await client.query<{ thinking_level: string | null }>(
        `SELECT thinking_level FROM agent WHERE id = $1`,
        [fixture!.agentId],
      );
      expect(persisted.rows[0]?.thinking_level).toBe("high");
    } finally {
      await client.end();
    }

    await page.reload({ waitUntil: "domcontentloaded" });
    await expect(
      page.getByRole("button", { name: "Thinking · High" }),
    ).toBeVisible();
  });
});
