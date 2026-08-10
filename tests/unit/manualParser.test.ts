import { describe, expect, it } from "vitest";
import { hasNonEmptyManualEntries, parseManualEntries } from "../../src/parsers/manualParser";
import { LIMITS } from "../../src/constants";

describe("manual parser", () => {
    it("detects whether manual entries contain data", () => {
        expect(hasNonEmptyManualEntries([{ name: "", points: "" }])).toBe(false);
        expect(hasNonEmptyManualEntries([{ name: "Alice", points: "" }])).toBe(true);
    });

    it("parses valid rows and records the source row", () => {
        const result = parseManualEntries([
            { name: "Alice", points: "80" },
            { name: "Bob", points: "45,5" }
        ]);

        expect(result.diagnostics).toHaveLength(0);
        expect(result.students).toEqual([
            { name: "Alice", points: 80, sourceRow: 1 },
            { name: "Bob", points: 45.5, sourceRow: 2 }
        ]);
    });

    it("skips completely empty rows without a diagnostic", () => {
        const result = parseManualEntries([
            { name: "", points: "" },
            { name: "Alice", points: "50" }
        ]);

        expect(result.students).toHaveLength(1);
        expect(result.diagnostics).toHaveLength(0);
    });

    it("warns per problematic row and keeps the valid ones", () => {
        const result = parseManualEntries([
            { name: "", points: "50" },
            { name: "Alice", points: "" },
            { name: "Bob", points: "abc" },
            { name: "Carol", points: "70" }
        ]);

        expect(result.students).toHaveLength(1);
        expect(result.students[0]?.name).toBe("Carol");

        const codes = result.diagnostics.map((d) => d.code);
        expect(codes).toContain("row-missing-name");
        expect(codes).toContain("row-missing-points");
        expect(codes).toContain("row-unparsable-points");
        expect(result.diagnostics.every((d) => d.severity === "warning")).toBe(true);
    });

    it("keeps the rows read so far when the limit is reached", () => {
        const entries = Array.from({ length: LIMITS.maxStudents + 3 }, (_, i) => ({
            name: `S${i}`,
            points: "10"
        }));
        const result = parseManualEntries(entries);

        expect(result.students).toHaveLength(LIMITS.maxStudents);
        expect(result.diagnostics.filter((d) => d.code === "row-limit-reached")).toHaveLength(1);
    });

    it("sanitizes names", () => {
        const result = parseManualEntries([{ name: "<Alice>", points: "10" }]);
        expect(result.students[0]?.name).toBe("Alice");
    });
});
