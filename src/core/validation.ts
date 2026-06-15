import { LIMITS } from "../constants";
import { InputMode, ValidationResult } from "../types";

export function validateCoreInputs(
    maxPoints: number,
    minPoints: number,
    breakPointPercent: number
): ValidationResult {
    const errors: string[] = [];

    if (!Number.isInteger(maxPoints) || maxPoints <= 0 || maxPoints > LIMITS.maxPoints) {
        errors.push("Ungueltige maximale Punktzahl (1-1000 erlaubt)");
    }

    if (!Number.isFinite(minPoints) || minPoints <= 0 || minPoints > maxPoints) {
        errors.push("Ungueltige Punkteschrittweite");
    }

    if (
        !Number.isFinite(breakPointPercent) ||
        breakPointPercent < LIMITS.minBreakPointPercent ||
        breakPointPercent > LIMITS.maxBreakPointPercent
    ) {
        errors.push("Ungueltiger Knickpunkt (1-99% erlaubt)");
    }

    return { valid: errors.length === 0, errors };
}

export function validateInputModeExclusivity(
    inputMode: InputMode,
    csvProvided: boolean,
    manualProvided: boolean
): ValidationResult {
    const errors: string[] = [];

    if (inputMode !== "csv" && inputMode !== "manual") {
        errors.push("Ungueltiger Eingabemodus");
    }

    if (csvProvided && manualProvided) {
        errors.push("Bitte entweder CSV-Import oder manuelle Eingabe verwenden. Eine Kombination ist nicht erlaubt.");
    }

    return { valid: errors.length === 0, errors };
}