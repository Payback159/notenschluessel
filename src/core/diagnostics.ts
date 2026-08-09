import { Diagnostic, InputMode } from "../types";

export function hasErrors(diagnostics: Diagnostic[]): boolean {
    return diagnostics.some((d) => d.severity === "error");
}

export function invalidCoreInput(message: string): Diagnostic {
    return { severity: "error", code: "invalid-core-input", message };
}

export function degenerateScale(): Diagnostic {
    return {
        severity: "error",
        code: "degenerate-scale",
        message:
            "Diese Kombination aus maximaler Punktzahl, Schrittweite und Knickpunkt ergibt keine gültige Notenskala."
    };
}

export function inputModeConflict(): Diagnostic {
    return {
        severity: "error",
        code: "input-mode-conflict",
        message:
            "Bitte entweder CSV-Import oder manuelle Eingabe verwenden. Eine Kombination ist nicht erlaubt."
    };
}

export function inputModeMismatch(selected: InputMode): Diagnostic {
    const other = selected === "csv" ? "der manuellen Tabelle" : "der CSV-Datei";
    const chosen = selected === "csv" ? "CSV-Import" : "manuelle Eingabe";
    return {
        severity: "error",
        code: "input-mode-mismatch",
        message: `Es ist ${chosen} ausgewählt, die Daten stehen aber in ${other}. Bitte den Eingabemodus wechseln.`
    };
}

export function pointsExceedMax(row: number, points: number, maxPoints: number): Diagnostic {
    return {
        severity: "error",
        code: "points-exceed-max",
        row,
        message: `Zeile ${row}: ${points} Punkte liegen über der Maximalpunktzahl von ${maxPoints}.`
    };
}

export function noValidStudents(): Diagnostic {
    return {
        severity: "error",
        code: "no-valid-students",
        message: "Keine gültigen Schülerdaten gefunden."
    };
}

export function rowTooFewColumns(row: number): Diagnostic {
    return {
        severity: "warning",
        code: "row-too-few-columns",
        row,
        message: `Zeile ${row}: übersprungen, weniger als zwei Spalten.`
    };
}

export function rowMissingName(row: number): Diagnostic {
    return {
        severity: "warning",
        code: "row-missing-name",
        row,
        message: `Zeile ${row}: übersprungen, Name fehlt.`
    };
}

export function rowMissingPoints(row: number): Diagnostic {
    return {
        severity: "warning",
        code: "row-missing-points",
        row,
        message: `Zeile ${row}: übersprungen, Punkte fehlen.`
    };
}

export function rowUnparsablePoints(row: number, raw: string): Diagnostic {
    return {
        severity: "warning",
        code: "row-unparsable-points",
        row,
        message: `Zeile ${row}: übersprungen, „${raw}" ist keine Punktzahl.`
    };
}

export function rowNegativePoints(row: number, points: number): Diagnostic {
    return {
        severity: "warning",
        code: "row-negative-points",
        row,
        message: `Zeile ${row}: übersprungen, ${points} Punkte sind negativ.`
    };
}

export function rowLimitReached(limit: number): Diagnostic {
    return {
        severity: "warning",
        code: "row-limit-reached",
        message: `Nur die ersten ${limit} Zeilen wurden gelesen, weitere wurden ignoriert.`
    };
}
