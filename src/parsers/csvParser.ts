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
 * Reads a points field the way the import loop does: German decimal comma first,
 * then float. Shared with `detectDelimiter` so a delimiter is judged by the very
 * rule that later decides whether a row is usable.
 */
function parsePointsField(raw: string): number {
    return Number.parseFloat(raw.trim().replaceAll(",", "."));
}

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
 *
 * Field count alone is still not enough. Without a header row, splitting
 * "Nachname, Vorname, Jr.;76,5;2" on the comma yields a consistent four fields
 * while the correct semicolon yields three, so "more fields wins" would pick the
 * comma and lose every row. The number of rows whose second field actually reads
 * as a points value therefore decides first.
 *
 * That still leaves a tie for a header-less semicolon file whose points all carry
 * a decimal comma: "Müller;76,5" splits into a usable ["Müller;76", "5"] under the
 * comma too. So a name field that swallowed the rival delimiter counts against a
 * candidate — it is the tell-tale of a wrong split. Field count only breaks what
 * is left.
 */
interface DelimiterScore {
    usableRows: number;
    cleanNames: number;
    fields: number;
}

function outranks(candidate: DelimiterScore, incumbent: DelimiterScore): boolean {
    if (candidate.usableRows !== incumbent.usableRows) {
        return candidate.usableRows > incumbent.usableRows;
    }
    if (candidate.cleanNames !== incumbent.cleanNames) {
        return candidate.cleanNames > incumbent.cleanNames;
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
        const first = records[0]?.length;
        if (first === undefined || first < 2) {
            continue;
        }

        const consistent = records.every((record) => record.length === first);
        if (!consistent) {
            continue;
        }

        const rivals = DELIMITER_CANDIDATES.filter((candidate) => candidate !== delimiter);
        const score: DelimiterScore = {
            usableRows: records.filter((record) => {
                const rawPoints = record[1];
                return rawPoints !== undefined && Number.isFinite(parsePointsField(rawPoints));
            }).length,
            cleanNames: records.filter((record) => {
                const name = record[0];
                return name !== undefined && !rivals.some((rival) => name.includes(rival));
            }).length,
            fields: first
        };

        if (!best || outranks(score, best.score)) {
            best = { delimiter, score };
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