import { LIMITS } from "../constants";
import { inputModeConflict, inputModeMismatch, invalidCoreInput } from "./diagnostics";
import { Diagnostic, GradeBound, InputMode, Student } from "../types";

export interface PersistedState {
    maxPoints: number;
    minPoints: number;
    breakPointPercent: number;
    gradeBounds: GradeBound[];
    students: Student[];
    averageGrade: number;
    inputMode?: InputMode;
}

function isValidGrade(grade: unknown): grade is number {
    return typeof grade === "number" && Number.isInteger(grade) && grade >= 1 && grade <= 5;
}

export function isValidRestoredState(value: unknown): value is PersistedState {
    if (!value || typeof value !== "object") {
        return false;
    }

    const candidate = value as Record<string, unknown>;
    if (
        typeof candidate["maxPoints"] !== "number" ||
        typeof candidate["minPoints"] !== "number" ||
        typeof candidate["breakPointPercent"] !== "number" ||
        typeof candidate["averageGrade"] !== "number" ||
        !Number.isFinite(candidate["averageGrade"])
    ) {
        return false;
    }

    if (
        validateCoreInputs(
            candidate["maxPoints"],
            candidate["minPoints"],
            candidate["breakPointPercent"]
        ).length > 0
    ) {
        return false;
    }

    if (!Array.isArray(candidate["gradeBounds"]) || !Array.isArray(candidate["students"])) {
        return false;
    }

    const boundsValid = candidate["gradeBounds"].every((b: unknown) => {
        if (!b || typeof b !== "object") {
            return false;
        }
        const bound = b as Record<string, unknown>;
        return (
            isValidGrade(bound["grade"]) &&
            typeof bound["lowerBound"] === "number" &&
            Number.isFinite(bound["lowerBound"]) &&
            typeof bound["upperBound"] === "number" &&
            Number.isFinite(bound["upperBound"])
        );
    });

    if (!boundsValid) {
        return false;
    }

    return candidate["students"].every((s: unknown) => {
        if (!s || typeof s !== "object") {
            return false;
        }
        const student = s as Record<string, unknown>;
        return (
            typeof student["name"] === "string" &&
            typeof student["points"] === "number" &&
            Number.isFinite(student["points"]) &&
            (student["grade"] === undefined || isValidGrade(student["grade"]))
        );
    });
}

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
