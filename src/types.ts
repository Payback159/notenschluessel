export type Severity = "error" | "warning";

export type DiagnosticCode =
    | "invalid-core-input"
    | "degenerate-scale"
    | "input-mode-conflict"
    | "input-mode-mismatch"
    | "points-exceed-max"
    | "no-valid-students"
    | "row-too-few-columns"
    | "row-missing-name"
    | "row-missing-points"
    | "row-unparsable-points"
    | "row-negative-points"
    | "row-limit-reached";

export interface Diagnostic {
    severity: Severity;
    code: DiagnosticCode;
    /** 1-based line number of the source file, including the header row. */
    row?: number;
    message: string;
}

export type InputMode = "csv" | "manual";

export interface Student {
    name: string;
    points: number;
    grade?: number;
    /** 1-based source line, set by the parsers. Absent for hand-built test data. */
    sourceRow?: number;
}

export interface GradeBound {
    grade: number;
    lowerBound: number;
    upperBound: number;
}

export interface ManualEntry {
    name: string;
    points: string;
}

export interface ValidationResult {
    valid: boolean;
    errors: string[];
}

export interface GradeBoundsValidationResult {
    valid: boolean;
    reason: string;
}

export interface CSVParseResult {
    students: Student[];
    diagnostics: Diagnostic[];
}

export interface ManualParseResult {
    students: Student[];
    errors: string[];
}