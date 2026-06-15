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
        const bounds = calculateGradeBounds(100, 0.5, 50);
        expect(validateGradeBounds(bounds)).toEqual({ valid: true, reason: "" });
    });

    it("rejects insufficient bounds", () => {
        expect(validateGradeBounds([])).toEqual({
            valid: false,
            reason: "insufficient grade bounds"
        });
    });

    it("rejects inverted ranges", () => {
        const result = validateGradeBounds([
            { grade: 1, lowerBound: 50, upperBound: 40 },
            { grade: 2, lowerBound: 30, upperBound: 39 },
            { grade: 3, lowerBound: 20, upperBound: 29 },
            { grade: 4, lowerBound: 10, upperBound: 19 },
            { grade: 5, lowerBound: 0, upperBound: 9 }
        ]);
        expect(result.valid).toBe(false);
    });
});