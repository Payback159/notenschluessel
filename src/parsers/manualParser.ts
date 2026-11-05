import { LIMITS } from "../constants";
import {
    rowLimitReached,
    rowMissingName,
    rowMissingPoints,
    rowNegativePoints,
    rowUnparsablePoints
} from "../core/diagnostics";
import { sanitizeName } from "../core/sanitize";
import { Diagnostic, ManualEntry, ManualParseResult, Student } from "../types";
import { parsePointsField } from "./csvParser";

export function hasNonEmptyManualEntries(entries: ManualEntry[]): boolean {
    return entries.some((entry) => entry.name.trim() !== "" || entry.points.trim() !== "");
}

export function parseManualEntries(entries: ManualEntry[]): ManualParseResult {
    const students: Student[] = [];
    const diagnostics: Diagnostic[] = [];
    let limitReached = false;

    for (let i = 0; i < entries.length; i++) {
        const entry = entries[i];
        if (!entry) {
            continue;
        }

        const row = i + 1;
        const pointsRaw = entry.points.trim();

        if (entry.name.trim() === "" && pointsRaw === "") {
            continue;
        }

        const name = sanitizeName(entry.name);

        if (name === "") {
            diagnostics.push(rowMissingName(row));
            continue;
        }

        if (pointsRaw === "") {
            diagnostics.push(rowMissingPoints(row));
            continue;
        }

        const points = parsePointsField(pointsRaw);

        if (!Number.isFinite(points)) {
            diagnostics.push(rowUnparsablePoints(row, pointsRaw));
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

    return { students, diagnostics };
}