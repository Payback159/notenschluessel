import { LIMITS } from "../constants";
import {
    noValidStudents,
    rowLimitReached,
    rowMissingName,
    rowNegativePoints,
    rowTooFewColumns,
    rowUnparsablePoints
} from "../core/diagnostics";
import { CSVParseResult, Diagnostic, Student } from "../types";

function sanitizeName(name: string): string {
    let result = name.replaceAll("<", "").replaceAll(">", "");
    result = result.replaceAll("\n", " ").replaceAll("\r", " ").replaceAll("\t", " ");
    result = result.trim();

    if (result.length > LIMITS.maxNameLength) {
        return result.slice(0, LIMITS.maxNameLength);
    }

    return result;
}

function parseCSVLine(line: string, delimiter: string): string[] {
    const fields: string[] = [];
    let current = "";
    let inQuotes = false;

    for (let i = 0; i < line.length; i++) {
        const char = line[i];
        const next = line[i + 1];

        if (char === '"') {
            if (inQuotes && next === '"') {
                current += '"';
                i++;
            } else {
                inQuotes = !inQuotes;
            }
        } else if (char === delimiter && !inQuotes) {
            fields.push(current);
            current = "";
        } else {
            current += char;
        }
    }

    fields.push(current);
    return fields;
}

export function detectDelimiter(content: string): "," | ";" {
    const sample = content.slice(0, 1024);
    const commaCount = (sample.match(/,/g) ?? []).length;
    const semicolonCount = (sample.match(/;/g) ?? []).length;
    return semicolonCount > commaCount ? ";" : ",";
}

export function parseCSVContent(content: string): CSVParseResult {
    const students: Student[] = [];
    const diagnostics: Diagnostic[] = [];

    if (content.trim() === "") {
        return { students: [], diagnostics: [noValidStudents()] };
    }

    const delimiter = detectDelimiter(content);
    const lines = content.split(/\r?\n/);
    let limitReached = false;

    for (let index = 0; index < lines.length; index++) {
        const line = lines[index];
        if (line === undefined || line.trim() === "") {
            continue;
        }

        // 1-based line number as the teacher sees it in an editor.
        const row = index + 1;
        const record = parseCSVLine(line, delimiter);
        const col0 = record[0];
        const col1 = record[1];

        if (index === 0 && col0 !== undefined && col0.trim().toLowerCase() === "name") {
            continue;
        }

        if (record.length < 2 || col0 === undefined || col1 === undefined) {
            diagnostics.push(rowTooFewColumns(row));
            continue;
        }

        const name = col0.trim();
        const rawPoints = col1.trim();

        if (name === "" && rawPoints === "") {
            continue;
        }

        if (name === "") {
            diagnostics.push(rowMissingName(row));
            continue;
        }

        const points = Number.parseFloat(rawPoints.replaceAll(",", "."));

        if (!Number.isFinite(points)) {
            diagnostics.push(rowUnparsablePoints(row, rawPoints));
            continue;
        }

        if (points < 0) {
            diagnostics.push(rowNegativePoints(row, points));
            continue;
        }

        if (points > LIMITS.maxPoints) {
            diagnostics.push(rowUnparsablePoints(row, rawPoints));
            continue;
        }

        if (students.length >= LIMITS.maxStudents) {
            limitReached = true;
            break;
        }

        students.push({ name: sanitizeName(name), points, sourceRow: row });
    }

    if (limitReached) {
        diagnostics.push(rowLimitReached(LIMITS.maxStudents));
    }

    if (students.length === 0 && !limitReached) {
        diagnostics.push(noValidStudents());
    }

    return { students, diagnostics };
}