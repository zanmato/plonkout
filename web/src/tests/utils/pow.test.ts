import { describe, expect, it } from "vitest";
import { sha256Hex, solvePow } from "@/utils/pow";

describe("solvePow", () => {
  it("finds the number the server hid", async () => {
    const salt = "abc?expires=1";
    const challenge = await sha256Hex(`${salt}37`);

    const solution = await solvePow({ algorithm: "SHA-256", challenge, maxNumber: 100, salt, signature: "sig" });

    expect(solution).toEqual({ challenge, number: 37, salt, signature: "sig" });
  });

  it("gives up past the maximum", async () => {
    const salt = "abc?expires=1";
    const challenge = await sha256Hex(`${salt}500`);

    await expect(
      solvePow({ algorithm: "SHA-256", challenge, maxNumber: 100, salt, signature: "sig" }),
    ).rejects.toThrow();
  });
});
