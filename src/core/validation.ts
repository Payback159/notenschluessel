import { LIMITS } from "../constants";
import { inputModeConflict, inputModeMismatch, invalidCoreInput } from "./diagnostics";
import { Diagnostic, InputMode } from "../types";

export function validateCoreInputs(
    maxPoints: number,
    minPoints: number,
    breakPointPercent: number
): Diagnostic[] {
    const diagnostics: Diagnostic[] = [];

    if (!Number.isInteger(maxPoints) || maxPoints <= 0 || maxPoints > LIMITS.maxPoints) {
        diagnostics.push(invalidCoreInput("Ungültige maximale Punktzahl (1-1000 erlaubt)"));
    }

    if (!Number.isFinite(minPoints) || minPoints <= 0 || minPoints > maxPoints) {
        diagnostics.push(invalidCoreInput("Ungültige Punkteschrittweite"));
    }

    if (
        !Number.isFinite(breakPointPercent) ||
        breakPointPercent < LIMITS.minBreakPointPercent ||
        breakPointPercent > LIMITS.maxBreakPointPercent
    ) {
        diagnostics.push(invalidCoreInput("Ungültiger Knickpunkt (1-99% erlaubt)"));
    }

    return diagnostics;
}

export function validateInputMode(
    inputMode: InputMode,
    csvProvided: boolean,
    manualProvided: boolean
): Diagnostic[] {
    if (csvProvided && manualProvided) {
        return [inputModeConflict()];
    }

    const activeProvided = inputMode === "csv" ? csvProvided : manualProvided;
    const otherProvided = inputMode === "csv" ? manualProvided : csvProvided;

    if (!activeProvided && otherProvided) {
        return [inputModeMismatch(inputMode)];
    }

    return [];
}
