import { GradeBound, Student } from "../types";

function sanitizeCSVField(field: string): string {
    if (!field) {
        return "";
    }

    const firstChar = field[0] ?? "";
    if ("=+-@\t\r".includes(firstChar)) {
        return `'${field}`;
    }

    if (field.includes(",") || field.includes('"') || field.includes("\n")) {
        return `"${field.replaceAll('"', '""')}"`;
    }

    return field;
}

export function exportGradeScaleCSV(bounds: GradeBound[]): string {
    const lines = ["Note,Untergrenze,Obergrenze"];
    for (const b of bounds) {
        lines.push(`${b.grade},${b.lowerBound},${b.upperBound}`);
    }
    return lines.join("\n");
}

export function exportStudentResultsCSV(students: Student[]): string {
    const lines = ["Name,Punkte,Note"];
    for (const s of students) {
        lines.push(`${sanitizeCSVField(s.name)},${s.points},${s.grade ?? ""}`);
    }
    return lines.join("\n");
}

export function exportCombinedCSV(
    bounds: GradeBound[],
    students: Student[],
    metadata: { maxPoints: number; minPoints: number; breakPointPercent: number }
): string {
    const lines: string[] = [];
    lines.push("# Notenschlüssel Export");
    lines.push(`# MaxPoints: ${metadata.maxPoints}`);
    lines.push(`# MinPoints: ${metadata.minPoints}`);
    lines.push(`# BreakPointPercent: ${metadata.breakPointPercent}`);
    lines.push("");
    lines.push("## Notenskala");
    lines.push(exportGradeScaleCSV(bounds));
    lines.push("");
    lines.push("## Schüler");
    lines.push(exportStudentResultsCSV(students));
    return lines.join("\n");
}

export function triggerTextDownload(content: string, filename: string, mimeType: string): void {
    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
}