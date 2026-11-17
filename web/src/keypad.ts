import type { Language, Result } from "./engine";

export type CategoryId = "rel" | "periods" | "zones" | "bizdays" | "recur";
export type InsertKind = "word" | "digit" | "unit" | "op" | "punct" | "cron";

export interface KeypadButton {
  id: string;
  label: string;
  insert: string;
  kind: InsertKind;
  ariaLabel?: string;
  /** Semantic expected tokens / contexts where this button is enabled. */
  roles: string[];
}

export interface CategoryTab {
  id: CategoryId;
  label: string;
  buttons: KeypadButton[];
  templates?: KeypadButton[];
}

export interface CoreSections {
  values: KeypadButton[];
  operators: KeypadButton[];
  digits: KeypadButton[];
  units: KeypadButton[];
}

const ES_WORDS: Record<string, { label: string; insert: string }> = {
  now: { label: "ahora", insert: "ahora" },
  today: { label: "hoy", insert: "hoy" },
  tomorrow: { label: "mañana", insert: "mañana" },
  yesterday: { label: "ayer", insert: "ayer" },
  to: { label: "a", insert: "a" },
  in: { label: "en", insert: "en" },
  bd: { label: "días hábiles", insert: "días hábiles" },
  ww: { label: "sem. lab.", insert: "semana laboral" },
  last: { label: "pasado", insert: "pasado" },
  next: { label: "próximo", insert: "próximo" },
  this: { label: "este", insert: "este" },
  closest: { label: "más cercano", insert: "más cercano" },
  ago: { label: "hace", insert: "hace" },
  inclusive: { label: "inclusive", insert: "inclusive" },
  start_of: { label: "inicio de", insert: "inicio de" },
  end_of: { label: "fin de", insert: "fin de" },
  from: { label: "desde", insert: "desde" },
  between: { label: "entre", insert: "entre" },
  and: { label: "y", insert: "y" },
  during: { label: "durante", insert: "durante" },
  starting: { label: "empezando", insert: "empezando" },
  how_many: { label: "cuántos", insert: "cuántos" },
  second: { label: "segundo", insert: "segundo" },
  minute: { label: "minuto", insert: "minuto" },
  hour: { label: "hora", insert: "hora" },
  day: { label: "día", insert: "día" },
  week: { label: "semana", insert: "semana" },
  month: { label: "mes", insert: "mes" },
  quarter: { label: "trimestre", insert: "trimestre" },
  half: { label: "semestre", insert: "semestre" },
  year: { label: "año", insert: "año" },
  mon: { label: "lun", insert: "lunes" },
  tue: { label: "mar", insert: "martes" },
  wed: { label: "mié", insert: "miércoles" },
  thu: { label: "jue", insert: "jueves" },
  fri: { label: "vie", insert: "viernes" },
  sat: { label: "sáb", insert: "sábado" },
  sun: { label: "dom", insert: "domingo" },
  jan: { label: "ene", insert: "enero" },
  feb: { label: "feb", insert: "febrero" },
  mar: { label: "mar", insert: "marzo" },
  apr: { label: "abr", insert: "abril" },
  may: { label: "may", insert: "mayo" },
  jun: { label: "jun", insert: "junio" },
  jul: { label: "jul", insert: "julio" },
  aug: { label: "ago", insert: "agosto" },
  sep: { label: "sep", insert: "septiembre" },
  oct: { label: "oct", insert: "octubre" },
  nov: { label: "nov", insert: "noviembre" },
  dec: { label: "dic", insert: "diciembre" },
  ord1: { label: "1ro", insert: "1st" },
  ord2: { label: "2do", insert: "2nd" },
  ord3: { label: "3ro", insert: "3rd" },
  ord4: { label: "4to", insert: "4th" },
  ord5: { label: "5to", insert: "5th" },
  of: { label: "de", insert: "de" },
  bizday: { label: "día hábil", insert: "día hábil" },
  bizdays: { label: "días hábiles", insert: "días hábiles" },
  workweek: { label: "semana laboral", insert: "semana laboral" },
  holidays: { label: "feriados", insert: "feriados" },
  every: { label: "cada", insert: "cada" },
  weekdaykw: { label: "día de semana", insert: "día de semana" },
  explain: { label: "explicar", insert: "explicar" },
  to_cron: { label: "a cron", insert: "a cron" },
  to_ics: { label: "a ics", insert: "a ics" },
  to_grid: { label: "a calendario", insert: "a calendario" },
  tpl_next_biz_after: {
    label: "próximo día hábil on or after",
    insert: "próximo día hábil on or after",
  },
  tpl_last_biz_before: {
    label: "último día hábil on or before",
    insert: "último día hábil on or before",
  },
  tpl_to_biz: {
    label: ", a días hábiles",
    insert: ", a días hábiles",
  },
  tpl_next_biz: {
    label: ", próximo día hábil",
    insert: ", próximo día hábil",
  },
  tpl_every_mon_9: {
    label: "cada lunes 9am",
    insert: "cada lunes 9am",
  },
  tpl_every_weekday: {
    label: "cada día de semana",
    insert: "cada día de semana",
  },
  tpl_every_2nd_mon: {
    label: "cada 2nd lunes",
    insert: "cada 2nd lunes",
  },
};

function locWord(
  lang: Language,
  id: string,
  enLabel: string,
  enInsert = enLabel
): { label: string; insert: string } {
  if (lang === "es" && ES_WORDS[id]) {
    return ES_WORDS[id];
  }
  return { label: enLabel, insert: enInsert };
}

export function getCoreSections(lang: Language): CoreSections {
  const w = (
    id: string,
    enLabel: string,
    roles: string[],
    enInsert = enLabel,
    kind: InsertKind = "word",
    ariaLabel?: string
  ): KeypadButton => {
    const t = locWord(lang, id, enLabel, enInsert);
    return {
      id,
      label: t.label,
      insert: t.insert,
      kind,
      ariaLabel,
      roles,
    };
  };

  return {
    values: [
      w("now", "now", ["value"]),
      w("today", "today", ["value"]),
      w("tomorrow", "tomorrow", ["value"]),
      w("yesterday", "yesterday", ["value"]),
    ],
    operators: [
      { id: "plus", label: "+", insert: "+", kind: "op", ariaLabel: "+ (add)", roles: ["after", "timezone_sign"] },
      { id: "minus", label: "−", insert: "-", kind: "op", ariaLabel: "− (subtract)", roles: ["after", "start_neg", "timezone_sign", "cron_part"] },
      { id: "mul", label: "×", insert: "*", kind: "op", ariaLabel: "× (multiply)", roles: ["after", "cron_part"] },
      { id: "comma", label: ",", insert: ",", kind: "punct", ariaLabel: ", (comma stage separator)", roles: ["after", "cron_part"] },
      { id: "pipe", label: "|", insert: "|", kind: "op", ariaLabel: "| (pipe stage separator)", roles: ["after"] },
      { id: "dotdot", label: "..", insert: "..", kind: "op", ariaLabel: ".. (range span)", roles: ["after"] },
      { id: "at", label: "@", insert: "@", kind: "op", ariaLabel: "@ (interpret in timezone)", roles: ["after", "timezone_prefix"] },
      w("to", "to", ["after", "kw_to"]),
      w("in", "in", ["value", "after"]),
    ],
    digits: [
      { id: "d7", label: "7", insert: "7", kind: "digit", roles: ["digit", "value", "duration", "unix", "weeknum", "time", "timezone_digit", "cron_part"] },
      { id: "d8", label: "8", insert: "8", kind: "digit", roles: ["digit", "value", "duration", "unix", "weeknum", "time", "timezone_digit", "cron_part"] },
      { id: "d9", label: "9", insert: "9", kind: "digit", roles: ["digit", "value", "duration", "unix", "weeknum", "time", "timezone_digit", "cron_part"] },
      { id: "d4", label: "4", insert: "4", kind: "digit", roles: ["digit", "value", "duration", "unix", "weeknum", "time", "timezone_digit", "cron_part"] },
      { id: "d5", label: "5", insert: "5", kind: "digit", roles: ["digit", "value", "duration", "unix", "weeknum", "time", "timezone_digit", "cron_part"] },
      { id: "d6", label: "6", insert: "6", kind: "digit", roles: ["digit", "value", "duration", "unix", "weeknum", "time", "timezone_digit", "cron_part"] },
      { id: "d1", label: "1", insert: "1", kind: "digit", roles: ["digit", "value", "duration", "unix", "weeknum", "time", "timezone_digit", "cron_part"] },
      { id: "d2", label: "2", insert: "2", kind: "digit", roles: ["digit", "value", "duration", "unix", "weeknum", "time", "timezone_digit", "cron_part"] },
      { id: "d3", label: "3", insert: "3", kind: "digit", roles: ["digit", "value", "duration", "unix", "weeknum", "time", "timezone_digit", "cron_part"] },
      { id: "d0", label: "0", insert: "0", kind: "digit", roles: ["digit", "value", "duration", "unix", "weeknum", "time", "timezone_digit", "cron_part"] },
      { id: "dot", label: ".", insert: ".", kind: "punct", ariaLabel: ". (decimal point)", roles: ["digit", "duration"] },
      { id: "colon", label: ":", insert: ":", kind: "punct", ariaLabel: ": (time colon)", roles: ["digit", "time", "timezone_digit", "after"] },
      { id: "slash", label: "/", insert: "/", kind: "punct", ariaLabel: "/ (slash date or cron step)", roles: ["digit", "value", "cron_part"] },
      { id: "quote", label: "\"", insert: "\"", kind: "punct", ariaLabel: "\" (quotation mark)", roles: ["value", "timezone", "cron_quote"] },
    ],
    units: [
      { id: "u_s", label: "s", insert: "s", kind: "unit", ariaLabel: "s (seconds)", roles: ["unit", "duration", "after_digit"] },
      { id: "u_m", label: "m", insert: "m", kind: "unit", ariaLabel: "m (minutes, lowercase)", roles: ["unit", "duration", "after_digit"] },
      { id: "u_h", label: "h", insert: "h", kind: "unit", ariaLabel: "h (hours)", roles: ["unit", "duration", "after_digit"] },
      { id: "u_d", label: "d", insert: "d", kind: "unit", ariaLabel: "d (days)", roles: ["unit", "duration", "after_digit"] },
      { id: "u_w", label: "w", insert: "w", kind: "unit", ariaLabel: "w (weeks)", roles: ["unit", "duration", "after_digit"] },
      { id: "u_M", label: "M", insert: "M", kind: "unit", ariaLabel: "M (months, capital)", roles: ["unit", "duration", "after_digit"] },
      { id: "u_q", label: "q", insert: "q", kind: "unit", ariaLabel: "q (quarters)", roles: ["unit", "duration", "after_digit"] },
      { id: "u_y", label: "y", insert: "y", kind: "unit", ariaLabel: "y (years)", roles: ["unit", "duration", "after_digit"] },
      {
        id: "bd",
        label: lang === "es" ? "días hábiles" : "bd",
        insert: lang === "es" ? "días hábiles" : "business days",
        kind: "word",
        ariaLabel: lang === "es" ? "días hábiles (unidades)" : "bd (business days)",
        roles: ["unit", "duration", "after_digit"],
      },
      {
        id: "ww",
        label: lang === "es" ? "sem. lab." : "ww",
        insert: lang === "es" ? "semana laboral" : "work week",
        kind: "word",
        ariaLabel: lang === "es" ? "sem. lab. (semana laboral)" : "ww (work week)",
        roles: ["unit", "duration", "period", "after_digit"],
      },
    ],
  };
}

export function getCategoryTabs(
  lang: Language,
  country: string,
  referenceYear: string
): CategoryTab[] {
  const b = (
    id: string,
    enLabel: string,
    roles: string[],
    enInsert = enLabel,
    kind: InsertKind = "word",
    ariaLabel?: string
  ): KeypadButton => {
    const t = locWord(lang, id, enLabel, enInsert);
    return {
      id,
      label: t.label,
      insert: t.insert,
      kind,
      ariaLabel,
      roles,
    };
  };

  const holidayCountry =
    country && country.toLowerCase() !== "none" ? country.toUpperCase() : "MX";
  const holidaysChip =
    lang === "es"
      ? `feriados ${holidayCountry} ${referenceYear}`
      : `holidays ${holidayCountry} ${referenceYear}`;

  return [
    {
      id: "rel",
      label: lang === "es" ? "Rel / Rango" : "Rel / Range",
      buttons: [
        b("last", "last", ["value", "kw_last"]),
        b("next", "next", ["value", "kw_next"]),
        b("this", "this", ["value", "kw_this"]),
        b("closest", "closest", ["value"]),
        b("ago", "ago", ["after"]),
        b("inclusive", "inclusive", ["after"]),
        b("start_of", "start of", ["value", "after"]),
        b("end_of", "end of", ["value", "after"]),
        b("from", "from", ["value", "kw_from"]),
        b("between", "between", ["value", "kw_between"]),
        b("and", "and", ["kw_and"]),
        b("during", "during", ["value"]),
        b("starting", "starting", ["after"]),
        b("how_many", "how many", ["value"]),
      ],
    },
    {
      id: "periods",
      label: lang === "es" ? "Periodos" : "Periods",
      buttons: [
        b("second", "second", ["unit", "period"]),
        b("minute", "minute", ["unit", "period"]),
        b("hour", "hour", ["unit", "period"]),
        b("day", "day", ["unit", "period"]),
        b("week", "week", ["unit", "period", "kw_week"]),
        b("month", "month", ["unit", "period"]),
        b("quarter", "quarter", ["unit", "period"]),
        b("half", "half", ["unit", "period"]),
        b("year", "year", ["unit", "period"]),
        { id: "q1", label: "Q1", insert: "Q1", kind: "word", roles: ["value", "period"] },
        { id: "q2", label: "Q2", insert: "Q2", kind: "word", roles: ["value", "period"] },
        { id: "q3", label: "Q3", insert: "Q3", kind: "word", roles: ["value", "period"] },
        { id: "q4", label: "Q4", insert: "Q4", kind: "word", roles: ["value", "period"] },
        { id: "h1", label: "H1", insert: "H1", kind: "word", roles: ["value", "period"] },
        { id: "h2", label: "H2", insert: "H2", kind: "word", roles: ["value", "period"] },
        b("mon", "Mon", ["value", "period", "weekday"], "monday"),
        b("tue", "Tue", ["value", "period", "weekday"], "tuesday"),
        b("wed", "Wed", ["value", "period", "weekday"], "wednesday"),
        b("thu", "Thu", ["value", "period", "weekday"], "thursday"),
        b("fri", "Fri", ["value", "period", "weekday"], "friday"),
        b("sat", "Sat", ["value", "period", "weekday"], "saturday"),
        b("sun", "Sun", ["value", "period", "weekday"], "sunday"),
        b("jan", "Jan", ["value", "period", "month"], "january"),
        b("feb", "Feb", ["value", "period", "month"], "february"),
        b("mar", "Mar", ["value", "period", "month"], "march"),
        b("apr", "Apr", ["value", "period", "month"], "april"),
        b("may", "May", ["value", "period", "month"], "may"),
        b("jun", "Jun", ["value", "period", "month"], "june"),
        b("jul", "Jul", ["value", "period", "month"], "july"),
        b("aug", "Aug", ["value", "period", "month"], "august"),
        b("sep", "Sep", ["value", "period", "month"], "september"),
        b("oct", "Oct", ["value", "period", "month"], "october"),
        b("nov", "Nov", ["value", "period", "month"], "november"),
        b("dec", "Dec", ["value", "period", "month"], "december"),
        b("ord1", "1st", ["value", "period"], "1st"),
        b("ord2", "2nd", ["value", "period"], "2nd"),
        b("ord3", "3rd", ["value", "period"], "3rd"),
        b("ord4", "4th", ["value", "period"], "4th"),
        b("ord5", "5th", ["value", "period"], "5th"),
        b("of", "of", ["kw_of", "after"]),
      ],
    },
    {
      id: "zones",
      label: lang === "es" ? "Zonas" : "Zones",
      buttons: [
        { id: "z_utc", label: "UTC", insert: "UTC", kind: "word", roles: ["timezone", "after"] },
        { id: "z_cdmx", label: "CDMX", insert: "CDMX", kind: "word", roles: ["timezone"] },
        { id: "z_nyc", label: "NYC", insert: "NYC", kind: "word", roles: ["timezone"] },
        { id: "z_tokyo", label: "Tokyo", insert: "Tokyo", kind: "word", roles: ["timezone"] },
        { id: "z_london", label: "London", insert: "London", kind: "word", roles: ["timezone"] },
        { id: "z_m6", label: "-06:00", insert: "-06:00", kind: "word", roles: ["timezone", "value"] },
        { id: "z_p9", label: "+09:00", insert: "+09:00", kind: "word", roles: ["timezone", "value"] },
      ],
    },
    {
      id: "bizdays",
      label: lang === "es" ? "Días hábiles" : "BizDays",
      buttons: [
        b("bizday", "business day", ["unit", "period"]),
        b("bizdays", "business days", ["unit", "duration", "after_digit"]),
        b("workweek", "work week", ["unit", "period"]),
        { id: "onorafter", label: "on or after", insert: "on or after", kind: "word", roles: ["value", "after"] },
        { id: "onorbefore", label: "on or before", insert: "on or before", kind: "word", roles: ["value", "after"] },
        b("holidays", "holidays", ["value"]),
      ],
      templates: [
        b("tpl_next_biz_after", "next business day on or after", ["value"]),
        b("tpl_last_biz_before", "last business day on or before", ["value"]),
        b("tpl_to_biz", ", to business days", ["after"], ", to business days", "punct"),
        b("tpl_next_biz", ", next business day", ["after"], ", next business day", "punct"),
        { id: "tpl_holidays_yr", label: holidaysChip, insert: holidaysChip, kind: "word", roles: ["value"] },
      ],
    },
    {
      id: "recur",
      label: lang === "es" ? "Recur / Cron" : "Recur / Cron",
      buttons: [
        b("every", "every", ["value"]),
        b("weekdaykw", "weekday", ["period", "weekday"]),
        { id: "dayyear", label: "day-year", insert: "day-year", kind: "word", roles: ["period"] },
        { id: "cron_str", label: 'cron "…"', insert: 'cron ""', kind: "cron", ariaLabel: 'cron quoted expression', roles: ["value"] },
        { id: "next_5", label: lang === "es" ? "próximo 5" : "next 5", insert: lang === "es" ? "próximo 5" : "next 5", kind: "word", roles: ["after"] },
        b("explain", "explain", ["after"]),
        b("to_cron", "to cron", ["after"]),
        b("to_ics", "to ics", ["after"]),
        b("to_grid", "to grid", ["after"]),
      ],
      templates: [
        b("tpl_every_mon_9", "every monday 9am", ["value"]),
        b("tpl_every_weekday", "every weekday", ["value"]),
        b("tpl_every_2nd_mon", "every 2nd monday", ["value"]),
        { id: "tpl_cron_mw", label: 'cron "0 9 * * 1,3"', insert: 'cron "0 9 * * 1,3"', kind: "word", roles: ["value"] },
      ],
    },
  ];
}

/** Converts a UTF-8 byte offset (from expr.Error.Pos) into a JS UTF-16 index. */
export function utf8ByteOffsetToUtf16Index(str: string, byteOffset: number): number {
  if (byteOffset <= 0) return 0;
  const encoder = new TextEncoder();
  let bytesSeen = 0;
  let utf16Index = 0;
  for (const cp of str) {
    if (bytesSeen >= byteOffset) break;
    bytesSeen += encoder.encode(cp).length;
    utf16Index += cp.length;
  }
  return Math.min(utf16Index, str.length);
}

/** Inserts an expression token at [selStart, selEnd] with smart engine spacing. */
export function insertIntoExpression(
  expr: string,
  selStart: number,
  selEnd: number,
  rawInsert: string,
  kind: InsertKind
): { text: string; caret: number } {
  const before = expr.slice(0, selStart);
  const after = expr.slice(selEnd);

  if (kind === "cron") {
    const prefix = before.length > 0 && !/[\s"]$/.test(before) ? " " : "";
    const text = before + prefix + 'cron ""' + (after.length > 0 && !/^\s/.test(after) ? " " : "") + after;
    const caret = before.length + prefix.length + 6;
    return { text, caret };
  }

  let prefix = "";
  const prevChar = before.slice(-1);
  if (before.length > 0 && !/\s$/.test(before)) {
    if (kind === "unit") {
      // Join directly after numbers/decimals (`2` + `d` => `2d`, `1.5` + `h` => `1.5h`)
      if (!/[0-9.]$/.test(before)) {
        prefix = " ";
      }
    } else if (kind === "digit") {
      // Join digits with digits, decimals, colons, slashes, or offset/leading signs (start `-2d`, `@ -06:00`, `to +09:00`)
      const insideNumericOrTime =
        /[0-9.:/]$/.test(before) ||
        /^\s*[-+]$/.test(before) ||
        /(?:@|\bto|\bin|\ba|\ben|,)\s+[-+]$/i.test(before);
      if (!insideNumericOrTime) {
        prefix = " ";
      }
    } else if (kind === "punct") {
      // `.` after digit joins; `:` or `/` after digit joins; `,` never gets leading space
      if (rawInsert === '"') {
        if (!/[\s"]$/.test(before)) {
          prefix = " ";
        }
      } else if (rawInsert.startsWith(",")) {
        prefix = "";
      } else if ((rawInsert === ":" || rawInsert === "/" || rawInsert === ".") && /[0-9]$/.test(before)) {
        prefix = "";
      } else if (prevChar !== '"' && !/[0-9]$/.test(before)) {
        prefix = " ";
      }
    } else {
      // word or op: separate from previous token unless right after opening quote
      if (prevChar !== '"') {
        prefix = " ";
      }
    }
  }

  let suffix = "";
  const nextChar = after[0] ?? "";
  if (after.length > 0 && !/^\s/.test(after)) {
    if (kind === "word" || kind === "op" || rawInsert.startsWith(",")) {
      suffix = " ";
    }
  }

  // Normalize comma templates: strip extra space before leading comma if `before` had spaces before nothing? Keep clean.
  const combined = before + prefix + rawInsert + suffix + after;
  const caret = before.length + prefix.length + rawInsert.length;
  void nextChar;
  return { text: combined, caret };
}

/** Replaces an existing YYYY-MM-DD or HH:MM token around the caret, or inserts cleanly. */
export function replaceTokenOrInsert(
  expr: string,
  selStart: number,
  selEnd: number,
  tokenRegex: RegExp,
  value: string
): { text: string; caret: number } {
  const globalRe = new RegExp(tokenRegex.source, "g");
  let match: RegExpExecArray | null;
  while ((match = globalRe.exec(expr)) !== null) {
    const mStart = match.index;
    const mEnd = mStart + match[0].length;
    if (selStart >= mStart && selEnd <= mEnd) {
      const text = expr.slice(0, mStart) + value + expr.slice(mEnd);
      return { text, caret: mStart + value.length };
    }
  }
  return insertIntoExpression(expr, selStart, selEnd, value, "word");
}

export function applyBackspace(
  expr: string,
  selStart: number,
  selEnd: number
): { text: string; caret: number } {
  if (selStart !== selEnd) {
    return {
      text: expr.slice(0, selStart) + expr.slice(selEnd),
      caret: selStart,
    };
  }
  if (selStart === 0) {
    return { text: expr, caret: 0 };
  }
  const text = expr.slice(0, selStart - 1) + expr.slice(selStart);
  return { text, caret: selStart - 1 };
}

export function isIncompleteParseError(input: string, result: Result | null): boolean {
  if (!result?.error) return false;
  if (input.trim() === "") return false;
  const err = result.error;
  if (!err.expected || err.expected.length === 0) return false;
  const trimmedBytes = new TextEncoder().encode(input.trimEnd()).length;
  return (
    err.message.startsWith("unexpected end of input") ||
    err.pos >= trimmedBytes
  );
}

const EXPECTED_TO_ROLES: Record<string, string[]> = {
  "a value": ["value"],
  "a unit": ["unit"],
  "a duration": ["duration", "digit", "unit"],
  "a timezone": ["timezone", "timezone_sign", "timezone_digit"],
  "a unix timestamp": ["unix", "digit"],
  "a period": ["period"],
  "a weekday": ["weekday"],
  "a month": ["month"],
  "a week number": ["weeknum", "digit"],
  "a time": ["time", "digit"],
  "a quoted 5-field cron string": ["cron_quote", "cron_part"],
  last: ["kw_last"],
  next: ["kw_next"],
  this: ["kw_this"],
  week: ["kw_week"],
  of: ["kw_of"],
  to: ["kw_to"],
  and: ["kw_and"],
  between: ["kw_between"],
  from: ["kw_from"],
};

const TRAILING_OPERATOR_RE =
  /(?:[+\-*|@]|\.\.|,|\b(?:to|in|from|between|and|a|en|desde|entre|y))\s*$/i;

/** Computes enabled keypad button IDs from the current input and Result. */
export function computeEnabledButtonIds(
  input: string,
  result: Result | null,
  allButtons: KeypadButton[]
): Set<string> {
  const allowedRoles = new Set<string>();
  const trimmed = input.trim();
  const endsWithDigit = /\d$/.test(trimmed);
  const endsWithOp = TRAILING_OPERATOR_RE.test(input);

  if (trimmed === "") {
    allowedRoles.add("value");
    allowedRoles.add("start_neg");
    allowedRoles.add("digit");
  } else if (result?.error) {
    const expected = result.error.expected;
    if (expected && expected.length > 0) {
      for (const exp of expected) {
        const mapped = EXPECTED_TO_ROLES[exp];
        if (mapped) {
          for (const r of mapped) allowedRoles.add(r);
        }
      }
      if (allowedRoles.size === 0) {
        allowedRoles.add("value");
        allowedRoles.add("digit");
      }
    } else {
      // Mid-expression or semantic error (e.g. `today + 2` before clicking `d`, `every`, `2026-06-02 + 2x`)
      allowedRoles.add("digit");
      allowedRoles.add("unit");
      allowedRoles.add("period");
      allowedRoles.add("weekday");
      allowedRoles.add("month");
      allowedRoles.add("after_digit");
      if (endsWithDigit) {
        allowedRoles.add("duration");
      }
      if (!endsWithOp) {
        allowedRoles.add("after");
        allowedRoles.add("value");
      } else {
        allowedRoles.add("value");
      }
    }
  } else {
    // Valid, complete expression
    if (endsWithOp) {
      allowedRoles.add("value");
      allowedRoles.add("digit");
    } else {
      allowedRoles.add("after");
      if (endsWithDigit) {
        allowedRoles.add("digit");
        allowedRoles.add("unit");
        allowedRoles.add("after_digit");
      } else {
        // Still allow typing digits for clock times after date expressions (e.g. `today 14:30`)
        allowedRoles.add("time");
      }
    }
  }

  const enabled = new Set<string>(["space", "backspace", "clear"]);
  for (const btn of allButtons) {
    if (btn.roles.some((r) => allowedRoles.has(r))) {
      enabled.add(btn.id);
    }
  }
  return enabled;
}
