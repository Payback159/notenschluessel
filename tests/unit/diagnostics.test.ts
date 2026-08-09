import { describe, expect, it } from "vitest";
import {
    hasErrors,
    inputModeMismatch,
    pointsExceedMax,
    rowLimitReached,
    rowUnparsablePoints
} from "../../src/core/diagnostics";

describe("diagnostics", () => {
    it("marks row problems as warnings and carries the row number", () => {
        const d = rowUnparsablePoints(7, "achtzig");
        expect(d.severity).toBe("warning");
        expect(d.code).toBe("row-unparsable-points");
        expect(d.row).toBe(7);
        expect(d.message).toContain("Zeile 7");
        expect(d.message).toContain("achtzig");
    });

    it("marks points above the maximum as an error", () => {
        const d = pointsExceedMax(4, 855, 100);
        expect(d.severity).toBe("error");
        expect(d.code).toBe("points-exceed-max");
        expect(d.row).toBe(4);
        expect(d.message).toContain("855");
        expect(d.message).toContain("100");
    });

    it("reports a mode mismatch without a row number", () => {
        const d = inputModeMismatch("csv");
        expect(d.severity).toBe("error");
        expect(d.code).toBe("input-mode-mismatch");
        expect(d.row).toBeUndefined();
    });

    it("reports the row limit as a warning", () => {
        const d = rowLimitReached(10000);
        expect(d.severity).toBe("warning");
        expect(d.code).toBe("row-limit-reached");
        expect(d.message).toContain("10000");
    });

    it("detects whether a list contains errors", () => {
        expect(hasErrors([rowUnparsablePoints(1, "x")])).toBe(false);
        expect(hasErrors([rowUnparsablePoints(1, "x"), pointsExceedMax(2, 200, 100)])).toBe(true);
        expect(hasErrors([])).toBe(false);
    });

    it("writes umlauts properly instead of transliterating them", () => {
        // Guards against the "Ungueltige"/"gefaehrlich" style used elsewhere in the codebase.
        expect(pointsExceedMax(1, 2, 1).message).toContain("über");
        expect(rowUnparsablePoints(1, "x").message).toContain("übersprungen");
    });
});
