import { useState } from "react";

type JsonValue = string | number | boolean | null | JsonValue[] | { [key: string]: JsonValue };

function valueClass(value: JsonValue): string {
  if (typeof value === "string") return "text-text-primary";
  if (typeof value === "number") return "text-accent";
  if (typeof value === "boolean") return "text-warning";
  if (value === null) return "text-text-faint italic";
  return "text-text-muted";
}

function renderScalar(value: string | number | boolean | null): string {
  if (typeof value === "string") return `"${value}"`;
  if (value === null) return "null";
  return String(value);
}

function JsonNode({ label, value, depth }: { label: string; value: JsonValue; depth: number }) {
  const isObject = value !== null && typeof value === "object" && !Array.isArray(value);
  const isArray = Array.isArray(value);
  const isExpandable = isObject || isArray;

  const [open, setOpen] = useState(depth < 1);

  if (!isExpandable) {
    return (
      <div className="py-1 leading-relaxed" style={{ paddingLeft: depth * 14 }}>
        <span className="text-text-faint">{label}: </span>
        <span className={`break-words ${valueClass(value)}`}>{renderScalar(value)}</span>
      </div>
    );
  }

  const entries = isArray
    ? (value as JsonValue[]).map((v, i) => [String(i), v] as const)
    : Object.entries(value as Record<string, JsonValue>);
  const count = entries.length;
  const bracket = isArray ? ["[", "]"] : ["{", "}"];

  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-1.5 py-1 text-left leading-relaxed hover:bg-elevated"
        style={{ paddingLeft: depth * 14 }}
      >
        <span className="w-3 flex-shrink-0 text-text-faint">{open ? "▾" : "▸"}</span>
        <span className="text-text-faint">{label}:</span>
        <span className="text-text-faint">
          {bracket[0]}
          {!open && `${count} ${isArray ? "items" : "keys"}${bracket[1]}`}
        </span>
      </button>
      {open && (
        <>
          {entries.map(([k, v]) => (
            <JsonNode key={k} label={isArray ? "" : k} value={v} depth={depth + 1} />
          ))}
          <div className="text-text-faint leading-relaxed" style={{ paddingLeft: depth * 14 }}>
            {bracket[1]}
          </div>
        </>
      )}
    </div>
  );
}

export function JsonTree({ data }: { data: unknown }) {
  const value = data as JsonValue;
  const isObject = value !== null && typeof value === "object" && !Array.isArray(value);
  const isArray = Array.isArray(value);

  if (!isObject && !isArray) {
    return <span className={valueClass(value)}>{renderScalar(value as string | number | boolean | null)}</span>;
  }

  const entries = isArray
    ? (value as JsonValue[]).map((v, i) => [String(i), v] as const)
    : Object.entries(value as Record<string, JsonValue>);

  return (
    <div className="font-plex-mono text-[11.5px]">
      {entries.map(([k, v]) => (
        <JsonNode key={k} label={k} value={v} depth={0} />
      ))}
    </div>
  );
}

// Line-based highlighter for JSON.stringify(data, null, 2) output — same
// color rules as the tree, so Tree/Raw read as one system, not two.
const LINE_RE = /^(\s*)(?:"([^"]*)":\s*)?(.*?)(,?)$/;

function classifyToken(token: string): string {
  if (token === "") return "text-text-faint";
  if (token === "null" || token === "null,") return "text-text-faint italic";
  if (token === "true" || token === "false" || token === "true," || token === "false,")
    return "text-warning";
  if (/^-?\d+(\.\d+)?,?$/.test(token)) return "text-accent";
  if (token.startsWith('"')) return "text-text-primary";
  return "text-text-faint";
}

export function JsonRaw({ text }: { text: string }) {
  const lines = text.split("\n");
  return (
    <pre className="font-plex-mono text-[11.5px] leading-relaxed whitespace-pre-wrap break-words">
      {lines.map((line, i) => {
        const m = line.match(LINE_RE);
        if (!m) return <div key={i}>{line}</div>;
        const [, indent, key, rest, comma] = m;
        return (
          <div key={i}>
            {indent}
            {key !== undefined && <span className="text-text-faint">"{key}": </span>}
            <span className={classifyToken(rest)}>{rest}</span>
            {comma}
          </div>
        );
      })}
    </pre>
  );
}
