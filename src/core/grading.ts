import { GradeBound, Student } from "../types";

/** Expects `gradeBounds` ordered from grade 1 down to 5, as `calculateGradeBounds` returns them. */
export function processStudents(students: Student[], gradeBounds: GradeBound[]): Student[] {
    return students.map((student) => ({
        ...student,
        grade: gradeBounds.find((bound) => student.points >= bound.lowerBound)?.grade ?? 5
    }));
}

export function calculateAverageGrade(students: Student[]): number {
    if (students.length === 0) {
        return 0;
    }

    const sum = students.reduce((acc, student) => acc + (student.grade ?? 0), 0);
    const average = sum / students.length;
    return Math.round(average * 100) / 100;
}
