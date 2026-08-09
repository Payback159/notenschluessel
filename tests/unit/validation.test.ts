import { describe, expect, it } from "vitest";
import { validateCoreInputs, validateInputMode } from "../../src/core/validation";

describe("core validation", () => {
    it("accepts valid core input values", () => {
        expect(validateCoreInputs(100, 0.5, 50)).toHaveLength(0);
    });

    it("rejects invalid max points", () => {
        const d = validateCoreInputs(0, 0.5, 50);
        expect(d.some((x) => x.code === "invalid-core-input" && x.message.includes("maximale Punktzahl"))).toBe(true);
    });

    it("rejects invalid min points", () => {
        const d = validateCoreInputs(100, -1, 50);
        expect(d.some((x) => x.message.includes("Punkteschrittweite"))).toBe(true);
    });

    it("rejects invalid break point", () => {
        const d = validateCoreInputs(100, 0.5, 100);
        expect(d.some((x) => x.message.includes("Knickpunkt"))).toBe(true);
    });

    it("reports every core problem at once", () => {
        expect(validateCoreInputs(0, -1, 100)).toHaveLength(3);
    });
});

describe("input mode validation", () => {
    it("accepts csv-only in csv mode", () => {
        expect(validateInputMode("csv", true, false)).toHaveLength(0);
    });

    it("accepts manual-only in manual mode", () => {
        expect(validateInputMode("manual", false, true)).toHaveLength(0);
    });

    it("accepts an empty input in either mode", () => {
        expect(validateInputMode("csv", false, false)).toHaveLength(0);
        expect(validateInputMode("manual", false, false)).toHaveLength(0);
    });

    it("rejects combined inputs", () => {
        const d = validateInputMode("manual", true, true);
        expect(d.some((x) => x.code === "input-mode-conflict")).toBe(true);
    });

    it("rejects data sitting in the mode that is not selected", () => {
        const csv = validateInputMode("csv", false, true);
        expect(csv.some((x) => x.code === "input-mode-mismatch")).toBe(true);

        const manual = validateInputMode("manual", true, false);
        expect(manual.some((x) => x.code === "input-mode-mismatch")).toBe(true);
    });
});
