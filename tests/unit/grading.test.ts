import { describe, expect, it } from "vitest";
import { calculateGradeBounds } from "../../src/core/calculator";
import { calculateAverageGrade, processStudents } from "../../src/core/grading";

describe("grading", () => {
    const bounds = calculateGradeBounds(100, 0.5, 50);

    it("assigns grade by lower bounds", () => {
        const grades = processStudents(
            [95, 75, 50, 49.5].map((points) => ({ name: "S", points })),
            bounds
        ).map((s) => s.grade);
        expect(grades).toEqual([1, 2, 4, 5]);
    });

    it("processes student grades", () => {
        const result = processStudents(
            [
                { name: "Alice", points: 95 },
                { name: "Bob", points: 76 },
                { name: "Charlie", points: 40 }
            ],
            bounds
        );

        expect(result[0]).toBeDefined();
        expect(result[1]).toBeDefined();
        expect(result[2]).toBeDefined();
        expect(result[0]?.grade).toBe(1);
        expect(result[1]?.grade).toBe(2);
        expect(result[2]?.grade).toBe(5);
    });

    it("calculates average with 2 decimals", () => {
        const avg = calculateAverageGrade([
            { name: "A", points: 0, grade: 1 },
            { name: "B", points: 0, grade: 1 },
            { name: "C", points: 0, grade: 2 }
        ]);

        expect(avg).toBe(1.33);
    });

    it("returns 0 average for empty list", () => {
        expect(calculateAverageGrade([])).toBe(0);
    });

    it("awards grade 1 when student score equals a 0.1-step lower bound", () => {
        const scale = calculateGradeBounds(50, 0.1, 50);
        const graded = processStudents([{ name: "Exact", points: 43.8 }], scale);
        expect(graded[0]?.grade).toBe(1);
    });
});