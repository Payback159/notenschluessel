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
        const name = entry.name.trim();
        const pointsRaw = entry.points.trim();

        if (name === "" && pointsRaw === "") {
            continue;
        }

        if (name === "") {
            diagnostics.push(rowMissingName(row));
            continue;
        }

        if (pointsRaw === "") {
            diagnostics.push(rowMissingPoints(row));
            continue;
        }

        const points = Number.parseFloat(pointsRaw.replaceAll(",", "."));

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

        students.push({ name: sanitizeName(name), points, sourceRow: row });
    }

    if (limitReached) {
        diagnostics.push(rowLimitReached(LIMITS.maxStudents));
    }

    return { students, diagnostics };
}