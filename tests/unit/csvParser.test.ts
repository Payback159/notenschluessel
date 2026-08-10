import { describe, expect, it } from "vitest";
import { detectDelimiter, parseCSVContent } from "../../src/parsers/csvParser";
import { LIMITS } from "../../src/constants";

describe("csv parser", () => {
    it("detects comma delimiter", () => {
        expect(detectDelimiter("Name,Punkte\nAlice,12")).toBe(",");
    });

    it("detects semicolon delimiter", () => {
        expect(detectDelimiter("Name;Punkte\nAlice;12")).toBe(";");
    });

    it("detects the delimiter from a single line without a header", () => {
        expect(detectDelimiter("Alice,12")).toBe(",");
        expect(detectDelimiter("Alice;12")).toBe(";");
    });

    it("is not fooled by commas inside names that outnumber the semicolon delimiters", () => {
        // Reproduces the review finding: a semicolon-delimited export with
        // decimal-comma points and names containing two commas each used to tip
        // the old raw-character-count heuristic towards comma after ~3 rows.
        const lines = ["Name;Punkte;Note"];
        for (let i = 0; i < 20; i++) {
            lines.push(`Nachname${i}, Vorname${i}, Jr.;76,5;2`);
        }
        expect(detectDelimiter(lines.join("\n"))).toBe(";");
    });

    it("is not fooled by comma-laden names when the header row is missing", () => {
        // Without the header the comma splits every row into a consistent four
        // fields while the correct semicolon yields three, so picking by field
        // count alone would choose the comma and drop all rows.
        const lines: string[] = [];
        for (let i = 0; i < 5; i++) {
            lines.push(`Nachname${i}, Vorname${i}, Jr.;76,5;2`);
        }
        const result = parseCSVContent(lines.join("\n"));
        expect(result.diagnostics).toHaveLength(0);
        expect(result.students).toHaveLength(5);
        expect(result.students[0]?.name).toBe("Nachname0, Vorname0, Jr.");
        expect(result.students[0]?.points).toBe(76.5);
    });

    it("does not split a header-less semicolon file on its decimal commas", () => {
        // "Müller;76,5" also splits into a usable ["Müller;76", "5"] under the
        // comma, so this used to import silently wrong data: no diagnostic, but
        // the name carried the points and the points were the decimals.
        const result = parseCSVContent("Müller;76,5\nÖz;88,5\nBauer;91,5");
        expect(result.diagnostics).toHaveLength(0);
        expect(result.students).toHaveLength(3);
        expect(result.students[0]).toEqual({ name: "Müller", points: 76.5, sourceRow: 1 });
    });

    it("still reads a header-less comma file as comma-delimited", () => {
        const result = parseCSVContent("Alice,95\nBob,42.5");
        expect(result.diagnostics).toHaveLength(0);
        expect(result.students).toHaveLength(2);
        expect(result.students[1]).toEqual({ name: "Bob", points: 42.5, sourceRow: 2 });
    });

    it("parses valid csv and records the source row", () => {
        const result = parseCSVContent("Name,Punkte\nAlice,95\nBob,42.5");
        expect(result.diagnostics).toHaveLength(0);
        expect(result.students).toHaveLength(2);
        expect(result.students[0]).toEqual({ name: "Alice", points: 95, sourceRow: 2 });
        expect(result.students[1]?.sourceRow).toBe(3);
    });

    it("warns about every skipped row instead of staying silent", () => {
        const result = parseCSVContent("Name,Punkte\nAlice,95\nKaputt\nBob,abc\nCharlie,20");
        expect(result.students).toHaveLength(2);

        const codes = result.diagnostics.map((d) => d.code);
        expect(codes).toContain("row-too-few-columns");
        expect(codes).toContain("row-unparsable-points");
        expect(result.diagnostics.every((d) => d.severity === "warning")).toBe(true);

        const rows = result.diagnostics.map((d) => d.row);
        expect(rows).toContain(3);
        expect(rows).toContain(4);
    });

    it("warns about a missing name", () => {
        const result = parseCSVContent("Name,Punkte\n,50");
        expect(result.diagnostics.some((d) => d.code === "row-missing-name" && d.row === 2)).toBe(true);
    });

    it("warns about negative points", () => {
        const result = parseCSVContent("Name,Punkte\nAlice,-5\nBob,10");
        expect(result.diagnostics.some((d) => d.code === "row-negative-points" && d.row === 2)).toBe(true);
        expect(result.students).toHaveLength(1);
    });

    it("returns an error diagnostic if no valid rows exist", () => {
        const result = parseCSVContent("Name,Punkte\nAlice,abc");
        expect(result.students).toHaveLength(0);
        expect(result.diagnostics.some((d) => d.code === "no-valid-students")).toBe(true);
    });

    it("returns an error diagnostic for empty input", () => {
        const result = parseCSVContent("   ");
        expect(result.students).toHaveLength(0);
        expect(result.diagnostics.some((d) => d.code === "no-valid-students")).toBe(true);
    });

    it("warns once when the row limit is reached, keeping the rows read so far", () => {
        const rows = Array.from({ length: LIMITS.maxStudents + 5 }, (_, i) => `S${i},10`).join("\n");
        const result = parseCSVContent(`Name,Punkte\n${rows}`);

        expect(result.students).toHaveLength(LIMITS.maxStudents);
        expect(result.diagnostics.filter((d) => d.code === "row-limit-reached")).toHaveLength(1);
        expect(result.diagnostics.some((d) => d.code === "no-valid-students")).toBe(false);
    });

    it("sanitizes names", () => {
        const result = parseCSVContent("Name,Punkte\n<bad>,10");
        expect(result.students[0]?.name).toBe("bad");
    });

    it("ignores a leading BOM when detecting the header", () => {
        const result = parseCSVContent("\uFEFFName,Punkte\nAlice,95");
        expect(result.students).toHaveLength(1);
        expect(result.students[0]?.name).toBe("Alice");
    });
});
