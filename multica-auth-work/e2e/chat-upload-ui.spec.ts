import "./env";
import { test, expect } from "@playwright/test";
import pg from "pg";
import { createTestApi, loginAsDefault } from "./helpers";
import type { TestApiClient } from "./fixtures";

const DATABASE_URL =
  process.env.DATABASE_URL ??
  "postgres://multica:multica@localhost:5432/multica?sslmode=disable";

type ChatFixture = {
  agentId: string;
  agentName: string;
  runtimeId: string;
};

async function seedChatAgent(api: TestApiClient): Promise<ChatFixture> {
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

    const agentName = `E2E Upload Agent ${Date.now()}`;
    const runtime = await client.query<{ id: string }>(
      `INSERT INTO agent_runtime (
         workspace_id, daemon_id, name, runtime_mode, provider, status,
         device_info, metadata, last_seen_at, owner_id
       ) VALUES ($1, NULL, $2, 'cloud', 'codex', 'online', $2, '{}'::jsonb, now(), $3)
       RETURNING id::text`,
      [workspace.id, `E2E Upload Runtime ${Date.now()}`, userId],
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

async function removeChatFixture(fixture: ChatFixture | null) {
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

test.describe("Chat upload UI", () => {
  let api: TestApiClient;
  let fixture: ChatFixture | null = null;

  test.beforeEach(async ({ page }) => {
    api = await createTestApi();
    fixture = await seedChatAgent(api);
    await loginAsDefault(page);
  });

  test.afterEach(async () => {
    await removeChatFixture(fixture);
    fixture = null;
    await api.cleanup();
  });

  test("rejects a malformed attachment contract and uses the URL fallback when no attachment row exists", async ({
    page,
  }) => {
    let uploadCount = 0;
    const corsHeaders = {
      "Access-Control-Allow-Origin": new URL(page.url()).origin,
      "Access-Control-Allow-Credentials": "true",
    };
    await page.route("**/api/upload-file", async (route) => {
      if (route.request().method() === "OPTIONS") {
        await route.continue();
        return;
      }
      uploadCount += 1;
      if (uploadCount === 1) {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          headers: corsHeaders,
          body: JSON.stringify({
            id: "degraded-attachment",
            url: "https://uploads.invalid/degraded.txt",
            filename: "degraded.txt",
          }),
        });
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        headers: corsHeaders,
        body: JSON.stringify({
          id: "",
          url: "https://uploads.invalid/no-workspace.txt",
          download_url: "https://uploads.invalid/no-workspace.txt",
          filename: "no-workspace.txt",
        }),
      });
    });

    await page.locator("svg.lucide-message-circle").locator("..").click();
    await expect(page.getByText(fixture!.agentName, { exact: true })).toBeVisible();
    const editor = page.locator('[contenteditable="true"]').last();
    await expect(editor).toBeVisible();

    const firstChooser = page.waitForEvent("filechooser");
    await page.getByRole("button", { name: "Attach file" }).click();
    await (await firstChooser).setFiles({
      name: "degraded.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("malformed contract"),
    });

    await expect.poll(() => uploadCount).toBe(1);
    await expect(
      page.locator('.file-card-node[data-type="fileCard"]').filter({
        hasText: "degraded.txt",
      }),
    ).toHaveCount(0);
    await expect(editor).toBeEditable();

    const secondChooser = page.waitForEvent("filechooser");
    await page.getByRole("button", { name: "Attach file" }).click();
    await (await secondChooser).setFiles({
      name: "no-workspace.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("URL fallback"),
    });

    await expect.poll(() => uploadCount).toBe(2);
    const uploadedCard = page
      .locator('.file-card-node[data-type="fileCard"]')
      .filter({ hasText: "no-workspace.txt" });
    await expect(uploadedCard).toBeVisible();
    await page.evaluate(() => {
      const state = window as typeof window & { __e2eOpenedUrl?: string };
      state.__e2eOpenedUrl = "";
      window.open = ((url?: string | URL) => {
        state.__e2eOpenedUrl = String(url ?? "");
        return null;
      }) as typeof window.open;
    });
    await uploadedCard.getByRole("button", { name: "Download" }).click();
    await expect
      .poll(() =>
        page.evaluate(
          () =>
            (window as typeof window & { __e2eOpenedUrl?: string })
              .__e2eOpenedUrl,
        ),
      )
      .toBe("https://uploads.invalid/no-workspace.txt");
    await expect(editor).toBeEditable();
  });
});
