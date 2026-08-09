import { describe, expect, it } from "vitest";
import { exportStudentResultsCSV, withBom } from "../../src/export/csvExport";
import { parseCSVContent } from "../../src/parsers/csvParser";

describe("csv round trip", () => {
    const students = [
        { name: "Müller", points: 76.5, grade: 2 },
        { name: "Öz", points: 100, grade: 1 }
    ];

    it("reads its own export back unchanged", () => {
        const csv = exportStudentResultsCSV(students, 0.5);
        const parsed = parseCSVContent(csv);

        expect(parsed.diagnostics).toHaveLength(0);
        expect(parsed.students.map((s) => s.name)).toEqual(["Müller", "Öz"]);
        expect(parsed.students.map((s) => s.points)).toEqual([76.5, 100]);
    });

    it("reads its own export back even with a leading BOM", () => {
        const csv = withBom(exportStudentResultsCSV(students, 0.5));
        const parsed = parseCSVContent(csv);

        expect(parsed.diagnostics).toHaveLength(0);
        expect(parsed.students.map((s) => s.name)).toEqual(["Müller", "Öz"]);
    });

    it("withBom prepends exactly one U+FEFF", () => {
        const content = "Name;Punkte;Note";
        const withMark = withBom(content);

        expect(withMark).toBe(`\uFEFF${content}`);
        expect(withMark.codePointAt(0)).toBe(0xfeff);
        expect(withMark.slice(1)).toBe(content);
    });

    it("round-trips 20 students whose names each contain two commas", () => {
        // The review finding: names like "Nachname, Vorname, Jr." push the raw
        // comma count above the semicolon count after only a few rows, so the old
        // statistical detectDelimiter misread the app's own semicolon export as
        // comma-delimited and produced zero students.
        const manyStudents = Array.from({ length: 20 }, (_, i) => ({
            name: `Nachname${i}, Vorname${i}, Jr.`,
            points: 76.5,
            grade: 2
        }));

        const csv = exportStudentResultsCSV(manyStudents, 0.5);
        const parsed = parseCSVContent(csv);

        expect(parsed.diagnostics).toHaveLength(0);
        expect(parsed.students).toHaveLength(20);
        expect(parsed.students.map((s) => s.name)).toEqual(manyStudents.map((s) => s.name));
        expect(parsed.students.every((s) => s.points === 76.5)).toBe(true);
    });
});
