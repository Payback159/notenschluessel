import { describe, expect, it } from "vitest";
import { hasNonEmptyManualEntries, parseManualEntries } from "../../src/parsers/manualParser";

describe("manual parser", () => {
    it("detects whether manual entries contain data", () => {
        expect(hasNonEmptyManualEntries([{ name: "", points: "" }])).toBe(false);
        expect(hasNonEmptyManualEntries([{ name: "Alice", points: "" }])).toBe(true);
    });

    it("parses valid rows", () => {
        const result = parseManualEntries([
            { name: "Alice", points: "80" },
            { name: "Bob", points: "45,5" }
        ]);

        expect(result.errors).toHaveLength(0);
        expect(result.students).toEqual([
            { name: "Alice", points: 80 },
            { name: "Bob", points: 45.5 }
        ]);
    });

    it("skips completely empty rows", () => {
        const result = parseManualEntries([
            { name: "", points: "" },
            { name: "Alice", points: "50" }
        ]);

        expect(result.students).toHaveLength(1);
        expect(result.students[0]).toBeDefined();
        expect(result.students[0]?.name).toBe("Alice");
    });

    it("reports validation errors", () => {
        const result = parseManualEntries([
            { name: "", points: "50" },
            { name: "Alice", points: "" },
            { name: "Bob", points: "abc" }
        ]);

        expect(result.students).toHaveLength(0);
        expect(result.errors.length).toBe(3);
    });

    it("sanitizes names", () => {
        const result = parseManualEntries([{ name: "<Alice>", points: "10" }]);
        expect(result.students[0]).toBeDefined();
        expect(result.students[0]?.name).toBe("Alice");
    });
});