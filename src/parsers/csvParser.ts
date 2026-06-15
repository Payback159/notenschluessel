import { LIMITS } from "../constants";
import { CSVParseResult, Student } from "../types";

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
    const errors: string[] = [];
    let skippedRows = 0;

    if (content.trim() === "") {
        return { students: [], skippedRows: 0, errors: ["Leere CSV-Datei"] };
    }

    const delimiter = detectDelimiter(content);
    const lines = content.split(/\r?\n/);

    for (let rowNum = 0; rowNum < lines.length; rowNum++) {
        const line = lines[rowNum];
        if (line === undefined) {
            continue;
        }

        if (line.trim() === "") {
            continue;
        }

        const record = parseCSVLine(line, delimiter);

        const col0 = record[0];
        const col1 = record[1];

        if (rowNum === 0 && col0 !== undefined && col0.trim().toLowerCase() === "name") {
            continue;
        }

        if (record.length < 2) {
            skippedRows++;
            continue;
        }

        if (col0 === undefined || col1 === undefined) {
            skippedRows++;
            continue;
        }

        const name = col0.trim();
        const rawPoints = col1.trim();

        if (name === "" && rawPoints === "") {
            continue;
        }

        if (name === "") {
            skippedRows++;
            continue;
        }

        const pointsStr = rawPoints.replaceAll(",", ".");
        const points = Number.parseFloat(pointsStr);

        if (!Number.isFinite(points)) {
            skippedRows++;
            continue;
        }

        if (points < 0 || points > LIMITS.maxPoints) {
            skippedRows++;
            continue;
        }

        students.push({
            name: sanitizeName(name),
            points
        });

        if (students.length >= LIMITS.maxStudents) {
            break;
        }
    }

    if (students.length === 0) {
        errors.push("Keine gültigen Schülerdaten in CSV gefunden");
    }

    return { students, skippedRows, errors };
}