import { describe, expect, it } from "vitest";
import { buildDiagnosticLines, summaryLine } from "../../src/ui/render";
import { pointsExceedMax, rowUnparsablePoints } from "../../src/core/diagnostics";

describe("buildDiagnosticLines", () => {
    it("returns one line per diagnostic, errors first", () => {
        const lines = buildDiagnosticLines([
            rowUnparsablePoints(3, "abc"),
            pointsExceedMax(2, 855, 100)
        ]);

        expect(lines).toHaveLength(2);
        expect(lines[0]).toContain("855");
        expect(lines[1]).toContain("abc");
    });

    it("returns an empty list for no diagnostics", () => {
        expect(buildDiagnosticLines([])).toHaveLength(0);
    });
});

describe("summaryLine", () => {
    it("names the student count when warnings occurred", () => {
        expect(summaryLine(25, true)).toContain("25");
    });

    it("stays short when everything went through", () => {
        expect(summaryLine(25, false)).toBe("Berechnung abgeschlossen.");
    });
});
