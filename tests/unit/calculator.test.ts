import { describe, expect, it } from "vitest";
import { calculateGradeBounds, validateGradeBounds } from "../../src/core/calculator";

describe("calculateGradeBounds", () => {
    it("calculates expected bounds for 45/0.5/50", () => {
        const bounds = calculateGradeBounds(45, 0.5, 50);
        expect(bounds).toEqual([
            { grade: 1, lowerBound: 39.5, upperBound: 45 },
            { grade: 2, lowerBound: 34, upperBound: 39 },
            { grade: 3, lowerBound: 28, upperBound: 33.5 },
            { grade: 4, lowerBound: 22.5, upperBound: 27.5 },
            { grade: 5, lowerBound: 0, upperBound: 22 }
        ]);
    });

    it("returns five bounds", () => {
        const bounds = calculateGradeBounds(100, 0.5, 50);
        expect(bounds).toHaveLength(5);
    });

    it("produces rounded lower bounds for minPoints", () => {
        const bounds = calculateGradeBounds(37, 0.25, 60);
        for (const bound of bounds) {
            const remainder = bound.lowerBound % 0.25;
            expect(remainder === 0 || Math.abs(remainder - 0.25) < 1e-9).toBe(true);
        }
    });
});

describe("validateGradeBounds", () => {
    it("accepts valid bounds", () => {
        expect(validateGradeBounds(calculateGradeBounds(100, 0.5, 50))).toHaveLength(0);
    });

    it("rejects insufficient bounds", () => {
        const d = validateGradeBounds([]);
        expect(d.some((x) => x.code === "degenerate-scale")).toBe(true);
    });

    it("rejects inverted ranges", () => {
        const d = validateGradeBounds([
            { grade: 1, lowerBound: 50, upperBound: 40 },
            { grade: 2, lowerBound: 30, upperBound: 39 },
            { grade: 3, lowerBound: 20, upperBound: 29 },
            { grade: 4, lowerBound: 10, upperBound: 19 },
            { grade: 5, lowerBound: 0, upperBound: 9 }
        ]);
        expect(d).toHaveLength(1);
        expect(d[0]?.code).toBe("degenerate-scale");
    });
});