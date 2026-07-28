import "./env";
import { test, expect } from "@playwright/test";
import pg from "pg";
import { createTestApi, loginAsDefault, reloadAppPage } from "./helpers";
import type { TestApiClient } from "./fixtures";

const DATABASE_URL =
  process.env.DATABASE_URL ??
  "postgres://multica:multica@localhost:5432/multica?sslmode=disable";

type DeleteFixture = {
  runtimeId: string;
  runtimeName: string;
};

async function seedRuntime(api: TestApiClient): Promise<DeleteFixture> {
  const workspace = (await api.getWorkspaces())[0];
  if (!workspace) throw new Error("E2E workspace missing");
  api.setWorkspaceId(workspace.id);
  api.setWorkspaceSlug(workspace.slug);
  const client = new pg.Client(DATABASE_URL);
  await client.connect();
  try {
    const user = await client.query<{ id: string }>(
      `SELECT id::text FROM "user" WHERE email = $1`,
      [api.getEmail()],
    );
    const userId = user.rows[0]?.id;
    if (!userId) throw new Error("E2E user missing");
    const runtimeName = `E2E Delete Runtime ${Date.now()}`;
    const runtime = await client.query<{ id: string }>(
      `INSERT INTO agent_runtime (
         workspace_id, daemon_id, name, runtime_mode, provider, status,
         device_info, metadata, last_seen_at, owner_id
       ) VALUES ($1, NULL, $2, 'cloud', 'codex', 'online', $2, '{}'::jsonb, now(), $3)
       RETURNING id::text`,
      [workspace.id, runtimeName, userId],
    );
    return { runtimeId: runtime.rows[0]!.id, runtimeName };
  } finally {
    await client.end();
  }
}

async function removeRuntime(fixture: DeleteFixture | null) {
  if (!fixture) return;
  const client = new pg.Client(DATABASE_URL);
  await client.connect();
  try {
    await client.query(`DELETE FROM agent_runtime WHERE id = $1`, [fixture.runtimeId]);
  } finally {
    await client.end();
  }
}

async function rowExists(table: "issue" | "agent_runtime", id: string) {
  const client = new pg.Client(DATABASE_URL);
  await client.connect();
  try {
    const result = await client.query(`SELECT 1 FROM ${table} WHERE id = $1`, [id]);
    return result.rowCount === 1;
  } finally {
    await client.end();
  }
}

test.describe("Delete flows", () => {
  let api: TestApiClient;
  let fixture: DeleteFixture | null = null;

  test.beforeEach(async ({ page }) => {
    api = await createTestApi();
    fixture = await seedRuntime(api);
    await loginAsDefault(page);
  });

  test.afterEach(async () => {
    await removeRuntime(fixture);
    fixture = null;
    await api.cleanup();
  });

  test("deletes an issue and a runtime after explicit confirmation", async ({ page }) => {
    const issueTitle = `E2E Delete Issue ${Date.now()}`;
    const issue = await api.createIssue(issueTitle);
    await reloadAppPage(page);

    const issueCard = page.getByText(issueTitle, { exact: true }).first();
    await expect(issueCard).toBeVisible();
    await issueCard.click({ button: "right" });
    await page.getByRole("menuitem", { name: "Delete issue" }).click();

    const issueDialog = page.getByRole("alertdialog");
    await expect(issueDialog.getByText("Delete issue", { exact: true })).toBeVisible();
    await issueDialog.getByRole("button", { name: "Delete", exact: true }).click();
    await expect(page.getByText(issueTitle, { exact: true })).toHaveCount(0);
    await expect.poll(() => rowExists("issue", issue.id)).toBe(false);

    const workspace = (await api.getWorkspaces())[0]!;
    await page.goto(`/${workspace.slug}/runtimes`, { waitUntil: "domcontentloaded" });
    const runtimeRow = page
      .getByRole("row")
      .filter({ hasText: fixture!.runtimeName });
    await expect(runtimeRow).toBeVisible();
    await runtimeRow.getByRole("button", { name: "Row actions" }).click();
    await page.getByRole("menuitem", { name: "Delete" }).click();

    const runtimeDialog = page.getByRole("alertdialog");
    await expect(runtimeDialog.getByText("Delete Runtime?", { exact: true })).toBeVisible();
    await runtimeDialog
      .getByRole("button", { name: "Delete runtime", exact: true })
      .click();

    await expect(runtimeRow).toHaveCount(0);
    await expect.poll(() => rowExists("agent_runtime", fixture!.runtimeId)).toBe(false);
  });
});
