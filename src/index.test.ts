import { describe, expect, it } from "vitest";
import worker from "./index";

describe("Cloudflare Worker routes", () => {
  it("answers health checks through D1", async () => {
    const env = { DB: { prepare: () => ({ first: async () => ({ ok: 1 }) }) } } as unknown as Parameters<typeof worker.fetch>[1];
    const response = await worker.fetch(new Request("https://minyo-go.example/healthz"), env);
    expect(response.status).toBe(200);
    expect(await response.json()).toEqual({ status: "ok" });
  });

  it("returns JSON 404 for unknown routes", async () => {
    const env = { DB: {} } as unknown as Parameters<typeof worker.fetch>[1];
    const response = await worker.fetch(new Request("https://minyo-go.example/not-found"), env);
    expect(response.status).toBe(404);
    expect(await response.json()).toEqual({ error: "not found" });
  });
});
