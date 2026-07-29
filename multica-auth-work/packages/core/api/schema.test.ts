import { afterEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";
import { ApiClient } from "./client";
import { parseWithFallback } from "./schema";

// Helper: stub fetch with a single JSON response. Status defaults to 200.
function stubFetchJson(body: unknown, status = 200) {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(
      new Response(typeof body === "string" ? body : JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
      }),
    ),
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

// Malformed server responses must fail closed instead of looking like valid
// empty/success records. Schema-level defaults remain covered separately.
describe("ApiClient schema contract enforcement", () => {
  describe("listTimeline", () => {
    it("fails closed when the body is null", async () => {
      stubFetchJson(null);
      const client = new ApiClient("https://api.example.test");
      await expect(client.listTimeline("issue-1")).rejects.toThrow(
        "API response failed schema validation",
      );
    });

    it("fails closed when the body is not an array", async () => {
      stubFetchJson({ wrong: "shape" });
      const client = new ApiClient("https://api.example.test");
      await expect(client.listTimeline("issue-1")).rejects.toThrow(
        "API response failed schema validation",
      );
    });

    it("accepts a new entry type rather than crashing on enum drift", async () => {
      stubFetchJson([
        {
          type: "future_kind", // not in TS union
          id: "e-1",
          actor_type: "member",
          actor_id: "u-1",
          created_at: "2026-01-01T00:00:00Z",
        },
      ]);
      const client = new ApiClient("https://api.example.test");
      const entries = await client.listTimeline("issue-1");
      expect(entries).toHaveLength(1);
      expect(entries[0]?.type).toBe("future_kind");
    });

    // Forward-compat: when the server adds a new field to an existing
    // shape, `.loose()` lets it pass through unchanged. Without `.loose()`
    // zod 4 strips it, which would silently break a future TS type that
    // adopts the field — see schemas.ts header comment.
    it("preserves unknown fields the schema didn't list", async () => {
      stubFetchJson([
        {
          type: "comment",
          id: "e-1",
          actor_type: "member",
          actor_id: "u-1",
          created_at: "2026-01-01T00:00:00Z",
          // New server-side field not present in TimelineEntrySchema:
          future_field: { nested: "value" },
        },
      ]);
      const client = new ApiClient("https://api.example.test");
      const entries = await client.listTimeline("issue-1");
      const entry = entries[0] as unknown as Record<string, unknown>;
      expect(entry.future_field).toEqual({ nested: "value" });
    });
  });

  describe("listIssues", () => {
    it("fails closed when the response is malformed", async () => {
      stubFetchJson({ issues: "not-an-array", total: 0 });
      const client = new ApiClient("https://api.example.test");
      await expect(client.listIssues()).rejects.toThrow(
        "API response failed schema validation",
      );
    });
  });

  describe("listAutopilots", () => {
    const baseAutopilot = {
      id: "ap-1",
      workspace_id: "ws-1",
      title: "Daily triage",
      description: null,
      assignee_id: "agent-1",
      status: "active",
      execution_mode: "run_only",
      issue_title_template: null,
      created_by_type: "member",
      created_by_id: "user-1",
      last_run_at: null,
      created_at: "2026-06-01T00:00:00Z",
      updated_at: "2026-06-01T00:00:00Z",
    };

    it("fails closed when the response is malformed", async () => {
      stubFetchJson({ autopilots: "not-an-array", total: 1 });
      const client = new ApiClient("https://api.example.test");
      await expect(client.listAutopilots()).rejects.toThrow(
        "API response failed schema validation",
      );
    });

    it("accepts an old-server row without assignee_type or derived fields", async () => {
      // Pre-MUL-2429 servers omit assignee_type; servers older than the
      // list-derived-fields change omit trigger_kinds/next_run_at/
      // last_run_status. Both must parse, not fall back.
      stubFetchJson({ autopilots: [baseAutopilot], total: 1 });
      const client = new ApiClient("https://api.example.test");
      const res = await client.listAutopilots();
      expect(res.autopilots).toHaveLength(1);
      expect(res.autopilots[0]?.assignee_type).toBe("agent");
      expect(res.autopilots[0]?.trigger_kinds).toBeUndefined();
      expect(res.autopilots[0]?.last_run_status).toBeUndefined();
    });

    it("passes derived fields through and tolerates enum drift", async () => {
      stubFetchJson({
        autopilots: [
          {
            ...baseAutopilot,
            assignee_type: "squad",
            trigger_kinds: ["schedule", "some_future_kind"],
            next_run_at: "2026-06-13T09:00:00Z",
            last_run_status: "some_future_status",
          },
        ],
        total: 1,
      });
      const client = new ApiClient("https://api.example.test");
      const res = await client.listAutopilots();
      expect(res.autopilots[0]?.trigger_kinds).toEqual([
        "schedule",
        "some_future_kind",
      ]);
      expect(res.autopilots[0]?.next_run_at).toBe("2026-06-13T09:00:00Z");
      expect(res.autopilots[0]?.last_run_status).toBe("some_future_status");
    });
  });

  describe("getConfig", () => {
    it("drops malformed daemon setup URLs instead of throwing", async () => {
      stubFetchJson({
        cdn_domain: "cdn.example.com",
        allow_signup: true,
        daemon_server_url: { wrong: "shape" },
        daemon_app_url: 123,
        workspace_creation_disabled: false,
      });
      const client = new ApiClient("https://api.example.test");
      const config = await client.getConfig();
      expect(config.cdn_domain).toBe("cdn.example.com");
      expect(config.allow_signup).toBe(true);
      expect(config.daemon_server_url).toBeUndefined();
      expect(config.daemon_app_url).toBeUndefined();
    });
  });

  describe("listGroupedIssues", () => {
    it("fails closed when grouped issues are malformed", async () => {
      stubFetchJson({ groups: "not-an-array" });
      const client = new ApiClient("https://api.example.test");
      await expect(
        client.listGroupedIssues({ group_by: "assignee" }),
      ).rejects.toThrow("API response failed schema validation");
    });
  });

  describe("listComments", () => {
    it("fails closed when the response is not an array", async () => {
      stubFetchJson({ wrong: "shape" });
      const client = new ApiClient("https://api.example.test");
      await expect(client.listComments("issue-1")).rejects.toThrow(
        "API response failed schema validation",
      );
    });
  });

  describe("previewCommentTriggers", () => {
    it("fails closed when the response is malformed", async () => {
      stubFetchJson({ agents: "not-an-array" });
      const client = new ApiClient("https://api.example.test");
      await expect(
        client.previewCommentTriggers("issue-1", "hello"),
      ).rejects.toThrow("API response failed schema validation");
    });
  });

  describe("listIssueSubscribers", () => {
    it("fails closed when the response is null", async () => {
      stubFetchJson(null);
      const client = new ApiClient("https://api.example.test");
      await expect(client.listIssueSubscribers("issue-1")).rejects.toThrow(
        "API response failed schema validation",
      );
    });
  });

  describe("listChildIssues", () => {
    it("returns { issues: [] } when the issues field is missing", async () => {
      stubFetchJson({});
      const client = new ApiClient("https://api.example.test");
      const res = await client.listChildIssues("issue-1");
      expect(res).toEqual({ issues: [] });
    });
  });

  // Agent template catalog is hit by the desktop create-agent picker.
  // Installed desktop builds outlive any given server, so the shape MUST
  // survive future field renames / wrapping without crashing. Each test
  // here mirrors a concrete future drift we want to absorb.
  describe("listAgentTemplates", () => {
    it("fails closed when the body is null", async () => {
      stubFetchJson(null);
      const client = new ApiClient("https://api.example.test");
      await expect(client.listAgentTemplates()).rejects.toThrow(
        "API response failed schema validation",
      );
    });

    it("defaults skills to [] when the field is missing from a template", async () => {
      // Future server: drops `skills` because the picker no longer reads
      // them. Picker code calls `template.skills.length` — must not throw.
      stubFetchJson([{ slug: "x", name: "X" }]);
      const client = new ApiClient("https://api.example.test");
      const tmpls = await client.listAgentTemplates();
      expect(tmpls).toHaveLength(1);
      expect(tmpls[0]?.skills).toEqual([]);
    });

    it("accepts the bare-array shape (current contract)", async () => {
      stubFetchJson([
        { slug: "a", name: "A", description: "", skills: [] },
        { slug: "b", name: "B", description: "", skills: [] },
      ]);
      const client = new ApiClient("https://api.example.test");
      const tmpls = await client.listAgentTemplates();
      expect(tmpls.map((t) => t.slug)).toEqual(["a", "b"]);
    });

    it("accepts a future {templates: [...]} envelope without breaking", async () => {
      // Server migrates to a paginated envelope. We unwrap so the picker
      // keeps working on the older bare-array consumer.
      stubFetchJson({
        templates: [{ slug: "a", name: "A", description: "", skills: [] }],
        total: 1,
      });
      const client = new ApiClient("https://api.example.test");
      const tmpls = await client.listAgentTemplates();
      expect(tmpls).toHaveLength(1);
      expect(tmpls[0]?.slug).toBe("a");
    });
  });

  describe("getAgentTemplate", () => {
    it("fails closed instead of fabricating a template detail", async () => {
      stubFetchJson({ wrong: "shape" });
      const client = new ApiClient("https://api.example.test");
      await expect(client.getAgentTemplate("code-reviewer")).rejects.toThrow(
        "API response failed schema validation",
      );
    });

    it("defaults instructions to '' when the field is missing", async () => {
      stubFetchJson({
        slug: "code-reviewer",
        name: "Code Reviewer",
        description: "",
        skills: [],
      });
      const client = new ApiClient("https://api.example.test");
      const detail = await client.getAgentTemplate("code-reviewer");
      expect(detail.instructions).toBe("");
    });
  });

  describe("listAutopilotDeliveries", () => {
    it("fails closed when the body is null", async () => {
      stubFetchJson(null);
      const client = new ApiClient("https://api.example.test");
      await expect(client.listAutopilotDeliveries("ap-1")).rejects.toThrow(
        "API response failed schema validation",
      );
    });

    it("fails closed when `deliveries` is not an array", async () => {
      stubFetchJson({ deliveries: "not-an-array", total: 0 });
      const client = new ApiClient("https://api.example.test");
      await expect(client.listAutopilotDeliveries("ap-1")).rejects.toThrow(
        "API response failed schema validation",
      );
    });

    it("accepts an unknown future status value rather than dropping the row", async () => {
      // Server-side enum drift (e.g. new `quarantined` state). The list
      // must still surface the row; downstream UI code's `default` arm
      // handles unknown values with a generic visual.
      stubFetchJson({
        deliveries: [
          {
            id: "d-1",
            workspace_id: "ws-1",
            autopilot_id: "ap-1",
            trigger_id: "t-1",
            provider: "github",
            event: "pull_request.opened",
            dedupe_key: "abc",
            dedupe_source: "x-github-delivery",
            signature_status: "valid",
            status: "quarantined",
            attempt_count: 1,
            content_type: "application/json",
            response_status: 200,
            autopilot_run_id: null,
            replayed_from_delivery_id: null,
            error: null,
            received_at: "2026-01-01T00:00:00Z",
            last_attempt_at: "2026-01-01T00:00:00Z",
            created_at: "2026-01-01T00:00:00Z",
          },
        ],
        total: 1,
      });
      const client = new ApiClient("https://api.example.test");
      const res = await client.listAutopilotDeliveries("ap-1");
      expect(res.deliveries).toHaveLength(1);
      expect(res.deliveries[0]?.status).toBe("quarantined");
    });
  });

  describe("getAutopilotDelivery", () => {
    it("fails closed instead of fabricating a delivery", async () => {
      stubFetchJson({ wrong: "shape" });
      const client = new ApiClient("https://api.example.test");
      await expect(client.getAutopilotDelivery("ap-1", "d-1")).rejects.toThrow(
        "API response failed schema validation",
      );
    });
  });

  describe("createAgentFromTemplate", () => {
    it("fails closed when the response is malformed", async () => {
      stubFetchJson({ unexpected: "shape" });
      const client = new ApiClient("https://api.example.test");
      await expect(
        client.createAgentFromTemplate({
          template_slug: "x",
          name: "X",
          runtime_id: "rt-1",
        }),
      ).rejects.toThrow("API response failed schema validation");
    });

    it("defaults imported_skill_ids / reused_skill_ids to [] when missing", async () => {
      stubFetchJson({ agent: { id: "agent-1" } });
      const client = new ApiClient("https://api.example.test");
      const resp = await client.createAgentFromTemplate({
        template_slug: "x",
        name: "X",
        runtime_id: "rt-1",
      });
      expect(resp.agent.id).toBe("agent-1");
      expect(resp.imported_skill_ids).toEqual([]);
      expect(resp.reused_skill_ids).toEqual([]);
    });
  });
});

// Direct tests for the helper, decoupled from any specific endpoint —
// guards against an endpoint refactor masking a regression in the helper.
describe("parseWithFallback", () => {
  const opts = { endpoint: "TEST /unit" };

  it("returns parsed data on success", () => {
    const schema = z.object({ id: z.string() });
    const out = parseWithFallback({ id: "x" }, schema, { id: "fallback" }, opts);
    expect(out).toEqual({ id: "x" });
  });

  it("throws a contract error when validation fails", () => {
    const schema = z.object({ id: z.string() });
    expect(() =>
      parseWithFallback({ id: 123 }, schema, { id: "fallback" }, opts),
    ).toThrow("API response failed schema validation: TEST /unit");
  });

  it("throws a contract error when data is null", () => {
    const schema = z.object({ id: z.string() });
    expect(() =>
      parseWithFallback(null, schema, { id: "fallback" }, opts),
    ).toThrow("API response failed schema validation: TEST /unit");
  });
});
