import { describe, expect, it } from "vitest";
import { sanitizeName } from "../../src/core/sanitize";
import { LIMITS } from "../../src/constants";

describe("sanitizeName", () => {
    it("removes angle brackets", () => {
        expect(sanitizeName("<Alice>")).toBe("Alice");
    });

    it("collapses line breaks and tabs into spaces", () => {
        expect(sanitizeName("A\nB\tC")).toBe("A B C");
    });

    it("trims surrounding whitespace", () => {
        expect(sanitizeName("  Alice  ")).toBe("Alice");
    });

    it("keeps umlauts intact", () => {
        expect(sanitizeName("Müller-Öztürk")).toBe("Müller-Öztürk");
    });

    it("truncates overly long names", () => {
        const long = "x".repeat(LIMITS.maxNameLength + 50);
        expect(sanitizeName(long)).toHaveLength(LIMITS.maxNameLength);
    });
});
