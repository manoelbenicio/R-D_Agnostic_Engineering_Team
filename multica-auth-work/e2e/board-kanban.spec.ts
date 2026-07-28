import "./env";
import { test, expect } from "@playwright/test";
import pg from "pg";
import { createTestApi, loginAsDefault, waitForPageText } from "./helpers";
import type { TestApiClient } from "./fixtures";

const DATABASE_URL =
  process.env.DATABASE_URL ??
  "postgres://multica:multica@localhost:5432/multica?sslmode=disable";

async function persistedStatus(issueId: string): Promise<string | null> {
  const client = new pg.Client(DATABASE_URL);
  await client.connect();
  try {
    const result = await client.query<{ status: string }>(
      `SELECT status FROM issue WHERE id = $1`,
      [issueId],
    );
    return result.rows[0]?.status ?? null;
  } finally {
    await client.end();
  }
}

test.use({ locale: "en-US" });

test.describe("Board Kanban", () => {
  let api: TestApiClient;

  test.beforeEach(async ({ page }) => {
    api = await createTestApi();
    await loginAsDefault(page);
  });

  test.afterEach(async () => {
    await api.cleanup();
  });

  test("moves a card from Backlog to Todo and preserves the status after reload", async ({
    page,
  }) => {
    const title = `E2E Kanban Move ${Date.now()}`;
    const issue = await api.createIssue(title, { status: "backlog" });
    await page.reload({ waitUntil: "domcontentloaded" });
    await waitForPageText(page, title);

    const backlogHeading = page.getByText("Backlog", { exact: true }).first();
    const todoHeading = page.getByText("Todo", { exact: true }).first();
    const columnXPath =
      "xpath=ancestor::div[contains(concat(' ', normalize-space(@class), ' '), ' shrink-0 ') and contains(concat(' ', normalize-space(@class), ' '), ' flex-col ')][1]";
    const backlogColumn = backlogHeading.locator(columnXPath);
    const todoColumn = todoHeading.locator(columnXPath);
    const titleNode = backlogColumn.getByText(title, { exact: true });
    await expect(titleNode).toBeVisible();

    const draggable = titleNode.locator("xpath=ancestor::a[1]/parent::div");
    const dropSurface = todoColumn.locator("div.absolute.inset-0").last();
    const sourceBox = await draggable.boundingBox();
    const targetBox = await dropSurface.boundingBox();
    if (!sourceBox || !targetBox) throw new Error("Kanban drag coordinates unavailable");

    const moveRequest = page.waitForRequest(
      (request) => {
        if (
          request.method() !== "PUT" ||
          !request.url().endsWith(`/api/issues/${issue.id}`)
        ) {
          return false;
        }
        const body = request.postDataJSON() as Record<string, unknown>;
        return body.status === "todo";
      },
      { timeout: 15000 },
    );

    await page.mouse.move(
      sourceBox.x + sourceBox.width / 2,
      sourceBox.y + sourceBox.height / 2,
    );
    await page.mouse.down();
    await page.mouse.move(sourceBox.x + sourceBox.width / 2 + 12, sourceBox.y + 12, {
      steps: 4,
    });
    await page.mouse.move(
      targetBox.x + targetBox.width / 2,
      targetBox.y + Math.min(120, targetBox.height / 2),
      { steps: 20 },
    );
    await page.mouse.up();

    const request = await moveRequest;
    const response = await request.response();
    expect(response?.ok()).toBe(true);
    await expect(todoColumn.getByText(title, { exact: true })).toBeVisible();
    await expect.poll(() => persistedStatus(issue.id)).toBe("todo");

    await page.reload({ waitUntil: "domcontentloaded" });
    await waitForPageText(page, title);
    const reloadedTodoColumn = page
      .getByText("Todo", { exact: true })
      .first()
      .locator(columnXPath);
    await expect(reloadedTodoColumn.getByText(title, { exact: true })).toBeVisible();
  });
});
