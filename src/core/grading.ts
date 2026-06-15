import { GradeBound, Student } from "../types";

export function calculateGrade(
    points: number,
    lowerBound1: number,
    lowerBound2: number,
    lowerBound3: number,
    lowerBound4: number,
    _lowerBound5: number
): number {
    if (points >= lowerBound1) {
        return 1;
    }
    if (points >= lowerBound2) {
        return 2;
    }
    if (points >= lowerBound3) {
        return 3;
    }
    if (points >= lowerBound4) {
        return 4;
    }
    return 5;
}

export function processStudents(students: Student[], gradeBounds: GradeBound[]): Student[] {
    if (gradeBounds.length < 5) {
        return students;
    }

    const [bound1, bound2, bound3, bound4, bound5] = gradeBounds;
    if (!bound1 || !bound2 || !bound3 || !bound4 || !bound5) {
        return students;
    }

    const lowerBound1 = bound1.lowerBound;
    const lowerBound2 = bound2.lowerBound;
    const lowerBound3 = bound3.lowerBound;
    const lowerBound4 = bound4.lowerBound;
    const lowerBound5 = bound5.lowerBound;

    return students.map((student) => ({
        ...student,
        grade: calculateGrade(
            student.points,
            lowerBound1,
            lowerBound2,
            lowerBound3,
            lowerBound4,
            lowerBound5
        )
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