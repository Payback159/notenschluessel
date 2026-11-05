import { LIMITS } from "../constants";
import {
    fileTooLarge,
    noValidStudents,
    rowLimitReached,
    rowMissingName,
    rowMissingPoints,
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
const POINTS_PATTERN = /^[+-]?\d+(?:[.,]\d+)?$/;

/**
 * Reads a points field strictly: German decimal comma or dot, no trailing
 * garbage or multiple decimal separators. Shared with `detectDelimiter` and
 * `manualParser` so all inputs are judged by the same rule.
 */
export function parsePointsField(raw: string): number {
    const trimmed = raw.trim();
    if (!POINTS_PATTERN.test(trimmed)) {
        return Number.NaN;
    }
    return Number.parseFloat(trimmed.replace(",", "."));
}

interface DelimiterScore {
    usableRows: number;
    cleanNames: number;
    multiColRows: number;
    fields: number;
}

function outranks(candidate: DelimiterScore, incumbent: DelimiterScore): boolean {
    if (candidate.usableRows !== incumbent.usableRows) {
        return candidate.usableRows > incumbent.usableRows;
    }
    if (candidate.cleanNames !== incumbent.cleanNames) {
        return candidate.cleanNames > incumbent.cleanNames;
    }
    if (candidate.multiColRows !== incumbent.multiColRows) {
        return candidate.multiColRows > incumbent.multiColRows;
    }
    return candidate.fields > incumbent.fields;
}

export function detectDelimiter(content: string): "," | ";" {
    const sampleLines = content
        .split(/\r?\n/)
        .filter((line) => line.trim() !== "")
        .slice(0, DELIMITER_SAMPLE_LINES);

    let best: { delimiter: "," | ";"; score: DelimiterScore } | undefined;

    for (const delimiter of DELIMITER_CANDIDATES) {
        const records = sampleLines.map((line) => parseCSVLine(line, delimiter));
        const multiColRecords = records.filter((record) => record.length >= 2);
        const first = multiColRecords[0]?.length;
        if (first === undefined) {
            continue;
        }

        const rivals = DELIMITER_CANDIDATES.filter((candidate) => candidate !== delimiter);
        const score: DelimiterScore = {
            usableRows: multiColRecords.filter((record) => {
                const rawPoints = record[1];
                return rawPoints !== undefined && Number.isFinite(parsePointsField(rawPoints));
            }).length,
            cleanNames: multiColRecords.filter((record) => {
                const name = record[0];
                return name !== undefined && !rivals.some((rival) => name.includes(rival));
            }).length,
            multiColRows: multiColRecords.length,
            fields: first
        };

        if (!best || outranks(score, best.score)) {
            best = { delimiter, score };
        }
    }

    return best?.delimiter ?? ",";
}

export function parseCSVContent(rawContent: string): CSVParseResult {
    if (rawContent.length > LIMITS.maxUploadSizeBytes) {
        return { students: [], diagnostics: [fileTooLarge()] };
    }

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

        const rawPoints = col1.trim();

        if (col0.trim() === "" && rawPoints === "") {
            continue;
        }

        const name = sanitizeName(col0);

        if (name === "") {
            diagnostics.push(rowMissingName(row));
            continue;
        }

        if (rawPoints === "") {
            diagnostics.push(rowMissingPoints(row));
            continue;
        }

        const points = parsePointsField(rawPoints);

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

        students.push({ name, points, sourceRow: row });
    }

    if (limitReached) {
        diagnostics.push(rowLimitReached(LIMITS.maxStudents));
    }

    if (students.length === 0 && !limitReached) {
        diagnostics.push(noValidStudents());
    }

    return { students, diagnostics };
}