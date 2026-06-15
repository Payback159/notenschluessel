import * as XLSX from "xlsx";
import { GradeBound, Student } from "../types";

export function buildGradeScaleWorkbook(bounds: GradeBound[]): XLSX.WorkBook {
    const workbook = XLSX.utils.book_new();
    const data = bounds.map((b) => ({
        Note: b.grade,
        Untergrenze: b.lowerBound,
        Obergrenze: b.upperBound
    }));
    const sheet = XLSX.utils.json_to_sheet(data);
    XLSX.utils.book_append_sheet(workbook, sheet, "Notenskala");
    return workbook;
}

export function buildStudentsWorkbook(students: Student[]): XLSX.WorkBook {
    const workbook = XLSX.utils.book_new();
    const data = students.map((s) => ({
        Name: s.name,
        Punkte: s.points,
        Note: s.grade ?? ""
    }));
    const sheet = XLSX.utils.json_to_sheet(data);
    XLSX.utils.book_append_sheet(workbook, sheet, "Schüler");
    return workbook;
}

export function buildCombinedWorkbook(
    bounds: GradeBound[],
    students: Student[],
    metadata: { maxPoints: number; minPoints: number; breakPointPercent: number }
): XLSX.WorkBook {
    const workbook = XLSX.utils.book_new();

    XLSX.utils.book_append_sheet(
        workbook,
        XLSX.utils.json_to_sheet(
            bounds.map((b) => ({ Note: b.grade, Untergrenze: b.lowerBound, Obergrenze: b.upperBound }))
        ),
        "Notenskala"
    );

    XLSX.utils.book_append_sheet(
        workbook,
        XLSX.utils.json_to_sheet(students.map((s) => ({ Name: s.name, Punkte: s.points, Note: s.grade ?? "" }))),
        "Schüler"
    );

    XLSX.utils.book_append_sheet(
        workbook,
        XLSX.utils.aoa_to_sheet([
            ["MaxPoints", metadata.maxPoints],
            ["MinPoints", metadata.minPoints],
            ["BreakPointPercent", metadata.breakPointPercent]
        ]),
        "Meta"
    );

    return workbook;
}

export function workbookToUint8Array(workbook: XLSX.WorkBook): Uint8Array {
    const arrayBuffer = XLSX.write(workbook, { type: "array", bookType: "xlsx" }) as ArrayBuffer;
    return new Uint8Array(arrayBuffer);
}

export function triggerWorkbookDownload(workbook: XLSX.WorkBook, filename: string): void {
    const arrayBuffer = XLSX.write(workbook, { type: "array", bookType: "xlsx" }) as ArrayBuffer;
    const blob = new Blob([arrayBuffer], {
        type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
}