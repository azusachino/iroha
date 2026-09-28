import { afterEach, describe, expect, it, vi } from "vitest";
import {
  ApiError,
  UNAUTHENTICATED_EVENT,
  getMetricCatalog,
  issueIntakeCredential,
  setCsrfToken,
} from "./api";

function recordingFetch(status = 200, body: unknown = {}) {
  const calls: { url: string; init: RequestInit }[] = [];
  const fn = (async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    return new Response(JSON.stringify(body), {
      status,
      headers: { "content-type": "application/json" },
    });
  }) as unknown as typeof fetch;
  return { fn, calls };
}

afterEach(() => {
  setCsrfToken("");
  vi.unstubAllGlobals();
});

describe("owner session plumbing", () => {
  it("sends the CSRF token on state-changing requests only", async () => {
    setCsrfToken("csrf-1");
    const post = recordingFetch(201, { credential: {}, token: "t" });
    await issueIntakeCredential("primary", post.fn);
    const postHeaders = post.calls[0].init.headers as Record<string, string>;
    expect(postHeaders["x-csrf-token"]).toBe("csrf-1");
    expect(post.calls[0].init.credentials).toBe("same-origin");

    const get = recordingFetch(200, { metrics: [] });
    await getMetricCatalog(get.fn);
    const getHeaders = get.calls[0].init.headers as Record<string, string>;
    expect(getHeaders["x-csrf-token"]).toBeUndefined();
  });

  it("announces an expired session so the app can show login", async () => {
    const dispatch = vi.fn();
    vi.stubGlobal("window", { dispatchEvent: dispatch });
    const unauthorized = recordingFetch(401, {
      code: "unauthenticated",
      message: "login required",
    });
    await expect(getMetricCatalog(unauthorized.fn)).rejects.toBeInstanceOf(
      ApiError,
    );
    expect(dispatch).toHaveBeenCalledOnce();
    expect((dispatch.mock.calls[0][0] as Event).type).toBe(
      UNAUTHENTICATED_EVENT,
    );
  });
});
