// Small fixed palette, deterministic hash so the same service always gets
// the same color across renders and re-fetches. Not tied to design tokens
// since it needs N distinct hues, not one semantic color.
const PALETTE = [
  "#6ba3d6",
  "#d97757",
  "#7fb069",
  "#b98cce",
  "#e0b03e",
  "#5fc9c0",
  "#d97ba0",
  "#8a97a8",
];

export function colorForService(serviceName: string): string {
  let hash = 0;
  for (let i = 0; i < serviceName.length; i++) {
    hash = (hash * 31 + serviceName.charCodeAt(i)) | 0;
  }
  return PALETTE[Math.abs(hash) % PALETTE.length];
}
