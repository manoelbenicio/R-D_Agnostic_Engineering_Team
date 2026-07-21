import assert from "node:assert/strict";
import { after, beforeEach, test } from "node:test";

import {
  bindResponseConnectionAffinity,
  getResponseConnectionAffinity,
  touchResponseConnectionAffinity,
} from "@/lib/db/sessionAccountAffinity";
import { closeDbInstance, getDbInstance } from "@/lib/db/core";

function resetResponseRows(): void {
  const db = getDbInstance();
  db.exec(`CREATE TABLE IF NOT EXISTS session_account_affinity (
    session_key TEXT NOT NULL,
    provider TEXT NOT NULL,
    connection_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL,
    PRIMARY KEY (session_key, provider)
  )`);
  db.prepare("DELETE FROM session_account_affinity WHERE session_key LIKE 'responses:%'").run();
}

beforeEach(resetResponseRows);
after(() => closeDbInstance());

test("origin bind and previous_response_id lookup persist no raw handle", () => {
  bindResponseConnectionAffinity("resp-sensitive-1", "provider-a", "connection-a", 1_000, 10_000, 10);
  assert.deepEqual(getResponseConnectionAffinity("resp-sensitive-1", 1_001, 10_000), {
    provider: "provider-a",
    connectionId: "connection-a",
  });
  const rows = getDbInstance()
    .prepare("SELECT session_key FROM session_account_affinity WHERE session_key LIKE 'responses:%'")
    .all() as Array<{ session_key: string }>;
  assert.equal(rows.length, 1);
  assert.equal(rows[0].session_key.includes("resp-sensitive-1"), false);
});

test("missing and expired handles fail closed", () => {
  assert.equal(getResponseConnectionAffinity("missing", 2_000, 100), null);
  bindResponseConnectionAffinity("expired", "provider-a", "connection-a", 1_000, 100, 10);
  assert.equal(getResponseConnectionAffinity("expired", 1_101, 100), null);
});

test("cap eviction is deterministic and keeps newest owners", () => {
  bindResponseConnectionAffinity("r1", "provider-a", "a", 1_000, 10_000, 2);
  bindResponseConnectionAffinity("r2", "provider-a", "b", 2_000, 10_000, 2);
  bindResponseConnectionAffinity("r3", "provider-a", "c", 3_000, 10_000, 2);
  assert.equal(getResponseConnectionAffinity("r1", 3_001, 10_000), null);
  assert.equal(getResponseConnectionAffinity("r2", 3_001, 10_000)?.connectionId, "b");
  assert.equal(getResponseConnectionAffinity("r3", 3_001, 10_000)?.connectionId, "c");
});

test("transactional concurrent-origin inserts remain bounded", async () => {
  await Promise.all(
    Array.from({ length: 32 }, (_, index) =>
      Promise.resolve().then(() =>
        bindResponseConnectionAffinity(
          `concurrent-${index}`,
          "provider-a",
          `connection-${index % 2}`,
          10_000 + index,
          10_000,
          8
        )
      )
    )
  );
  const row = getDbInstance()
    .prepare(
      "SELECT COUNT(*) AS count FROM session_account_affinity WHERE session_key LIKE 'responses:%'"
    )
    .get() as { count: number };
  assert.equal(row.count, 8);
});

test("binding survives database close and reopen", () => {
  bindResponseConnectionAffinity("restart", "provider-a", "connection-a", 1_000, 10_000, 10);
  closeDbInstance();
  assert.deepEqual(getResponseConnectionAffinity("restart", 1_001, 10_000), {
    provider: "provider-a",
    connectionId: "connection-a",
  });
});

test("touch requires the exact owner and extends only after success", () => {
  bindResponseConnectionAffinity("touch", "provider-a", "connection-a", 1_000, 100, 10);
  assert.equal(touchResponseConnectionAffinity("touch", "provider-a", "wrong", 1_050), false);
  assert.equal(touchResponseConnectionAffinity("touch", "provider-a", "connection-a", 1_050), true);
  assert.equal(getResponseConnectionAffinity("touch", 1_149, 100)?.connectionId, "connection-a");
});
