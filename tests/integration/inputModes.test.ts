import { describe, expect, it } from "vitest";
import { runCalculationWorkflow } from "../../src/ui/workflow";

describe("input mode behavior", () => {
    it("allows empty student input and still calculates grade scale", () => {
        const result = runCalculationWorkflow({
            maxPoints: 100,
            minPoints: 1,
            breakPointPercent: 50,
            inputMode: "csv",
            csvContent: "",
            manualEntries: []
        });

        expect(result.ok).toBe(true);
        expect(result.gradeBounds).toHaveLength(5);
        expect(result.students).toHaveLength(0);
    });

    it("returns error for invalid core inputs", () => {
        const result = runCalculationWorkflow({
            maxPoints: 0,
            minPoints: 1,
            breakPointPercent: 50,
            inputMode: "csv",
            csvContent: "",
            manualEntries: []
        });

        expect(result.ok).toBe(false);
        expect(result.errors.length).toBeGreaterThan(0);
    });
});