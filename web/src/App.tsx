import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type KeyboardEvent,
} from "react";
import {
  initEngine,
  readCurrentTimeAndZone,
  run,
  type Language,
  type MonthOverflowRule,
  type OutputType,
  type Result,
  type RunOptions,
  type WeekOneRule,
} from "./engine";
import {
  applyBackspace,
  computeEnabledButtonIds,
  getCategoryTabs,
  getCoreSections,
  insertIntoExpression,
  isIncompleteParseError,
  replaceTokenOrInsert,
  utf8ByteOffsetToUtf16Index,
  type CategoryId,
  type KeypadButton,
} from "./keypad";

export interface AppProps {
  defaultNow?: string;
  defaultTz?: string;
}

type ThemeOverride = "system" | "light" | "dark";

interface StoredSettings {
  tz: string;
  lang: Language;
  type: OutputType;
  format: string;
  customFormat: string;
  explain: boolean;
  strict: boolean;
  week_start: string;
  week_one: WeekOneRule;
  month_overflow: MonthOverflowRule;
  count_inclusive: boolean;
  country: string;
  weekend: string[];
  days_off: string[];
}

const SETTINGS_STORAGE_KEY = "ahorita.settings.v1";
const THEME_STORAGE_KEY = "color-scheme";

const FALLBACK_ZONES = [
  "UTC",
  "America/Mexico_City",
  "America/New_York",
  "America/Chicago",
  "America/Denver",
  "America/Los_Angeles",
  "America/Bogota",
  "America/Sao_Paulo",
  "America/Argentina/Buenos_Aires",
  "Europe/London",
  "Europe/Madrid",
  "Europe/Paris",
  "Europe/Berlin",
  "Asia/Tokyo",
  "Asia/Shanghai",
  "Australia/Sydney",
];

const WEEKDAYS_LIST = [
  "monday",
  "tuesday",
  "wednesday",
  "thursday",
  "friday",
  "saturday",
  "sunday",
];

const FORMAT_PRESETS = ["iso", "rfc3339", "unix", "human", "short"] as const;

const PATTERN_CHIPS = [
  "yyyy",
  "MM",
  "dd",
  "EEE",
  "HH",
  "mm",
  "ss",
  "a",
  "XXX",
];

function getAvailableTimeZones(currentTZ: string): string[] {
  const intlWithZones = Intl as unknown as {
    supportedValuesOf?: (key: string) => string[];
  };
  let list: string[] = [];
  if (typeof intlWithZones.supportedValuesOf === "function") {
    try {
      list = intlWithZones.supportedValuesOf("timeZone");
    } catch {
      list = [];
    }
  }
  const combined = new Set<string>([
    ...FALLBACK_ZONES,
    ...(currentTZ ? [currentTZ] : []),
    ...list,
  ]);
  return Array.from(combined);
}

function getSystemDark(): boolean {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return false;
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches;
}

function getSafeStorage(): Storage | null {
  try {
    if (
      typeof window !== "undefined" &&
      window.localStorage &&
      typeof window.localStorage.getItem === "function"
    ) {
      return window.localStorage;
    }
  } catch {
    // ignore storage errors
  }
  return null;
}

function readStoredThemeOverride(): ThemeOverride {
  const val = getSafeStorage()?.getItem(THEME_STORAGE_KEY);
  if (val === "light" || val === "dark") {
    return val;
  }
  return "system";
}

function applyThemeOverride(mode: ThemeOverride): void {
  const store = getSafeStorage();
  try {
    if (mode === "system") {
      store?.removeItem(THEME_STORAGE_KEY);
    } else {
      store?.setItem(THEME_STORAGE_KEY, mode);
    }
  } catch {
    // ignore storage errors
  }
  const meta = document.querySelector('meta[name="color-scheme"]');
  if (meta) {
    meta.setAttribute("content", mode === "system" ? "light dark" : mode);
  }
}

function loadInitialState(defaultTz?: string): {
  expr: string;
  settings: StoredSettings;
  tzFallbackNotice: boolean;
} {
  let detectedTZ = defaultTz ?? "";
  let tzFallbackNotice = false;
  if (!detectedTZ) {
    const info = readCurrentTimeAndZone();
    detectedTZ = info.tz;
    tzFallbackNotice = info.tzFallbackUsed;
  }

  let settings: StoredSettings = {
    tz: detectedTZ,
    lang: "en",
    type: "",
    format: "",
    customFormat: "dd-MM-yyyy HH:mm",
    explain: false,
    strict: false,
    week_start: "monday",
    week_one: "iso",
    month_overflow: "clamp",
    count_inclusive: false,
    country: "MX",
    weekend: ["saturday", "sunday"],
    days_off: [],
  };

  try {
    const raw = getSafeStorage()?.getItem(SETTINGS_STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<StoredSettings>;
      settings = {
        ...settings,
        ...parsed,
        tz: defaultTz || parsed.tz || detectedTZ,
        weekend: Array.isArray(parsed.weekend) ? parsed.weekend : settings.weekend,
        days_off: Array.isArray(parsed.days_off) ? parsed.days_off : settings.days_off,
      };
    }
  } catch {
    // ignore invalid stored JSON
  }

  let expr = "";
  if (typeof window !== "undefined") {
    const params = new URLSearchParams(window.location.search);
    const q = params.get("q");
    if (q !== null) expr = q;

    const t = params.get("t");
    if (
      t === "date" ||
      t === "datetime" ||
      t === "time" ||
      t === "range" ||
      t === "relative"
    ) {
      settings.type = t;
    }

    const f = params.get("f");
    if (f) {
      if ((FORMAT_PRESETS as readonly string[]).includes(f)) {
        settings.format = f;
      } else {
        settings.format = "custom";
        settings.customFormat = f;
      }
    }

    const tz = params.get("tz");
    if (tz) settings.tz = tz;

    const lang = params.get("lang");
    if (lang === "en" || lang === "es") {
      settings.lang = lang;
    }
  }

  return { expr, settings, tzFallbackNotice };
}

function formatZoneForInsert(zone: string): string {
  const trimmed = zone.trim();
  if (!trimmed) return "";
  if (/\s/.test(trimmed) && !/^".*"$/.test(trimmed)) {
    return `"${trimmed}"`;
  }
  return trimmed;
}

export function App({ defaultNow, defaultTz }: AppProps) {
  const initial = useMemo(() => loadInitialState(defaultTz), [defaultTz]);

  const [engineReady, setEngineReady] = useState(
    () => typeof globalThis.ahoritaRun === "function"
  );
  const [initError, setInitError] = useState<string | null>(null);
  const [expr, setExpr] = useState<string>(initial.expr);
  const [settings, setSettings] = useState<StoredSettings>(initial.settings);
  const [activeTab, setActiveTab] = useState<CategoryId>("rel");
  const [themeOverride, setThemeOverride] = useState<ThemeOverride>(() =>
    readStoredThemeOverride()
  );
  const [systemDark, setSystemDark] = useState<boolean>(() => getSystemDark());
  const [zoneInsertInput, setZoneInsertInput] = useState<string>("");
  const [liveStatus, setLiveStatus] = useState<string>("");
  const [datePickerVal, setDatePickerVal] = useState<string>("");
  const [timePickerVal, setTimePickerVal] = useState<string>("");

  const exprInputRef = useRef<HTMLInputElement | null>(null);
  const settingsDialogRef = useRef<HTMLDialogElement | null>(null);
  const zoneOptions = useMemo(
    () => getAvailableTimeZones(settings.tz),
    [settings.tz]
  );

  // Initialize WASM once
  useEffect(() => {
    let active = true;
    initEngine()
      .then(() => {
        if (active) setEngineReady(true);
      })
      .catch((err: unknown) => {
        if (active) {
          setInitError(err instanceof Error ? err.message : String(err));
        }
      });
    return () => {
      active = false;
    };
  }, []);

  // Sync <html lang> and localStorage settings
  useEffect(() => {
    document.documentElement.lang = settings.lang;
    try {
      getSafeStorage()?.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(settings));
    } catch {
      // ignore storage errors
    }
  }, [settings]);

  // Mirror query + non-default display options in URL (?q=, &t=, &f=, &tz=, &lang=) (F.9 = A ★)
  useEffect(() => {
    if (typeof window === "undefined") return;
    const params = new URLSearchParams();
    if (expr.trim() !== "") {
      params.set("q", expr);
    }
    if (settings.type) {
      params.set("t", settings.type);
    }
    const effectiveFormat =
      settings.format === "custom" ? settings.customFormat : settings.format;
    if (effectiveFormat) {
      params.set("f", effectiveFormat);
    }
    if (settings.tz) {
      params.set("tz", settings.tz);
    }
    if (settings.lang !== "en") {
      params.set("lang", settings.lang);
    }
    const qs = params.toString();
    const targetUrl = qs
      ? `${window.location.pathname}?${qs}`
      : window.location.pathname;
    try {
      window.history.replaceState(null, "", targetUrl);
    } catch {
      // ignore sandboxed history errors
    }
  }, [
    expr,
    settings.type,
    settings.format,
    settings.customFormat,
    settings.tz,
    settings.lang,
  ]);

  // Listen to prefers-color-scheme changes and cross-tab theme storage updates without mutating stored override
  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key === THEME_STORAGE_KEY) {
        const next = readStoredThemeOverride();
        setThemeOverride(next);
        applyThemeOverride(next);
      }
    };
    window.addEventListener("storage", onStorage);

    if (typeof window.matchMedia === "function") {
      const mq = window.matchMedia("(prefers-color-scheme: dark)");
      const onMediaChange = (ev: MediaQueryListEvent) => {
        setSystemDark(ev.matches);
      };
      mq.addEventListener("change", onMediaChange);
      return () => {
        window.removeEventListener("storage", onStorage);
        mq.removeEventListener("change", onMediaChange);
      };
    }

    return () => {
      window.removeEventListener("storage", onStorage);
    };
  }, []);

  // Progressive enhancement: closedby="any" backdrop light-dismiss only if native support is missing
  useEffect(() => {
    const dialog = settingsDialogRef.current;
    if (!dialog) return;

    const onClose = () => {
      dialog.returnValue = "";
    };
    dialog.addEventListener("close", onClose);

    if ("closedBy" in HTMLDialogElement.prototype) {
      return () => {
        dialog.removeEventListener("close", onClose);
      };
    }

    const isOutside = (e: PointerEvent) => {
      const r = dialog.getBoundingClientRect();
      return (
        e.clientX < r.left ||
        e.clientX > r.right ||
        e.clientY < r.top ||
        e.clientY > r.bottom
      );
    };
    let pressedOutside = false;
    const onPointerDown = (e: PointerEvent) => {
      pressedOutside = dialog.open && isOutside(e);
    };
    const onPointerUp = (e: PointerEvent) => {
      if (dialog.open && pressedOutside && isOutside(e)) {
        dialog.close();
      }
      pressedOutside = false;
    };
    dialog.addEventListener("pointerdown", onPointerDown);
    dialog.addEventListener("pointerup", onPointerUp);
    return () => {
      dialog.removeEventListener("close", onClose);
      dialog.removeEventListener("pointerdown", onPointerDown);
      dialog.removeEventListener("pointerup", onPointerUp);
    };
  }, []);

  const buildRunOptions = useCallback(
    (overrides: Partial<RunOptions> = {}): RunOptions => {
      const effectiveFormat =
        settings.format === "custom" ? settings.customFormat : settings.format;
      const cleanDaysOff = settings.days_off
        .map((line) => line.trim())
        .filter((line) => line !== "" && !line.startsWith("#"));
      return {
        now: defaultNow,
        tz: settings.tz,
        type: settings.type,
        format: effectiveFormat,
        explain: settings.explain,
        lang: settings.lang,
        strict: settings.strict,
        week_start: settings.week_start,
        week_one: settings.week_one,
        month_overflow: settings.month_overflow,
        count_inclusive: settings.count_inclusive,
        country: settings.country,
        weekend:
          settings.weekend.length === 0 ? "none" : settings.weekend.join(","),
        days_off: cleanDaysOff,
        ...overrides,
      };
    },
    [defaultNow, settings]
  );

  const result: Result | null = useMemo(() => {
    if (!engineReady || expr.trim() === "") {
      return null;
    }
    return run(expr, buildRunOptions());
  }, [engineReady, expr, buildRunOptions]);

  // Reference year for holiday chips derived via engine when ready
  const referenceYear = useMemo(() => {
    if (!engineReady) return "2026";
    const yearRes = run("today", buildRunOptions({ type: "date", format: "yyyy", explain: false }));
    return yearRes.text && /^\d{4}$/.test(yearRes.text) ? yearRes.text : "2026";
  }, [engineReady, buildRunOptions]);

  const coreSections = useMemo(() => getCoreSections(settings.lang), [settings.lang]);
  const categoryTabs = useMemo(
    () => getCategoryTabs(settings.lang, settings.country, referenceYear),
    [settings.lang, settings.country, referenceYear]
  );

  const allButtons: KeypadButton[] = useMemo(() => {
    const list: KeypadButton[] = [
      ...coreSections.values,
      ...coreSections.operators,
      ...coreSections.digits,
      ...coreSections.units,
    ];
    for (const tab of categoryTabs) {
      list.push(...tab.buttons);
      if (tab.templates) list.push(...tab.templates);
    }
    return list;
  }, [coreSections, categoryTabs]);

  const enabledButtonIds = useMemo(
    () => computeEnabledButtonIds(expr, result, allButtons),
    [expr, result, allButtons]
  );

  const incomplete = isIncompleteParseError(expr, result);
  const fatalError = Boolean(result?.error && !incomplete && expr.trim() !== "");

  // Debounced polite announcement for evaluated output changes
  useEffect(() => {
    if (!result) return;
    const msg = result.text
      ? `Result: ${result.text}`
      : result.error && !incomplete
        ? `Error: ${result.error.message}`
        : "";
    if (!msg) return;
    const handle = window.setTimeout(() => {
      setLiveStatus(msg);
    }, 250);
    return () => window.clearTimeout(handle);
  }, [result, incomplete]);

  const setExprWithCaret = useCallback((nextText: string, nextCaret: number) => {
    setExpr(nextText);
    window.requestAnimationFrame(() => {
      const el = exprInputRef.current;
      if (el && document.activeElement === el) {
        el.setSelectionRange(nextCaret, nextCaret);
      }
    });
  }, []);

  const handleKeypadClick = useCallback(
    (btn: KeypadButton) => {
      const el = exprInputRef.current;
      const start = el?.selectionStart ?? expr.length;
      const end = el?.selectionEnd ?? expr.length;
      const { text, caret } = insertIntoExpression(
        expr,
        start,
        end,
        btn.insert,
        btn.kind
      );
      setExprWithCaret(text, caret);
    },
    [expr, setExprWithCaret]
  );

  const handleEditButton = useCallback(
    (action: "space" | "backspace" | "clear") => {
      if (action === "clear") {
        setExpr("");
        setLiveStatus("Expression cleared");
        exprInputRef.current?.focus();
        return;
      }
      const el = exprInputRef.current;
      const start = el?.selectionStart ?? expr.length;
      const end = el?.selectionEnd ?? expr.length;
      if (action === "space") {
        const next = expr.slice(0, start) + " " + expr.slice(end);
        setExprWithCaret(next, start + 1);
      } else {
        const { text, caret } = applyBackspace(expr, start, end);
        setExprWithCaret(text, caret);
      }
    },
    [expr, setExprWithCaret]
  );

  const handleDatePick = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      const val = e.target.value;
      setDatePickerVal(val);
      if (!val) return;
      const el = exprInputRef.current;
      const start = el?.selectionStart ?? expr.length;
      const end = el?.selectionEnd ?? expr.length;
      const { text, caret } = replaceTokenOrInsert(
        expr,
        start,
        end,
        /\d{4}-\d{2}-\d{2}/,
        val
      );
      setExprWithCaret(text, caret);
    },
    [expr, setExprWithCaret]
  );

  const handleTimePick = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      const val = e.target.value;
      setTimePickerVal(val);
      if (!val) return;
      const el = exprInputRef.current;
      const start = el?.selectionStart ?? expr.length;
      const end = el?.selectionEnd ?? expr.length;
      const { text, caret } = replaceTokenOrInsert(
        expr,
        start,
        end,
        /\b\d{1,2}:\d{2}\b/,
        val
      );
      setExprWithCaret(text, caret);
    },
    [expr, setExprWithCaret]
  );

  const handleInsertZone = useCallback(() => {
    const formatted = formatZoneForInsert(zoneInsertInput);
    if (!formatted) return;
    const el = exprInputRef.current;
    const start = el?.selectionStart ?? expr.length;
    const end = el?.selectionEnd ?? expr.length;
    const { text, caret } = insertIntoExpression(
      expr,
      start,
      end,
      formatted,
      "word"
    );
    setExprWithCaret(text, caret);
    setZoneInsertInput("");
  }, [expr, zoneInsertInput, setExprWithCaret]);

  const copyToClipboard = useCallback(async (text: string, label: string) => {
    if (!text) return;
    try {
      await navigator.clipboard?.writeText(text);
      setLiveStatus(`${label} copied`);
    } catch {
      setLiveStatus(`Could not copy ${label.toLowerCase()}`);
    }
  }, []);

  const canDownloadICS = useMemo(() => {
    if (!result || result.error) return false;
    if (result.kind === "text" && result.text?.startsWith("BEGIN:VCALENDAR")) {
      return true;
    }
    return (
      result.kind === "recurrence" ||
      result.kind === "date" ||
      result.kind === "datetime" ||
      result.kind === "span"
    );
  }, [result]);

  const handleDownloadICS = useCallback(() => {
    if (!engineReady || !result || result.error) return;
    const icsText =
      result.kind === "text" && result.text?.startsWith("BEGIN:VCALENDAR")
        ? result.text
        : run(`${expr}, to ics`, buildRunOptions({ explain: false })).text;
    if (!icsText || !icsText.startsWith("BEGIN:VCALENDAR")) {
      setLiveStatus("Unable to export .ics for this expression");
      return;
    }
    const blob = new Blob([icsText], { type: "text/calendar;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "ahorita.ics";
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
    setLiveStatus("ahorita.ics downloaded");
  }, [engineReady, result, expr, buildRunOptions]);

  const handleThemeHeaderToggle = useCallback(() => {
    const renderedDark =
      themeOverride === "dark" || (themeOverride === "system" && systemDark);
    const target: ThemeOverride = renderedDark ? "light" : "dark";
    const osTarget: "light" | "dark" = systemDark ? "dark" : "light";
    // Divergence checked only at storage time (dark-mode-toggle rule): revert to system if target matches OS
    const next: ThemeOverride = target === osTarget ? "system" : target;
    setThemeOverride(next);
    applyThemeOverride(next);
  }, [themeOverride, systemDark]);

  const handleThemeSelect = useCallback((next: ThemeOverride) => {
    setThemeOverride(next);
    applyThemeOverride(next);
  }, []);

  // Global keyboard shortcuts (/, Ctrl/Cmd+K, Ctrl/Cmd+Shift+C, Alt+E, Esc when modal closed)
  useEffect(() => {
    const onKeyDown = (e: globalThis.KeyboardEvent) => {
      if (settingsDialogRef.current?.open) {
        return;
      }
      const target = e.target as HTMLElement | null;
      const isTextField =
        target?.tagName === "INPUT" ||
        target?.tagName === "TEXTAREA" ||
        target?.tagName === "SELECT";

      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        exprInputRef.current?.focus();
        return;
      }
      if (e.key === "/" && !isTextField) {
        e.preventDefault();
        exprInputRef.current?.focus();
        return;
      }
      if (
        (e.ctrlKey || e.metaKey) &&
        e.shiftKey &&
        e.key.toLowerCase() === "c"
      ) {
        if (result?.text) {
          e.preventDefault();
          void copyToClipboard(result.text, "Result");
        }
        return;
      }
      if (e.altKey && e.key.toLowerCase() === "e") {
        e.preventDefault();
        setSettings((prev) => ({ ...prev, explain: !prev.explain }));
        return;
      }
      if (e.key === "Escape" && expr !== "") {
        e.preventDefault();
        setExpr("");
        setLiveStatus("Expression cleared");
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [result, expr, copyToClipboard]);

  const handleTabKeyDown = (
    e: KeyboardEvent<HTMLButtonElement>,
    idx: number
  ) => {
    if (e.key === "ArrowRight") {
      e.preventDefault();
      setActiveTab(categoryTabs[(idx + 1) % categoryTabs.length].id);
    } else if (e.key === "ArrowLeft") {
      e.preventDefault();
      setActiveTab(
        categoryTabs[(idx - 1 + categoryTabs.length) % categoryTabs.length].id
      );
    } else if (e.key === "Home") {
      e.preventDefault();
      setActiveTab(categoryTabs[0].id);
    } else if (e.key === "End") {
      e.preventDefault();
      setActiveTab(categoryTabs[categoryTabs.length - 1].id);
    }
  };

  const activeCategoryObj =
    categoryTabs.find((t) => t.id === activeTab) ?? categoryTabs[0];

  const renderErrorMirror = () => {
    if (!result?.error || result.error.pos < 0 || expr === "") return null;
    const idx = utf8ByteOffsetToUtf16Index(expr, result.error.pos);
    const prefix = expr.slice(0, idx);
    const badChar = expr.slice(idx, idx + 1) || " ";
    const suffix = expr.slice(idx + 1);
    return (
      <div className="error-mirror" aria-hidden="true">
        <span>{prefix}</span>
        <span className="error-underline">{badChar}</span>
        <span>{suffix}</span>
      </div>
    );
  };

  const isMultiLine =
    result?.kind === "list" ||
    result?.kind === "text" ||
    Boolean(result?.text?.includes("\n"));

  const isES = settings.lang === "es";

  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="brand">
          <h1>ahorita</h1>
          <span className="brand-subtitle">
            {isES ? "Calculadora de fechas" : "Date & time calculator"}
          </span>
        </div>

        <div className="header-controls">
          <label className="inline-field" htmlFor="header-tz">
            <span>{isES ? "Zona" : "Zone"}</span>
            <select
              id="header-tz"
              value={settings.tz}
              onChange={(e) =>
                setSettings((prev) => ({ ...prev, tz: e.target.value }))
              }
            >
              {zoneOptions.map((z) => (
                <option key={z} value={z}>
                  {z}
                </option>
              ))}
            </select>
          </label>

          <label className="inline-field" htmlFor="header-lang">
            <span>{isES ? "Idioma" : "Lang"}</span>
            <select
              id="header-lang"
              value={settings.lang}
              onChange={(e) =>
                setSettings((prev) => ({
                  ...prev,
                  lang: e.target.value as Language,
                }))
              }
            >
              <option value="en">EN</option>
              <option value="es">ES</option>
            </select>
          </label>

          <button
            type="button"
            onClick={handleThemeHeaderToggle}
            aria-label={
              themeOverride === "dark" ||
              (themeOverride === "system" && systemDark)
                ? "Switch to light theme"
                : "Switch to dark theme"
            }
          >
            <span className="theme-toggle-opt-light">☀ Light</span>
            <span className="theme-toggle-opt-dark">☾ Dark</span>
          </button>

          <button
            type="button"
            onClick={() => settingsDialogRef.current?.showModal()}
          >
            {isES ? "Ajustes" : "Settings"}
          </button>
        </div>
      </header>

      <main className="workspace">
        <div>
          <section className="card sticky-mobile-display">
            <div className="result-header">
              <h2>{isES ? "Expresión y resultado" : "Expression & Result"}</h2>
              {result?.kind && (
                <span className="kind-badge" data-testid="result-kind">
                  {result.kind}
                </span>
              )}
            </div>

            {initial.tzFallbackNotice && (
              <div className="note-banner" role="status">
                Browser timezone unavailable; using UTC until another zone is
                selected.
              </div>
            )}

            {initError && <div className="error-banner" role="alert">{initError}</div>}
            {!engineReady && !initError && (
              <p className="result-placeholder">Loading WebAssembly engine…</p>
            )}

            <label className="visually-hidden" htmlFor="expr-input">
              {isES ? "Expresión de fecha o tiempo" : "Date or time expression"}
            </label>
            <div className="expr-row">
              <input
                id="expr-input"
                ref={exprInputRef}
                type="text"
                className="expr-input"
                value={expr}
                onChange={(e) => setExpr(e.target.value)}
                aria-invalid={fatalError ? "true" : undefined}
                aria-describedby={fatalError ? "expr-error-msg" : undefined}
                placeholder={isES ? "ej. hoy + 2d" : "e.g. now + 2d"}
              />
              <button
                type="button"
                onClick={() => handleEditButton("clear")}
                aria-label="Clear expression"
              >
                Clear
              </button>
            </div>

            {renderErrorMirror()}

            {incomplete && result?.error && (
              <div className="hint-banner" role="status">
                <strong>{isES ? "Esperado:" : "Expected:"}</strong>{" "}
                {result.error.expected?.join(", ")}
              </div>
            )}

            {fatalError && result?.error && (
              <div id="expr-error-msg" className="error-banner" role="alert">
                {result.error.pos >= 0 ? `Position ${result.error.pos}: ` : ""}
                {result.error.message}
              </div>
            )}

            <div aria-live="off" className="result-box" data-testid="result-box">
              {expr.trim() === "" ? (
                <p className="result-placeholder">
                  {isES
                    ? 'Escribe o toca una expresión, ej. "hoy + 2d"'
                    : 'Type or tap an expression, e.g. "now + 2d"'}
                </p>
              ) : result?.text !== undefined && !result.error ? (
                isMultiLine ? (
                  <pre
                    className="result-pre"
                    role="region"
                    aria-label="Formatted result output"
                    tabIndex={0}
                  >
                    <code data-testid="result-text">{result.text}</code>
                  </pre>
                ) : (
                  <p className="result-value" data-testid="result-text">
                    {result.text}
                  </p>
                )
              ) : null}

              {result && !result.error && (
                <div className="result-meta">
                  {result.iso && result.iso !== result.text && (
                    <span>ISO: {result.iso}</span>
                  )}
                  {result.unix !== undefined && (
                    <span>Unix: {result.unix}</span>
                  )}
                  {result.start && result.end && (
                    <span>
                      Range: {result.start} → {result.end}
                    </span>
                  )}
                </div>
              )}
            </div>

            {result?.note && (
              <div className="note-banner" role="status">
                <strong>Note:</strong> {result.note}
              </div>
            )}

            <div className="action-bar">
              <button
                type="button"
                disabled={!expr.trim()}
                onClick={() => void copyToClipboard(expr, "Expression")}
              >
                Copy Expr
              </button>
              <button
                type="button"
                disabled={!result?.text || Boolean(result?.error)}
                onClick={() =>
                  result?.text && void copyToClipboard(result.text, "Result")
                }
              >
                Copy Result
              </button>
              <button
                type="button"
                disabled={!expr.trim()}
                onClick={() =>
                  void copyToClipboard(window.location.href, "Share link")
                }
              >
                Share
              </button>
              {canDownloadICS && (
                <button type="button" onClick={handleDownloadICS}>
                  Download .ics
                </button>
              )}
            </div>
          </section>

          <section className="card">
            <h2>{isES ? "Opciones de salida" : "Output Options"}</h2>
            <div className="options-stack">
              <div>
                <div className="pad-label">{isES ? "Tipo" : "Type"}</div>
                <div
                  className="pill-row"
                  role="group"
                  aria-label="Output type"
                >
                  {(
                    [
                      { id: "", label: "Auto" },
                      { id: "date", label: "date" },
                      { id: "datetime", label: "datetime" },
                      { id: "time", label: "time" },
                      { id: "range", label: "range" },
                      { id: "relative", label: "relative" },
                    ] as const
                  ).map((opt) => (
                    <button
                      key={opt.label}
                      type="button"
                      aria-pressed={settings.type === opt.id}
                      onClick={() =>
                        setSettings((prev) => ({ ...prev, type: opt.id }))
                      }
                    >
                      {opt.label}
                    </button>
                  ))}
                </div>
              </div>

              <div className="format-row">
                <label className="inline-field" htmlFor="format-select">
                  <span>{isES ? "Formato" : "Format"}</span>
                  <select
                    id="format-select"
                    value={settings.format}
                    onChange={(e) =>
                      setSettings((prev) => ({
                        ...prev,
                        format: e.target.value,
                      }))
                    }
                  >
                    <option value="">default (iso)</option>
                    <option value="iso">iso</option>
                    <option value="rfc3339">rfc3339</option>
                    <option value="unix">unix</option>
                    <option value="human">human</option>
                    <option value="short">short</option>
                    <option value="custom">custom pattern…</option>
                  </select>
                </label>

                <label className="setting-checkbox" htmlFor="explain-toggle">
                  <input
                    id="explain-toggle"
                    type="checkbox"
                    checked={settings.explain}
                    onChange={(e) =>
                      setSettings((prev) => ({
                        ...prev,
                        explain: e.target.checked,
                      }))
                    }
                  />
                  <span>{isES ? "Explicar pasos" : "Explain steps"}</span>
                </label>
              </div>

              {settings.format === "custom" && (
                <div>
                  <label
                    className="inline-field"
                    htmlFor="format-custom-pattern"
                  >
                    <span>Pattern</span>
                    <input
                      id="format-custom-pattern"
                      type="text"
                      value={settings.customFormat}
                      onChange={(e) =>
                        setSettings((prev) => ({
                          ...prev,
                          customFormat: e.target.value,
                        }))
                      }
                    />
                  </label>
                  <div className="pill-row" style={{ marginTop: "0.35rem" }}>
                    {PATTERN_CHIPS.map((tok) => (
                      <button
                        key={tok}
                        type="button"
                        onClick={() =>
                          setSettings((prev) => ({
                            ...prev,
                            customFormat: prev.customFormat + tok,
                          }))
                        }
                      >
                        {tok}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {settings.explain &&
                result?.explain &&
                result.explain.length > 0 && (
                  <table className="explain-table">
                    <caption>
                      {isES ? "Pasos de evaluación" : "Pipeline stages"}
                    </caption>
                    <thead>
                      <tr>
                        <th scope="col">#</th>
                        <th scope="col">Stage</th>
                        <th scope="col">Value</th>
                      </tr>
                    </thead>
                    <tbody>
                      {result.explain.map((step, idx) => (
                        <tr key={`${idx}-${step.stage}`}>
                          <td>{idx + 1}</td>
                          <td>{step.stage}</td>
                          <td>{step.value}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
            </div>
          </section>
        </div>

        <section className="card">
          <h2>{isES ? "Teclado y selectores" : "Keypad & Pickers"}</h2>

          <div className="pad-group">
            <div className="pad-label">{isES ? "Valores" : "Values"}</div>
            <div className="pad-grid">
              {coreSections.values.map((btn) => (
                <button
                  key={btn.id}
                  type="button"
                  disabled={!enabledButtonIds.has(btn.id)}
                  aria-label={btn.ariaLabel}
                  onClick={() => handleKeypadClick(btn)}
                >
                  {btn.label}
                </button>
              ))}
            </div>
          </div>

          <div className="pad-group">
            <div className="pad-label">
              {isES ? "Operadores" : "Operators"}
            </div>
            <div className="pad-grid">
              {coreSections.operators.map((btn) => (
                <button
                  key={btn.id}
                  type="button"
                  disabled={!enabledButtonIds.has(btn.id)}
                  aria-label={btn.ariaLabel}
                  onClick={() => handleKeypadClick(btn)}
                >
                  {btn.label}
                </button>
              ))}
            </div>
          </div>

          <div className="pad-group">
            <div className="pad-label">{isES ? "Unidades" : "Units"}</div>
            <div className="pad-grid">
              {coreSections.units.map((btn) => (
                <button
                  key={btn.id}
                  type="button"
                  disabled={!enabledButtonIds.has(btn.id)}
                  aria-label={btn.ariaLabel}
                  onClick={() => handleKeypadClick(btn)}
                >
                  {btn.label}
                </button>
              ))}
            </div>
          </div>

          <div className="pad-group">
            <div className="pad-label">
              {isES ? "Números y puntuación" : "Numbers & Punctuation"}
            </div>
            <div className="num-grid">
              {coreSections.digits.map((btn) => (
                <button
                  key={btn.id}
                  type="button"
                  disabled={!enabledButtonIds.has(btn.id)}
                  aria-label={btn.ariaLabel}
                  onClick={() => handleKeypadClick(btn)}
                >
                  {btn.label}
                </button>
              ))}
              <button
                type="button"
                aria-label="Space"
                onClick={() => handleEditButton("space")}
              >
                Space
              </button>
              <button
                type="button"
                aria-label="⌫ Backspace"
                onClick={() => handleEditButton("backspace")}
              >
                ⌫
              </button>
            </div>
          </div>

          <nav aria-label="Keypad categories">
            <div className="keypad-tabs" role="tablist">
              {categoryTabs.map((tab, idx) => (
                <button
                  key={tab.id}
                  id={`tab-${tab.id}`}
                  type="button"
                  role="tab"
                  aria-selected={activeTab === tab.id}
                  aria-controls={`panel-${tab.id}`}
                  onClick={() => setActiveTab(tab.id)}
                  onKeyDown={(e) => handleTabKeyDown(e, idx)}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </nav>

          <div
            id={`panel-${activeCategoryObj.id}`}
            role="tabpanel"
            aria-labelledby={`tab-${activeCategoryObj.id}`}
          >
            <div className="pad-grid">
              {activeCategoryObj.buttons.map((btn) => (
                <button
                  key={btn.id}
                  type="button"
                  disabled={!enabledButtonIds.has(btn.id)}
                  aria-label={btn.ariaLabel}
                  onClick={() => handleKeypadClick(btn)}
                >
                  {btn.label}
                </button>
              ))}
            </div>

            {activeCategoryObj.templates &&
              activeCategoryObj.templates.length > 0 && (
                <div className="template-chips">
                  <div className="pad-label">
                    {isES ? "Plantillas" : "Quick Templates"}
                  </div>
                  <div className="pad-grid">
                    {activeCategoryObj.templates.map((btn) => (
                      <button
                        key={btn.id}
                        type="button"
                        disabled={!enabledButtonIds.has(btn.id)}
                        aria-label={btn.ariaLabel}
                        onClick={() => handleKeypadClick(btn)}
                      >
                        {btn.label}
                      </button>
                    ))}
                  </div>
                </div>
              )}
          </div>

          <div className="pickers-grid">
            <div className="picker-field">
              <label htmlFor="picker-date">
                {isES ? "Selector de fecha" : "Date picker"}
              </label>
              <input
                id="picker-date"
                type="date"
                value={datePickerVal}
                onChange={handleDatePick}
              />
            </div>

            <div className="picker-field">
              <label htmlFor="picker-time">
                {isES ? "Selector de hora" : "Time picker"}
              </label>
              <input
                id="picker-time"
                type="time"
                value={timePickerVal}
                onChange={handleTimePick}
              />
            </div>

            <div className="picker-field">
              <label htmlFor="picker-zone">
                {isES ? "Insertar zona" : "Insert zone"}
              </label>
              <div className="picker-inline">
                <input
                  id="picker-zone"
                  type="text"
                  list="iana-zones"
                  value={zoneInsertInput}
                  onChange={(e) => setZoneInsertInput(e.target.value)}
                />
                <button type="button" onClick={handleInsertZone}>
                  {isES ? "Insertar" : "Insert"}
                </button>
              </div>
              <datalist id="iana-zones">
                {zoneOptions.map((z) => (
                  <option key={z} value={z} />
                ))}
              </datalist>
            </div>
          </div>
        </section>
      </main>

      <footer className="app-footer">
        <span>
          {isES
            ? "Un motor (Go WASM) · Sin lógica de fechas en JavaScript"
            : "One engine (Go WASM) · Zero date math in JavaScript"}
        </span>
        <span>Shortcuts: `/` focus · `Ctrl/Cmd+Shift+C` copy · `Alt+E` explain</span>
      </footer>

      <div className="visually-hidden" aria-live="polite" aria-atomic="true">
        {liveStatus}
      </div>

      <dialog
        ref={settingsDialogRef}
        aria-labelledby="settings-dialog-title"
        {...({ closedby: "any" } as Record<string, string>)}
      >
        <div className="dialog-header">
          <h1 id="settings-dialog-title">
            {isES ? "Ajustes del motor" : "Engine Settings"}
          </h1>
          <form method="dialog">
            <button type="submit">{isES ? "Cerrar" : "Close"}</button>
          </form>
        </div>

        <div className="settings-grid">
          <div className="setting-item">
            <label htmlFor="setting-theme">Theme</label>
            <select
              id="setting-theme"
              value={themeOverride}
              onChange={(e) =>
                handleThemeSelect(e.target.value as ThemeOverride)
              }
            >
              <option value="system">
                System (Currently {systemDark ? "dark" : "light"})
              </option>
              <option value="light">Light</option>
              <option value="dark">Dark</option>
            </select>
          </div>

          <div className="setting-item">
            <label htmlFor="setting-week-start">Week starts on</label>
            <select
              id="setting-week-start"
              value={settings.week_start}
              onChange={(e) =>
                setSettings((prev) => ({
                  ...prev,
                  week_start: e.target.value,
                }))
              }
            >
              {WEEKDAYS_LIST.map((d) => (
                <option key={d} value={d}>
                  {d}
                </option>
              ))}
            </select>
          </div>

          <div className="setting-item">
            <label htmlFor="setting-week-one">First week rule</label>
            <select
              id="setting-week-one"
              value={settings.week_one}
              onChange={(e) =>
                setSettings((prev) => ({
                  ...prev,
                  week_one: e.target.value as WeekOneRule,
                }))
              }
            >
              <option value="iso">iso</option>
              <option value="first-day">first-day</option>
              <option value="full">full</option>
              <option value="days">days</option>
            </select>
          </div>

          <div className="setting-item">
            <label htmlFor="setting-month-overflow">Month overflow</label>
            <select
              id="setting-month-overflow"
              value={settings.month_overflow}
              onChange={(e) =>
                setSettings((prev) => ({
                  ...prev,
                  month_overflow: e.target.value as MonthOverflowRule,
                }))
              }
            >
              <option value="clamp">clamp</option>
              <option value="overflow">overflow</option>
            </select>
          </div>

          <div className="setting-item">
            <label htmlFor="setting-country">Holiday country code</label>
            <input
              id="setting-country"
              type="text"
              value={settings.country}
              onChange={(e) =>
                setSettings((prev) => ({
                  ...prev,
                  country: e.target.value,
                }))
              }
            />
          </div>

          <div className="setting-item">
            <label className="setting-checkbox" htmlFor="setting-strict">
              <input
                id="setting-strict"
                type="checkbox"
                checked={settings.strict}
                onChange={(e) =>
                  setSettings((prev) => ({
                    ...prev,
                    strict: e.target.checked,
                  }))
                }
              />
              <span>Strict mode</span>
            </label>

            <label className="setting-checkbox" htmlFor="setting-inclusive">
              <input
                id="setting-inclusive"
                type="checkbox"
                checked={settings.count_inclusive}
                onChange={(e) =>
                  setSettings((prev) => ({
                    ...prev,
                    count_inclusive: e.target.checked,
                  }))
                }
              />
              <span>Count endpoints inclusive</span>
            </label>
          </div>
        </div>

        <fieldset style={{ marginTop: "0.75rem", borderRadius: "8px", border: "1px solid var(--border)" }}>
          <legend>Weekend days</legend>
          <div className="weekend-chips">
            {WEEKDAYS_LIST.map((day) => {
              const checked = settings.weekend.includes(day);
              return (
                <label key={day} className="weekend-chip" htmlFor={`wk-${day}`}>
                  <input
                    id={`wk-${day}`}
                    type="checkbox"
                    checked={checked}
                    onChange={(e) => {
                      const next = e.target.checked
                        ? [...settings.weekend, day]
                        : settings.weekend.filter((d) => d !== day);
                      setSettings((prev) => ({ ...prev, weekend: next }));
                    }}
                  />
                  <span>{day.slice(0, 3)}</span>
                </label>
              );
            })}
          </div>
        </fieldset>

        <div className="setting-item" style={{ marginTop: "0.75rem" }}>
          <label htmlFor="setting-days-off">
            Custom days off (one expression or span per line)
          </label>
          <textarea
            id="setting-days-off"
            rows={4}
            value={settings.days_off.join("\n")}
            onChange={(e) =>
              setSettings((prev) => ({
                ...prev,
                days_off: e.target.value.split("\n"),
              }))
            }
          />
        </div>
      </dialog>
    </div>
  );
}

export default App;
