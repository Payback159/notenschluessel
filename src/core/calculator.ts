import { GradeBound, GradeBoundsValidationResult } from "../types";

/**
 * Calculates grade boundaries based on the Austrian 1–5 scale.
 * 
 * The algorithm uses a "break point" (passing threshold), defined by `breakPointPercent`.
 * The area between the break point and the maximum points is divided into four equal segments,
 * each assigned to a grade. All boundaries are rounded to the nearest `minPoints` interval.
 * 
 * @param maxPoints - The maximum possible score (e.g., 100).
 * @param minPoints - The rounding interval for the grade boundaries.
 * @param breakPointPercent - Percentage of the maximum score where the passing threshold (Grade 4) lies.
 * @returns An array of `GradeBound` objects defining the range for each grade.
 */
export function calculateGradeBounds(
    maxPoints: number,
    minPoints: number,
    breakPointPercent: number
): GradeBound[] {
    const breakPointAbsolute = maxPoints * (breakPointPercent / 100.0);
    const segment = (maxPoints - breakPointAbsolute) / 4.0;

    let lowerBound4 = breakPointAbsolute;
    let lowerBound3 = breakPointAbsolute + segment;
    let lowerBound2 = breakPointAbsolute + 2 * segment;
    let lowerBound1 = breakPointAbsolute + 3 * segment;
    const lowerBound5 = 0.0;

    lowerBound1 = Math.round(lowerBound1 / minPoints) * minPoints;
    lowerBound2 = Math.round(lowerBound2 / minPoints) * minPoints;
    lowerBound3 = Math.round(lowerBound3 / minPoints) * minPoints;
    lowerBound4 = Math.round(lowerBound4 / minPoints) * minPoints;

    lowerBound4 = Math.max(0, lowerBound4);
    lowerBound3 = Math.max(lowerBound4, lowerBound3);
    lowerBound2 = Math.max(lowerBound3, lowerBound2);
    lowerBound1 = Math.max(lowerBound2, lowerBound1);

    return [
        { grade: 1, lowerBound: lowerBound1, upperBound: maxPoints },
        { grade: 2, lowerBound: lowerBound2, upperBound: lowerBound1 - minPoints },
        { grade: 3, lowerBound: lowerBound3, upperBound: lowerBound2 - minPoints },
        { grade: 4, lowerBound: lowerBound4, upperBound: lowerBound3 - minPoints },
        { grade: 5, lowerBound: lowerBound5, upperBound: lowerBound4 - minPoints }
    ];
}

/**
 * Validates the integrity of an array of grade boundaries.
 * 
 * Checks if all five grades are present, whether ranges are within bounds,
 * and ensures there are no overlaps or inverted intervals between grades.
 * 
 * @param gradeBounds - The array of `GradeBound` objects to validate.
 * @returns An object containing the validation status (`valid`) and an error message (`reason`), if invalid.
 */
export function validateGradeBounds(gradeBounds: GradeBound[]): GradeBoundsValidationResult {
    if (gradeBounds.length !== 5) {
        return { valid: false, reason: "insufficient grade bounds" };
    }

    for (const bound of gradeBounds) {
        if (bound.upperBound < bound.lowerBound) {
            return { valid: false, reason: `grade ${bound.grade} has inverted range` };
        }
    }

    for (let i = 1; i < gradeBounds.length; i++) {
        const current = gradeBounds[i];
        const previous = gradeBounds[i - 1];
        if (!current || !previous) {
            return { valid: false, reason: "insufficient grade bounds" };
        }

        if (current.upperBound >= previous.lowerBound) {
            return {
                valid: false,
                reason: `grade ${current.grade} overlaps with grade ${previous.grade}`
            };
        }
    }

    return { valid: true, reason: "" };
}