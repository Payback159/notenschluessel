import { describe, expect, it } from "vitest";
import { validateCoreInputs, validateInputModeExclusivity } from "../../src/core/validation";

describe("core validation", () => {
    it("accepts valid core input values", () => {
        const result = validateCoreInputs(100, 0.5, 50);
        expect(result.valid).toBe(true);
        expect(result.errors).toHaveLength(0);
    });

    it("rejects invalid max points", () => {
        const result = validateCoreInputs(0, 0.5, 50);
        expect(result.valid).toBe(false);
        expect(result.errors.some((error) => error.includes("maximale Punktzahl"))).toBe(true);
    });

    it("rejects invalid min points", () => {
        const result = validateCoreInputs(100, -1, 50);
        expect(result.valid).toBe(false);
        expect(result.errors.some((error) => error.includes("Punkteschrittweite"))).toBe(true);
    });

    it("rejects invalid break point", () => {
        const result = validateCoreInputs(100, 0.5, 100);
        expect(result.valid).toBe(false);
        expect(result.errors.some((error) => error.includes("Knickpunkt"))).toBe(true);
    });
});

describe("input mode exclusivity", () => {
    it("accepts csv-only", () => {
        const result = validateInputModeExclusivity("csv", true, false);
        expect(result.valid).toBe(true);
    });

    it("accepts manual-only", () => {
        const result = validateInputModeExclusivity("manual", false, true);
        expect(result.valid).toBe(true);
    });

    it("rejects combined inputs", () => {
        const result = validateInputModeExclusivity("manual", true, true);
        expect(result.valid).toBe(false);
        expect(result.errors[0]).toContain("Kombination");
    });
});