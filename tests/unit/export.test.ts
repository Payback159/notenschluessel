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

    it("exports grade scale csv", () => {
        const csv = exportGradeScaleCSV(bounds);
        expect(csv).toContain("Note,Untergrenze,Obergrenze");
    });

    it("sanitizes csv fields for student export", () => {
        const csv = exportStudentResultsCSV(students);
        expect(csv).toContain("'=cmd");
    });

    it("exports combined csv", () => {
        const csv = exportCombinedCSV(bounds, students, { maxPoints: 45, minPoints: 0.5, breakPointPercent: 50 });
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