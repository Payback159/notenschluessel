import * as XLSX from "xlsx-js-style";
import { GRADE_COLORS } from "../constants";
import { GradeBound, Student } from "../types";
import { guardFormulaPrefix } from "./csvExport";

function getGradeStyle(grade: number): XLSX.CellObject["s"] | undefined {
    const colors = GRADE_COLORS[grade as keyof typeof GRADE_COLORS];
    if (!colors) {
        return undefined;
    }
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
    if (!style) {
        return;
    }
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

function createGradeScaleSheet(bounds: GradeBound[]): XLSX.WorkSheet {
    const data = bounds.map((b) => ({
        Note: b.grade,
        Untergrenze: b.lowerBound,
        Obergrenze: b.upperBound
    }));
    const sheet = XLSX.utils.json_to_sheet(data);
    styleHeaderRow(sheet);
    bounds.forEach((bound, i) => {
        styleGradeRow(sheet, i + 2, bound.grade);
    });
    return sheet;
}

function createStudentsSheet(students: Student[]): XLSX.WorkSheet {
    const data = students.map((s) => ({
        Name: guardFormulaPrefix(s.name),
        Punkte: s.points,
        Note: s.grade ?? ""
    }));
    const sheet = XLSX.utils.json_to_sheet(data);
    styleHeaderRow(sheet);
    students.forEach((student, i) => {
        if (student.grade) {
            styleGradeRow(sheet, i + 2, student.grade);
        }
    });
    return sheet;
}

export function buildGradeScaleWorkbook(bounds: GradeBound[]): XLSX.WorkBook {
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, createGradeScaleSheet(bounds), "Notenskala");
    return workbook;
}

export function buildStudentsWorkbook(students: Student[]): XLSX.WorkBook {
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, createStudentsSheet(students), "Schüler");
    return workbook;
}

export function buildCombinedWorkbook(
    bounds: GradeBound[],
    students: Student[],
    metadata: { maxPoints: number; minPoints: number; breakPointPercent: number }
): XLSX.WorkBook {
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, createGradeScaleSheet(bounds), "Notenskala");
    XLSX.utils.book_append_sheet(workbook, createStudentsSheet(students), "Schüler");
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