import { describe, expect, it } from "vitest";
import { exportCombinedCSV } from "../../src/export/csvExport";
import { buildCombinedWorkbook, workbookToUint8Array } from "../../src/export/excelExport";
import { runCalculationWorkflow } from "../../src/ui/workflow";

describe("full workflow", () => {
    it("runs csv-only flow and exports", () => {
        const result = runCalculationWorkflow({
            maxPoints: 100,
            minPoints: 0.5,
            breakPointPercent: 50,
            inputMode: "csv",
            csvContent: "Name,Punkte\nAlice,95\nBob,45",
            manualEntries: []
        });

        expect(result.ok).toBe(true);
        expect(result.gradeBounds).toHaveLength(5);
        expect(result.students).toHaveLength(2);

        const csv = exportCombinedCSV(result.gradeBounds, result.students, {
            maxPoints: 100,
            minPoints: 0.5,
            breakPointPercent: 50
        });
        expect(csv).toContain("Alice");

        const workbook = buildCombinedWorkbook(result.gradeBounds, result.students, {
            maxPoints: 100,
            minPoints: 0.5,
            breakPointPercent: 50
        });
        const bytes = workbookToUint8Array(workbook);
        expect(bytes.length).toBeGreaterThan(100);
    });

    it("runs manual-only flow", () => {
        const result = runCalculationWorkflow({
            maxPoints: 45,
            minPoints: 0.5,
            breakPointPercent: 50,
            inputMode: "manual",
            manualEntries: [
                { name: "Alice", points: "22.5" },
                { name: "Bob", points: "39.5" }
            ]
        });

        expect(result.ok).toBe(true);
        expect(result.students).toHaveLength(2);
        expect(result.students[0]?.grade).toBe(4);
        expect(result.students[1]?.grade).toBe(1);
    });

    it("rejects combining csv and manual", () => {
        const result = runCalculationWorkflow({
            maxPoints: 100,
            minPoints: 0.5,
            breakPointPercent: 50,
            inputMode: "manual",
            csvContent: "Name,Punkte\nAlice,95",
            manualEntries: [{ name: "Bob", points: "50" }]
        });

        expect(result.ok).toBe(false);
        expect(result.diagnostics.map((d) => d.message).join(" ")).toContain("Kombination");
    });

    it("calculates the remaining students and warns about every skipped row", () => {
        const result = runCalculationWorkflow({
            maxPoints: 100,
            minPoints: 0.5,
            breakPointPercent: 50,
            inputMode: "csv",
            csvContent: "Name,Punkte\nAlice,95\nKaputt\nBob,abc\nCharlie,20",
            manualEntries: []
        });

        expect(result.ok).toBe(true);
        expect(result.students).toHaveLength(2);
        expect(result.diagnostics).toHaveLength(2);
        expect(result.diagnostics.every((d) => d.severity === "warning")).toBe(true);
    });

    it("aborts when a score exceeds the maximum, naming the row", () => {
        const result = runCalculationWorkflow({
            maxPoints: 100,
            minPoints: 0.5,
            breakPointPercent: 50,
            inputMode: "csv",
            csvContent: "Name,Punkte\nTippfehler,855\nNormal,85.5",
            manualEntries: []
        });

        expect(result.ok).toBe(false);
        expect(result.students).toHaveLength(0);
        expect(result.gradeBounds).toHaveLength(0);

        const exceed = result.diagnostics.find((d) => d.code === "points-exceed-max");
        expect(exceed?.row).toBe(2);
        expect(exceed?.message).toContain("855");
    });

    it("reports an unusable scale", () => {
        const result = runCalculationWorkflow({
            maxPoints: 1,
            minPoints: 1,
            breakPointPercent: 99,
            inputMode: "csv",
            csvContent: "",
            manualEntries: []
        });

        expect(result.ok).toBe(false);
        expect(result.diagnostics.some((d) => d.code === "degenerate-scale")).toBe(true);
    });
});