import { quantityDisplay } from "../report/rowPresentation.js";

const COLUMN_INDEX = new Map();

const MAX_FORMULA_RANGE_CELLS = 4000;

export function formulaOverlays(formulas = [], { overlays = new Map(), values = {} } = {}) {
  const items = [];
  const formulaCells = new Set();
  const list = Array.isArray(formulas) ? formulas : [];
  for (const item of list) {
    const row = Number(item?.row);
    const column = Number(item?.column);
    const formula = String(item?.formula || item?.text || "").replace(/^\s*=/, "").trim();
    if (!Number.isFinite(row) || row < 1 || !Number.isFinite(column) || column < 1 || !formula) continue;
    items.push({ row, column, formula });
    formulaCells.add(`${row}:${column}`);
  }

  const computed = new Map();
  const subtotalKeys = new Set(
    items.filter((item) => /^SUBTOTAL\s*\(/i.test(item.formula)).map((item) => `${item.row}:${item.column}`),
  );

  function getNumber(row, column) {
    const key = `${row}:${column}`;
    const overlay = overlays.get(key);
    if (overlay?.field === "value") {
      const number = toNumber(overlay.value);
      if (number != null) return number;
    }
    if (computed.has(key)) {
      const number = toNumber(computed.get(key).value);
      if (number != null) return number;
    }
    if (formulaCells.has(key) && !computed.has(key)) {
      throw new Error("pending");
    }
    const number = toNumber(values[key]);
    return number == null ? 0 : number;
  }

  let pending = items;
  for (let pass = 0; pass < items.length + 2 && pending.length; pass += 1) {
    const next = [];
    for (const item of pending) {
      const key = `${item.row}:${item.column}`;
      if (overlays.get(key)?.field === "value") continue;
      let result;
      try {
        result = evaluateFormula(item.formula, getNumber, subtotalKeys);
      } catch (error) {
        if (error?.message === "pending") {
          next.push(item);
          continue;
        }
        continue;
      }
      if (result == null || !Number.isFinite(result)) continue;
      computed.set(key, { field: "formula", value: quantityDisplay(result) });
    }
    if (next.length === pending.length) break;
    pending = next;
  }
  return computed;
}

export function evaluateFormula(formula, getNumber, subtotalKeys = new Set()) {
  const tokens = tokenize(String(formula || "").replace(/^\s*=/, ""));
  const parser = { tokens, index: 0, getNumber, subtotalKeys };
  const value = parseExpr(parser);
  if (parser.index < parser.tokens.length) return null;
  return value;
}

function tokenize(source) {
  const tokens = [];
  const text = String(source || "").replace(/\s+/g, "").toUpperCase();
  let index = 0;
  while (index < text.length) {
    const char = text[index];
    if (char >= "0" && char <= "9" || char === ".") {
      let end = index + 1;
      while (end < text.length && ((text[end] >= "0" && text[end] <= "9") || text[end] === ".")) end += 1;
      const value = Number(text.slice(index, end));
      if (!Number.isFinite(value)) throw new Error("number");
      tokens.push({ type: "num", value });
      index = end;
      continue;
    }
    if (char >= "A" && char <= "Z" || char === "$") {
      let end = index + 1;
      while (end < text.length && /[A-Z$0-9:]/.test(text[end])) end += 1;
      const value = text.slice(index, end);
      if (/^[A-Z]+$/.test(value) && text[end] === "(") {
        tokens.push({ type: "fn", value });
      } else {
        tokens.push({ type: "ref", value });
      }
      index = end;
      continue;
    }
    if ("+-*/(),".includes(char)) {
      tokens.push({ type: char });
      index += 1;
      continue;
    }
    throw new Error("token");
  }
  return tokens;
}

function parseExpr(parser) {
  let value = parseTerm(parser);
  while (match(parser, "+") || match(parser, "-")) {
    const op = parser.tokens[parser.index - 1].type;
    const right = parseTerm(parser);
    value = op === "+" ? value + right : value - right;
  }
  return value;
}

function parseTerm(parser) {
  let value = parseFactor(parser);
  while (match(parser, "*") || match(parser, "/")) {
    const op = parser.tokens[parser.index - 1].type;
    const right = parseFactor(parser);
    value = op === "*" ? value * right : right === 0 ? NaN : value / right;
  }
  return value;
}

function parseFactor(parser) {
  if (match(parser, "+")) return parseFactor(parser);
  if (match(parser, "-")) return -parseFactor(parser);
  const token = peek(parser);
  if (!token) throw new Error("eof");
  if (token.type === "num") {
    parser.index += 1;
    return token.value;
  }
  if (token.type === "ref") {
    parser.index += 1;
    return sumRefs(token.value, parser.getNumber);
  }
  if (token.type === "fn") {
    parser.index += 1;
    expect(parser, "(");
    if (token.value === "SUM") {
      return parseSumArgs(parser, false);
    }
    if (token.value === "SUBTOTAL") {
      const code = parseExpr(parser);
      expect(parser, ",");
      if (code !== 9 && code !== 109) throw new Error("subtotal");
      return parseSumArgs(parser, true);
    }
    if (token.value === "ROUND") {
      const value = parseExpr(parser);
      expect(parser, ",");
      const digits = parseExpr(parser);
      expect(parser, ")");
      const factor = 10 ** Math.max(0, Math.min(10, Math.trunc(digits)));
      return Math.round(value * factor) / factor;
    }
    throw new Error("fn");
  }
  if (match(parser, "(")) {
    const value = parseExpr(parser);
    expect(parser, ")");
    return value;
  }
  throw new Error("factor");
}

function parseSumArgs(parser, skipSubtotals) {
  let total = 0;
  total += parseSumArg(parser, skipSubtotals);
  while (match(parser, ",")) {
    total += parseSumArg(parser, skipSubtotals);
  }
  expect(parser, ")");
  return total;
}

function parseSumArg(parser, skipSubtotals) {
  const token = peek(parser);
  if (token?.type === "ref") {
    parser.index += 1;
    return sumRefs(token.value, parser.getNumber, skipSubtotals ? parser.subtotalKeys : null);
  }
  return parseExpr(parser);
}

function sumRefs(text, getNumber, skipKeys) {
  const parts = String(text || "").split(":");
  if (parts.length === 1) {
    const cell = parseA1(parts[0]);
    if (!cell) throw new Error("ref");
    if (skipKeys?.has(`${cell.row}:${cell.column}`)) return 0;
    return getNumber(cell.row, cell.column);
  }
  const start = parseA1(parts[0]);
  const end = parseA1(parts[1]);
  if (!start || !end) throw new Error("range");
  const rowFrom = Math.min(start.row, end.row);
  const rowTo = Math.max(start.row, end.row);
  const colFrom = Math.min(start.column, end.column);
  const colTo = Math.max(start.column, end.column);
  const count = (rowTo - rowFrom + 1) * (colTo - colFrom + 1);
  if (count > MAX_FORMULA_RANGE_CELLS) throw new Error("range");
  let total = 0;
  for (let row = rowFrom; row <= rowTo; row += 1) {
    for (let column = colFrom; column <= colTo; column += 1) {
      if (skipKeys?.has(`${row}:${column}`)) continue;
      total += getNumber(row, column);
    }
  }
  return total;
}

function parseA1(text) {
  const match = /^\$?([A-Z]+)\$?([1-9][0-9]*)$/i.exec(String(text || "").trim());
  if (!match) return null;
  const column = columnIndex(match[1]);
  const row = Number(match[2]);
  if (!column || !Number.isFinite(row)) return null;
  return { row, column };
}

function columnIndex(letters) {
  const key = String(letters || "").toUpperCase();
  if (COLUMN_INDEX.has(key)) return COLUMN_INDEX.get(key);
  let column = 0;
  for (const char of key) {
    if (char < "A" || char > "Z") return 0;
    column = column * 26 + (char.charCodeAt(0) - 64);
  }
  COLUMN_INDEX.set(key, column);
  return column;
}

function toNumber(value) {
  if (value == null || value === "") return null;
  const number = Number(String(value).replace(",", "."));
  return Number.isFinite(number) ? number : null;
}

function peek(parser) {
  return parser.tokens[parser.index];
}

function match(parser, type) {
  if (peek(parser)?.type !== type) return false;
  parser.index += 1;
  return true;
}

function expect(parser, type) {
  if (!match(parser, type)) throw new Error(type);
}
