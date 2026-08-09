import { describe, expect, it } from "vitest";
import { decimalsFor, formatNumberDe, formatPoints, formatRange } from "../../src/ui/format";
import { calculateGradeBounds } from "../../src/core/calculator";

describe("decimalsFor", () => {
    it("derives the decimals from the step size", () => {
        expect(decimalsFor(1)).toBe(0);
        expect(decimalsFor(0.5)).toBe(1);
        expect(decimalsFor(0.25)).toBe(2);
        expect(decimalsFor(0.1)).toBe(1);
    });

    it("caps at three decimals", () => {
        expect(decimalsFor(0.00001)).toBe(3);
    });
});

describe("formatNumberDe", () => {
    it("uses a decimal comma", () => {
        expect(formatNumberDe(76.5, 1)).toBe("76,5");
        expect(formatNumberDe(100, 0)).toBe("100");
        expect(formatNumberDe(33.25, 2)).toBe("33,25");
    });
});

describe("formatPoints", () => {
    it("keeps quarter points intact instead of rounding them away", () => {
        // The bug from the review: toFixed(1) turned 33.25 into "33.3".
        expect(formatPoints(33.25, 0.25)).toBe("33,25");
        expect(formatPoints(39.5, 0.5)).toBe("39,5");
        expect(formatPoints(45, 1)).toBe("45");
    });
});

describe("formatRange", () => {
    it("formats a grade boundary with the step size precision", () => {
        const bounds = calculateGradeBounds(37, 0.25, 60);
        const grade1 = bounds[0];
        expect(grade1).toBeDefined();
        expect(grade1?.lowerBound).toBe(33.25);
        expect(formatRange(grade1!, 0.25)).toBe("33,25 - 37");
    });

    it("matches the displayed scale for half points", () => {
        const bounds = calculateGradeBounds(45, 0.5, 50);
        expect(formatRange(bounds[0]!, 0.5)).toBe("39,5 - 45");
        expect(formatRange(bounds[4]!, 0.5)).toBe("0 - 22");
    });
});
