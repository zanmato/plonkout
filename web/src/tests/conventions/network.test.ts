import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative } from "node:path";
import { describe, expect, it } from "vitest";

const srcRoot = join(__dirname, "..", "..");

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) return sourceFiles(path);
    return /\.(ts|vue)$/.test(name) ? [path] : [];
  });
}

describe("network access", () => {
  it("only happens through src/api", () => {
    const offenders = sourceFiles(srcRoot)
      .map((path) => relative(srcRoot, path))
      .filter((path) => !path.startsWith("api/") && !path.startsWith("tests/"))
      .filter((path) => {
        // Type only imports of the generated API types carry no runtime code
        const source = readFileSync(join(srcRoot, path), "utf8").replace(/^import type [^;]*;/gm, "");
        return /\bfetch\(|XMLHttpRequest|["'`]\/api\/|from ["']@\/api\/gen/.test(source);
      });

    expect(offenders).toEqual([]);
  });
});
