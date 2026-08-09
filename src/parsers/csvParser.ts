import { LIMITS } from "../constants";
import {
    noValidStudents,
    rowLimitReached,
    rowMissingName,
    rowNegativePoints,
    rowTooFewColumns,
    rowUnparsablePoints
} from "../core/diagnostics";
import { sanitizeName } from "../core/sanitize";
import { CSVParseResult, Diagnostic, Student } from "../types";

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

const DELIMITER_CANDIDATES = [",", ";"] as const;
const DELIMITER_SAMPLE_LINES = 5;

/**
 * Picks the delimiter structurally instead of by counting raw characters.
 *
 * Counting `,` and `;` across the first kilobyte breaks on the app's own export:
 * a semicolon-delimited row with a decimal comma and a name containing commas
 * (e.g. "Nachname, Vorname, Jr.") contributes as many commas as semicolons, and a
 * few such rows tip the raw count towards comma even though the file is
 * semicolon-delimited. Splitting each candidate with the real quote-aware parser
 * and requiring a consistent field count of at least two per line is immune to
 * that, because a wrong delimiter almost never produces the same field count on
 * every sampled line.
 */
export function detectDelimiter(content: string): "," | ";" {
    const sampleLines = content
        .split(/\r?\n/)
        .filter((line) => line.trim() !== "")
        .slice(0, DELIMITER_SAMPLE_LINES);

    let best: { delimiter: "," | ";"; fields: number } | undefined;

    for (const delimiter of DELIMITER_CANDIDATES) {
        const fieldCounts = sampleLines.map((line) => parseCSVLine(line, delimiter).length);
        const first = fieldCounts[0];
        if (first === undefined || first < 2) {
            continue;
        }

        const consistent = fieldCounts.every((count) => count === first);
        if (!consistent) {
            continue;
        }

        if (!best || first > best.fields) {
            best = { delimiter, fields: first };
        }
    }

    return best?.delimiter ?? ",";
}

export function parseCSVContent(rawContent: string): CSVParseResult {
    // A file exported by this app starts with a BOM so Excel reads it as UTF-8.
    const content = rawContent.startsWith("\uFEFF") ? rawContent.slice(1) : rawContent;

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