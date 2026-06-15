import { describe, expect, it } from "vitest";
import { detectDelimiter, parseCSVContent } from "../../src/parsers/csvParser";

describe("csv parser", () => {
    it("detects comma delimiter", () => {
        expect(detectDelimiter("Name,Punkte\nAlice,12")).toBe(",");
    });

    it("detects semicolon delimiter", () => {
        expect(detectDelimiter("Name;Punkte\nAlice;12")).toBe(";");
    });

    it("parses valid csv", () => {
        const result = parseCSVContent("Name,Punkte\nAlice,95\nBob,42.5");
        expect(result.errors).toHaveLength(0);
        expect(result.students).toHaveLength(2);
        expect(result.students[0]).toEqual({ name: "Alice", points: 95 });
    });

    it("skips invalid rows and returns valid ones", () => {
        const result = parseCSVContent("Name,Punkte\nAlice,95\nBob,abc\nCharlie,20");
        expect(result.students).toHaveLength(2);
        expect(result.skippedRows).toBeGreaterThan(0);
    });

    it("returns error if no valid rows exist", () => {
        const result = parseCSVContent("Name,Punkte\nAlice,abc");
        expect(result.students).toHaveLength(0);
        expect(result.errors[0]).toContain("Keine gültigen");
    });

    it("sanitizes names", () => {
        const result = parseCSVContent("Name,Punkte\n<bad>,10");
        expect(result.students[0]).toBeDefined();
        expect(result.students[0]?.name).toBe("bad");
    });
});