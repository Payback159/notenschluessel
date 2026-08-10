import { describe, expect, it } from "vitest";
import { runCalculationWorkflow } from "../../src/ui/workflow";

describe("input mode behavior", () => {
    it("allows empty student input and still calculates the grade scale", () => {
        const result = runCalculationWorkflow({
            maxPoints: 100,
            minPoints: 1,
            breakPointPercent: 50,
            inputMode: "csv",
            csvContent: "",
            manualEntries: []
        });

        expect(result.ok).toBe(true);
        expect(result.diagnostics).toHaveLength(0);
        expect(result.gradeBounds).toHaveLength(5);
        expect(result.students).toHaveLength(0);
    });

    it("returns an error for invalid core inputs", () => {
        const result = runCalculationWorkflow({
            maxPoints: 0,
            minPoints: 1,
            breakPointPercent: 50,
            inputMode: "csv",
            csvContent: "",
            manualEntries: []
        });

        expect(result.ok).toBe(false);
        expect(result.diagnostics.some((d) => d.code === "invalid-core-input")).toBe(true);
    });

    it("reports data sitting in the mode that is not selected", () => {
        const result = runCalculationWorkflow({
            maxPoints: 100,
            minPoints: 0.5,
            breakPointPercent: 50,
            inputMode: "csv",
            csvContent: "",
            manualEntries: [{ name: "Anna", points: "90" }]
        });

        expect(result.ok).toBe(false);
        expect(result.diagnostics.some((d) => d.code === "input-mode-mismatch")).toBe(true);
    });
});
