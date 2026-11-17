import { afterEach, beforeAll, describe, expect, it } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import axe from "axe-core";
import { App } from "./App";
import { initEngine } from "./engine";

const FIXED_NOW = "2026-11-14T14:30:00-06:00";
const FIXED_TZ = "America/Mexico_City";

describe("ahorita calculator UI", () => {
  beforeAll(async () => {
    await initEngine();
  });

  afterEach(() => {
    cleanup();
    try {
      if (
        typeof window.localStorage !== "undefined" &&
        typeof window.localStorage.clear === "function"
      ) {
        window.localStorage.clear();
      }
    } catch {
      // ignore unavailable storage in Node 26 jsdom
    }
    window.history.replaceState(null, "", "/");
  });

  it("evaluates typed today + 2d to 2026-11-16", async () => {
    const user = userEvent.setup();
    render(<App defaultNow={FIXED_NOW} defaultTz={FIXED_TZ} />);

    const input = screen.getByLabelText("Date or time expression");
    await user.type(input, "today + 2d");

    expect(screen.getByTestId("result-text").textContent).toBe("2026-11-16");
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("builds today + 2d by clicking today, +, 2, d", async () => {
    const user = userEvent.setup();
    render(<App defaultNow={FIXED_NOW} defaultTz={FIXED_TZ} />);

    await user.click(screen.getByRole("button", { name: "today" }));
    await user.click(screen.getByRole("button", { name: "+ (add)" }));
    await user.click(screen.getByRole("button", { name: "2" }));
    await user.click(screen.getByRole("button", { name: "d (days)" }));

    const input = screen.getByLabelText(
      "Date or time expression"
    ) as HTMLInputElement;
    expect(input.value).toBe("today + 2d");
    expect(screen.getByTestId("result-text").textContent).toBe("2026-11-16");
  });

  it("shows no error banner for incomplete now + and disables operator buttons", async () => {
    const user = userEvent.setup();
    render(<App defaultNow={FIXED_NOW} defaultTz={FIXED_TZ} />);

    const input = screen.getByLabelText("Date or time expression");
    await user.type(input, "now +");

    expect(screen.queryByRole("alert")).toBeNull();
    expect(screen.getByText(/Expected:/)).toBeTruthy();

    expect(
      (screen.getByRole("button", { name: "+ (add)" }) as HTMLButtonElement)
        .disabled
    ).toBe(true);
    expect(
      (
        screen.getByRole("button", {
          name: "− (subtract)",
        }) as HTMLButtonElement
      ).disabled
    ).toBe(true);
    expect(
      (
        screen.getByRole("button", {
          name: "× (multiply)",
        }) as HTMLButtonElement
      ).disabled
    ).toBe(true);
  });

  it("inserts ISO date when picking a date from the date picker", () => {
    render(<App defaultNow={FIXED_NOW} defaultTz={FIXED_TZ} />);

    const picker = screen.getByLabelText("Date picker");
    fireEvent.change(picker, { target: { value: "2026-06-02" } });

    const input = screen.getByLabelText(
      "Date or time expression"
    ) as HTMLInputElement;
    expect(input.value).toBe("2026-06-02");
    expect(screen.getByTestId("result-text").textContent).toBe("2026-06-02");
  });

  it("switches output type to relative and updates the result", async () => {
    const user = userEvent.setup();
    render(<App defaultNow={FIXED_NOW} defaultTz={FIXED_TZ} />);

    const input = screen.getByLabelText("Date or time expression");
    await user.type(input, "today + 2d");
    expect(screen.getByTestId("result-text").textContent).toBe("2026-11-16");

    await user.click(screen.getByRole("button", { name: "relative" }));
    expect(screen.getByTestId("result-text").textContent).toBe("in 2 days");
  });

  it("answers in Spanish for hace 2 días and switches keypad labels for lang=es", async () => {
    const user = userEvent.setup();
    render(<App defaultNow={FIXED_NOW} defaultTz={FIXED_TZ} />);

    const langSelect = screen.getByLabelText("Lang");
    await user.selectOptions(langSelect, "es");

    expect(screen.getByRole("button", { name: "hoy" })).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "días hábiles (unidades)" })
    ).toBeTruthy();

    const input = screen.getByLabelText("Expresión de fecha o tiempo");
    await user.type(input, "hace 2 días");
    await user.click(screen.getByRole("button", { name: "relative" }));

    expect(screen.getByTestId("result-text").textContent).toBe("hace 2 días");
  });

  it("has zero critical or serious axe-core accessibility violations", async () => {
    const user = userEvent.setup();
    const { container } = render(
      <App defaultNow={FIXED_NOW} defaultTz={FIXED_TZ} />
    );

    const input = screen.getByLabelText("Date or time expression");
    await user.type(input, "today + 2d");

    const results = await axe.run(container, {
      rules: {
        // jsdom does not compute layout/canvas colors for contrast ratios
        "color-contrast": { enabled: false },
      },
    });
    expect(results.violations).toEqual([]);
  });
});
