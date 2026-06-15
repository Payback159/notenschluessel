import { LIMITS } from "../constants";
import { ManualEntry, ManualParseResult, Student } from "../types";

function sanitizeName(name: string): string {
    let result = name.replaceAll("<", "").replaceAll(">", "");
    result = result.replaceAll("\n", " ").replaceAll("\r", " ").replaceAll("\t", " ");
    result = result.trim();

    if (result.length > LIMITS.maxNameLength) {
        return result.slice(0, LIMITS.maxNameLength);
    }

    return result;
}

export function hasNonEmptyManualEntries(entries: ManualEntry[]): boolean {
    return entries.some((entry) => entry.name.trim() !== "" || entry.points.trim() !== "");
}

export function parseManualEntries(entries: ManualEntry[]): ManualParseResult {
    const students: Student[] = [];
    const errors: string[] = [];

    for (let i = 0; i < entries.length; i++) {
        const entry = entries[i];
        if (!entry) {
            continue;
        }

        const name = entry.name.trim();
        const pointsRaw = entry.points.trim();

        if (name === "" && pointsRaw === "") {
            continue;
        }

        if (name === "") {
            errors.push(`Zeile ${i + 1}: Name fehlt`);
            continue;
        }

        if (pointsRaw === "") {
            errors.push(`Zeile ${i + 1}: Punkte fehlen`);
            continue;
        }

        const points = Number.parseFloat(pointsRaw.replaceAll(",", "."));
        if (!Number.isFinite(points)) {
            errors.push(`Zeile ${i + 1}: Ungueltige Punktzahl`);
            continue;
        }

        if (points < 0 || points > LIMITS.maxPoints) {
            errors.push(`Zeile ${i + 1}: Punktzahl ausserhalb des erlaubten Bereichs (0-1000)`);
            continue;
        }

        students.push({
            name: sanitizeName(name),
            points
        });

        if (students.length > LIMITS.maxStudents) {
            errors.push(`Zu viele Schülerdaten (maximal ${LIMITS.maxStudents})`);
            return { students: [], errors };
        }
    }

    return { students, errors };
}