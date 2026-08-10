import { pointsExceedMax } from "./diagnostics";
import { Diagnostic, Student } from "../types";

/**
 * Checks parsed students against the maximum score the teacher entered.
 *
 * This cannot live in the parsers: they do not know the maximum and only guard
 * against the absolute ceiling in `LIMITS`. A score above the maximum is almost
 * always a typo that would otherwise silently produce a top grade.
 */
export function validateStudentPoints(students: Student[], maxPoints: number): Diagnostic[] {
    const diagnostics: Diagnostic[] = [];

    for (const student of students) {
        if (student.points > maxPoints) {
            diagnostics.push(pointsExceedMax(student.sourceRow ?? 0, student.points, maxPoints));
        }
    }

    return diagnostics;
}
