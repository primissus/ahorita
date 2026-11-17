import { describe, expect, it } from "vitest";
import { initEngine, run } from "./engine";

describe("ahorita WASM engine bridge", () => {
  it("evaluates 2026-06-02 + 2d to 2026-06-04", async () => {
    await initEngine();
    const res = run("2026-06-02 + 2d", {
      tz: "America/Mexico_City",
      now: "2026-11-14T14:30:00-06:00",
    });
    expect(res.error).toBeUndefined();
    expect(res.kind).toBe("date");
    expect(res.text).toBe("2026-06-04");
    expect(res.iso).toBe("2026-06-04");
  });
});
