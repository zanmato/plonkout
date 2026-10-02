/**
 * Reads a number typed by a person, with a decimal comma or point, e.g. "1,5".
 * Undefined when the text is empty or not a number.
 */
export function parseDecimal(text: string | number | null | undefined): number | undefined {
  if (typeof text === "number") return Number.isFinite(text) ? text : undefined;
  const trimmed = (text ?? "").trim().replace(",", ".");
  if (trimmed === "" || !/^-?\d*\.?\d+$|^-?\d+\.$/.test(trimmed)) return undefined;
  const value = Number(trimmed);
  return Number.isFinite(value) ? value : undefined;
}

/** A nutrient amount for a weight, rounded to a tenth like the server. */
export function scale(per100g: number, grams: number): number {
  return Math.round((per100g * grams) / 10) / 10;
}
