import { formatNumberDe, formatPoints } from "../ui/format";
import { GradeBound, Student } from "../types";

const DELIMITER = ";";
const FORMULA_PREFIXES = "=+-@\t\r";

/**
 * Guards a field against spreadsheet formula injection **and** against breaking
 * the column layout. Both steps run: prefixing alone left a value containing the
 * delimiter split across cells.
 */
function sanitizeCSVField(field: string): string {
    if (!field) {
        return "";
    }

    const firstChar = field[0] ?? "";
    const guarded = FORMULA_PREFIXES.includes(firstChar) ? `'${field}` : field;

    if (
        guarded.includes(DELIMITER) ||
        guarded.includes('"') ||
        guarded.includes("\n") ||
        guarded !== field
    ) {
        return `"${guarded.replaceAll('"', '""')}"`;
    }

    return guarded;
}

export function exportGradeScaleCSV(bounds: GradeBound[], minPoints: number): string {
    const lines = [["Note", "Untergrenze", "Obergrenze"].join(DELIMITER)];
    for (const b of bounds) {
        lines.push(
            [
                String(b.grade),
                formatPoints(b.lowerBound, minPoints),
                formatPoints(b.upperBound, minPoints)
            ].join(DELIMITER)
        );
    }
    return lines.join("\n");
}

export function exportStudentResultsCSV(students: Student[], minPoints: number): string {
    const lines = [["Name", "Punkte", "Note"].join(DELIMITER)];
    for (const s of students) {
        lines.push(
            [
                sanitizeCSVField(s.name),
                formatPoints(s.points, minPoints),
                s.grade === undefined ? "" : String(s.grade)
            ].join(DELIMITER)
        );
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
    lines.push(`# MinPoints: ${formatNumberDe(metadata.minPoints, 2)}`);
    lines.push(`# BreakPointPercent: ${metadata.breakPointPercent}`);
    lines.push("");
    lines.push("## Notenskala");
    lines.push(exportGradeScaleCSV(bounds, metadata.minPoints));
    lines.push("");
    lines.push("## Schüler");
    lines.push(exportStudentResultsCSV(students, metadata.minPoints));
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
