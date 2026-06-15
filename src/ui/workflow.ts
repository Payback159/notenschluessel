import { calculateGradeBounds, validateGradeBounds } from "../core/calculator";
import { calculateAverageGrade, processStudents } from "../core/grading";
import { validateCoreInputs, validateInputModeExclusivity } from "../core/validation";
import { parseCSVContent } from "../parsers/csvParser";
import { hasNonEmptyManualEntries, parseManualEntries } from "../parsers/manualParser";
import { GradeBound, InputMode, ManualEntry, Student } from "../types";

export interface WorkflowInput {
    maxPoints: number;
    minPoints: number;
    breakPointPercent: number;
    inputMode: InputMode;
    csvContent?: string;
    manualEntries: ManualEntry[];
}

export interface WorkflowResult {
    ok: boolean;
    errors: string[];
    gradeBounds: GradeBound[];
    students: Student[];
    averageGrade: number;
}

export function runCalculationWorkflow(input: WorkflowInput): WorkflowResult {
    const validation = validateCoreInputs(input.maxPoints, input.minPoints, input.breakPointPercent);
    if (!validation.valid) {
        return emptyResult(validation.errors);
    }

    const gradeBounds = calculateGradeBounds(input.maxPoints, input.minPoints, input.breakPointPercent);
    const boundsValidation = validateGradeBounds(gradeBounds);
    if (!boundsValidation.valid) {
        return emptyResult([
            "Diese Kombination aus maximaler Punktzahl, Schrittweite und Knickpunkt ergibt keine gueltige Notenskala."
        ]);
    }

    const csvProvided = (input.csvContent ?? "").trim() !== "";
    const manualProvided = hasNonEmptyManualEntries(input.manualEntries);
    const modeValidation = validateInputModeExclusivity(input.inputMode, csvProvided, manualProvided);
    if (!modeValidation.valid) {
        return emptyResult(modeValidation.errors);
    }

    let students: Student[] = [];
    const errors: string[] = [];

    if (input.inputMode === "csv" && csvProvided) {
        const result = parseCSVContent(input.csvContent ?? "");
        students = result.students;
        errors.push(...result.errors);
    }

    if (input.inputMode === "manual" && manualProvided) {
        const result = parseManualEntries(input.manualEntries);
        students = result.students;
        errors.push(...result.errors);
    }

    if (errors.length > 0) {
        return emptyResult(errors);
    }

    const gradedStudents = students.length > 0 ? processStudents(students, gradeBounds) : [];
    const averageGrade = gradedStudents.length > 0 ? calculateAverageGrade(gradedStudents) : 0;

    return {
        ok: true,
        errors: [],
        gradeBounds,
        students: gradedStudents,
        averageGrade
    };
}

function emptyResult(errors: string[]): WorkflowResult {
    return {
        ok: false,
        errors,
        gradeBounds: [],
        students: [],
        averageGrade: 0
    };
}