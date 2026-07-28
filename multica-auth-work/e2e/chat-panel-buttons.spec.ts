import "./env";
import { test, expect } from "@playwright/test";
import pg from "pg";
import { createTestApi, loginAsDefault } from "./helpers";
import type { TestApiClient } from "./fixtures";

const DATABASE_URL =
  process.env.DATABASE_URL ??
  "postgres://multica:multica@localhost:5432/multica?sslmode=disable";

type PanelFixture = {
  agentId: string;
  agentName: string;
  runtimeId: string;
  sessionId: string;
  sessionTitle: string;
};

async function seedPanelFixture(api: TestApiClient): Promise<PanelFixture> {
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
    const agentName = `E2E Panel Agent ${suffix}`;
    const runtime = await client.query<{ id: string }>(
      `INSERT INTO agent_runtime (
         workspace_id, daemon_id, name, runtime_mode, provider, status,
         device_info, metadata, last_seen_at, owner_id
       ) VALUES ($1, NULL, $2, 'cloud', 'codex', 'online', $2, '{}'::jsonb, now(), $3)
       RETURNING id::text`,
      [workspace.id, `E2E Panel Runtime ${suffix}`, userId],
    );
    const runtimeId = runtime.rows[0]!.id;
    const agent = await client.query<{ id: string }>(
      `INSERT INTO agent (
         workspace_id, name, description, runtime_mode, runtime_config,
         runtime_id, visibility, max_concurrent_tasks, owner_id
       ) VALUES ($1, $2, '', 'cloud', '{}'::jsonb, $3, 'workspace', 1, $4)
       RETURNING id::text`,
      [workspace.id, agentName, runtimeId, userId],
    );
    const agentId = agent.rows[0]!.id;
    const sessionTitle = `E2E Panel Session ${suffix}`;
    const session = await client.query<{ id: string }>(
      `INSERT INTO chat_session (workspace_id, agent_id, creator_id, title, status)
       VALUES ($1, $2, $3, $4, 'active') RETURNING id::text`,
      [workspace.id, agentId, userId, sessionTitle],
    );
    const fixture = {
      agentId,
      agentName,
      runtimeId,
      sessionId: session.rows[0]!.id,
      sessionTitle,
    };
    await client.query("COMMIT");
    return fixture;
  } catch (error) {
    await client.query("ROLLBACK");
    throw error;
  } finally {
    await client.end();
  }
}

async function removePanelFixture(fixture: PanelFixture | null) {
  if (!fixture) return;
  const client = new pg.Client(DATABASE_URL);
  await client.connect();
  try {
    await client.query(`DELETE FROM chat_session WHERE id = $1`, [fixture.sessionId]);
    await client.query(`DELETE FROM agent WHERE id = $1`, [fixture.agentId]);
    await client.query(`DELETE FROM agent_runtime WHERE id = $1`, [fixture.runtimeId]);
  } finally {
    await client.end();
  }
}

test.describe("Chat panel buttons", () => {
  let api: TestApiClient;
  let fixture: PanelFixture | null = null;

  test.beforeEach(async ({ page }) => {
    api = await createTestApi();
    fixture = await seedPanelFixture(api);
    await loginAsDefault(page);
    await page.locator("svg.lucide-message-circle").locator("..").click();
    await page.getByText("New chat", { exact: true }).first().click();
    const history = page.getByRole("group", { name: "Chat history" });
    await history.getByText(fixture.sessionTitle, { exact: true }).click();
    await expect(page.getByText(fixture.agentName, { exact: true })).toBeVisible();
  });

  test.afterEach(async () => {
    await removePanelFixture(fixture);
    fixture = null;
    await api.cleanup();
  });

  test("attach opens a chooser, send posts the draft, and stop posts task cancellation", async ({
    page,
  }) => {
    const taskId = "11111111-1111-4111-8111-111111111111";
    const messageId = "22222222-2222-4222-8222-222222222222";
    let sentBody: Record<string, unknown> | null = null;
    let cancelled = false;
    const corsHeaders = {
      "Access-Control-Allow-Origin": new URL(page.url()).origin,
      "Access-Control-Allow-Credentials": "true",
    };

    await page.route(
      `**/api/chat/sessions/${fixture!.sessionId}/messages`,
      async (route) => {
        if (route.request().method() === "OPTIONS") {
          await route.continue();
          return;
        }
        sentBody = route.request().postDataJSON() as Record<string, unknown>;
        await route.fulfill({
          status: 201,
          contentType: "application/json",
          headers: corsHeaders,
          body: JSON.stringify({
            message_id: messageId,
            task_id: taskId,
            created_at: new Date().toISOString(),
            attachment_ids: [],
          }),
        });
      },
    );
    await page.route(`**/api/tasks/${taskId}/cancel`, async (route) => {
      if (route.request().method() === "OPTIONS") {
        await route.continue();
        return;
      }
      cancelled = true;
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        headers: corsHeaders,
        body: JSON.stringify({
          id: taskId,
          agent_id: fixture!.agentId,
          runtime_id: fixture!.runtimeId,
          issue_id: "",
          status: "cancelled",
          priority: 0,
          dispatched_at: null,
          started_at: null,
          completed_at: new Date().toISOString(),
          result: null,
          error: null,
          created_at: new Date().toISOString(),
        }),
      });
    });

    const attachChooser = page.waitForEvent("filechooser");
    await page.getByRole("button", { name: "Attach file" }).click();
    const chooser = await attachChooser;
    expect(chooser.isMultiple()).toBe(false);

    const draft = `dispatch proof ${Date.now()}`;
    const editor = page.locator('[contenteditable="true"]').last();
    await editor.fill(draft);
    await editor.press("Control+Enter");

    await expect.poll(() => sentBody).not.toBeNull();
    expect(sentBody).toMatchObject({ content: draft });

    const stopButton = page.locator("button:has(svg.lucide-square)").last();
    await expect(stopButton).toBeVisible();
    await stopButton.click();
    await expect.poll(() => cancelled).toBe(true);
  });
});
