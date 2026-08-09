import { describe, expect, it } from "vitest";
import {
    exportCombinedCSV,
    exportGradeScaleCSV,
    exportStudentResultsCSV
} from "../../src/export/csvExport";
import {
    buildCombinedWorkbook,
    buildGradeScaleWorkbook,
    buildStudentsWorkbook,
    workbookToUint8Array
} from "../../src/export/excelExport";

describe("export modules", () => {
    const bounds = [
        { grade: 1, lowerBound: 39.5, upperBound: 45 },
        { grade: 2, lowerBound: 34, upperBound: 39 },
        { grade: 3, lowerBound: 28, upperBound: 33.5 },
        { grade: 4, lowerBound: 22.5, upperBound: 27.5 },
        { grade: 5, lowerBound: 0, upperBound: 22 }
    ];
    const students = [
        { name: "=cmd", points: 45, grade: 1 },
        { name: "Alice", points: 22.5, grade: 4 }
    ];

    it("keeps a dangerous name in a single field even when it contains the delimiter", () => {
        // The old code returned early after prefixing, so the delimiter split the row.
        const csv = exportStudentResultsCSV([{ name: "=Mueller; Hans", points: 90, grade: 1 }], 0.5);
        const dataLine = csv.split("\n")[1];
        expect(dataLine).toBeDefined();
        expect(dataLine).toBe(`"'=Mueller; Hans";90;1`);
    });

    it("still prefixes formula characters", () => {
        // Quoted, not bare: dropping the `guarded !== field` re-check would leave
        // this field unquoted even though it was prefixed.
        const csv = exportStudentResultsCSV([{ name: "=cmd", points: 45, grade: 1 }], 0.5);
        const dataLine = csv.split("\n")[1];
        expect(dataLine).toBeDefined();
        expect(dataLine).toBe(`"'=cmd";45;1`);
    });

    it("doubles embedded quotes when escaping a field", () => {
        const csv = exportStudentResultsCSV([{ name: 'Bob "Bobby" Smith', points: 30, grade: 3 }], 0.5);
        const dataLine = csv.split("\n")[1];
        expect(dataLine).toBeDefined();
        expect(dataLine).toBe(`"Bob ""Bobby"" Smith";30;3`);
    });

    it("uses semicolons and a decimal comma", () => {
        const csv = exportStudentResultsCSV([{ name: "Alice", points: 76.5, grade: 2 }], 0.5);
        expect(csv.split("\n")[0]).toBe("Name;Punkte;Note");
        expect(csv).toContain("Alice;76,5;2");
    });

    it("exports the grade scale with semicolons", () => {
        const csv = exportGradeScaleCSV(bounds, 0.5);
        expect(csv.split("\n")[0]).toBe("Note;Untergrenze;Obergrenze");
        expect(csv).toContain("1;39,5;45");
    });

    it("exports combined csv", () => {
        const csv = exportCombinedCSV(bounds, students, {
            maxPoints: 45,
            minPoints: 0.5,
            breakPointPercent: 50
        });
        expect(csv).toContain("## Notenskala");
        expect(csv).toContain("## Schüler");
    });

    it("builds non-empty excel workbooks", () => {
        const a = workbookToUint8Array(buildGradeScaleWorkbook(bounds));
        const b = workbookToUint8Array(buildStudentsWorkbook(students));
        const c = workbookToUint8Array(buildCombinedWorkbook(bounds, students, {
            maxPoints: 45,
            minPoints: 0.5,
            breakPointPercent: 50
        }));

        expect(a.length).toBeGreaterThan(100);
        expect(b.length).toBeGreaterThan(100);
        expect(c.length).toBeGreaterThan(100);
    });

    it("applies grade colors to excel cells", () => {
        const wb = buildGradeScaleWorkbook(bounds);
        const sheet = wb.Sheets["Notenskala"]!;
        // Note 1 sits in row 2 (header in row 1); the whole row must be colored
        for (const col of ["A", "B", "C"]) {
            const cell = sheet[`${col}2`] as { s?: { fill?: { fgColor?: { rgb?: string } } } };
            expect(cell.s?.fill?.fgColor?.rgb).toBe("D4EDDA");
        }
    });
});