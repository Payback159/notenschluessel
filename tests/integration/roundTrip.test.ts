import { describe, expect, it } from "vitest";
import { exportStudentResultsCSV } from "../../src/export/csvExport";
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
        const csv = `\uFEFF${exportStudentResultsCSV(students, 0.5)}`;
        const parsed = parseCSVContent(csv);

        expect(parsed.diagnostics).toHaveLength(0);
        expect(parsed.students.map((s) => s.name)).toEqual(["Müller", "Öz"]);
    });
});
