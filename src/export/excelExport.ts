import * as XLSX from "xlsx-js-style";
import { GRADE_COLORS } from "../constants";
import { GradeBound, Student } from "../types";

function getGradeStyle(grade: number) {
    const colors = GRADE_COLORS[grade as keyof typeof GRADE_COLORS];
    const textColorHex = colors.text.substring(1).toUpperCase(); // Remove #
    const bgColorHex = colors.bg.substring(1).toUpperCase(); // Remove #

    return {
        fill: {
            patternType: "solid",
            fgColor: { rgb: bgColorHex }
        },
        font: {
            color: { rgb: textColorHex },
            bold: true
        },
        alignment: { horizontal: "center", vertical: "center" }
    };
}

function applyCellStyle(sheet: XLSX.WorkSheet, cellRef: string, style: XLSX.CellObject["s"]) {
    if (!sheet[cellRef]) {
        sheet[cellRef] = { t: "n", v: 0 };
    }
    sheet[cellRef]!.s = style;
}

function styleHeaderRow(sheet: XLSX.WorkSheet): void {
    for (const col of ["A", "B", "C"]) {
        applyCellStyle(sheet, `${col}1`, {
            fill: { patternType: "solid", fgColor: { rgb: "F0F0F0" } },
            font: { bold: true }
        });
    }
}

function styleGradeRow(sheet: XLSX.WorkSheet, rowIndex: number, grade: number): void {
    const style = getGradeStyle(grade);
    for (const col of ["A", "B", "C"]) {
        applyCellStyle(sheet, `${col}${rowIndex}`, style);
    }
}

export function buildGradeScaleWorkbook(bounds: GradeBound[]): XLSX.WorkBook {
    const workbook = XLSX.utils.book_new();
    const data = bounds.map((b) => ({
        Note: b.grade,
        Untergrenze: b.lowerBound,
        Obergrenze: b.upperBound
    }));
    const sheet = XLSX.utils.json_to_sheet(data);

    // Apply styles to header row
    styleHeaderRow(sheet);

    // Apply grade colors
    bounds.forEach((bound, i) => {
        styleGradeRow(sheet, i + 2, bound.grade);
    });

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

    // Apply styles to header row
    styleHeaderRow(sheet);

    // Apply grade colors to the whole row
    students.forEach((student, i) => {
        if (student.grade) {
            styleGradeRow(sheet, i + 2, student.grade);
        }
    });

    XLSX.utils.book_append_sheet(workbook, sheet, "Schüler");
    return workbook;
}

export function buildCombinedWorkbook(
    bounds: GradeBound[],
    students: Student[],
    metadata: { maxPoints: number; minPoints: number; breakPointPercent: number }
): XLSX.WorkBook {
    const workbook = XLSX.utils.book_new();

    // Grade scale sheet
    const scaleData = bounds.map((b) => ({ Note: b.grade, Untergrenze: b.lowerBound, Obergrenze: b.upperBound }));
    const scaleSheet = XLSX.utils.json_to_sheet(scaleData);

    styleHeaderRow(scaleSheet);

    bounds.forEach((bound, i) => {
        styleGradeRow(scaleSheet, i + 2, bound.grade);
    });
    XLSX.utils.book_append_sheet(workbook, scaleSheet, "Notenskala");

    // Students sheet
    const studentData = students.map((s) => ({ Name: s.name, Punkte: s.points, Note: s.grade ?? "" }));
    const studentSheet = XLSX.utils.json_to_sheet(studentData);

    styleHeaderRow(studentSheet);

    students.forEach((student, i) => {
        if (student.grade) {
            styleGradeRow(studentSheet, i + 2, student.grade);
        }
    });
    XLSX.utils.book_append_sheet(workbook, studentSheet, "Schüler");

    // Metadata sheet
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