import { describe, expect, it } from "vitest";
import { calculateGradeBounds } from "../../src/core/calculator";
import { calculateAverageGrade, calculateGrade, processStudents } from "../../src/core/grading";

describe("grading", () => {
    const bounds = calculateGradeBounds(100, 0.5, 50);

    it("assigns grade by lower bounds", () => {
        expect(calculateGrade(95, 87.5, 75, 62.5, 50, 0)).toBe(1);
        expect(calculateGrade(75, 87.5, 75, 62.5, 50, 0)).toBe(2);
        expect(calculateGrade(50, 87.5, 75, 62.5, 50, 0)).toBe(4);
        expect(calculateGrade(49.5, 87.5, 75, 62.5, 50, 0)).toBe(5);
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

    it("keeps students unchanged for insufficient bounds", () => {
        const input = [{ name: "Test", points: 50 }];
        const result = processStudents(input, [{ grade: 1, lowerBound: 90, upperBound: 100 }]);
        expect(result[0]).toBeDefined();
        expect(result[0]?.grade).toBeUndefined();
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
});