export type InputMode = "csv" | "manual";

export interface Student {
    name: string;
    points: number;
    grade?: number;
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
    skippedRows: number;
    errors: string[];
}

export interface ManualParseResult {
    students: Student[];
    errors: string[];
}