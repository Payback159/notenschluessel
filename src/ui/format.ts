import { GradeBound } from "../types";

const MAX_DECIMALS = 3;

/**
 * Derives how many decimals a value needs from the configured step size.
 *
 * A step of 0.25 produces boundaries with two decimals; formatting those with a
 * fixed single decimal (as the old `toFixed(1)` did) showed 33.3 where 33.25 was
 * meant, so the printed scale disagreed with the calculation.
 */
export function decimalsFor(minPoints: number): number {
    if (!Number.isFinite(minPoints) || minPoints <= 0) {
        return 0;
    }

    const text = String(minPoints);
    if (text.includes("e") || text.includes("E")) {
        return MAX_DECIMALS;
    }

    const fraction = text.split(".")[1];
    if (fraction === undefined) {
        return 0;
    }

    return Math.min(fraction.length, MAX_DECIMALS);
}

/** Formats a number German-style, with a decimal comma and no trailing zeros. */
export function formatNumberDe(value: number, decimals: number): string {
    const fixed = value.toFixed(decimals);
    const trimmed = decimals > 0 ? fixed.replace(/\.?0+$/, "") : fixed;
    return trimmed.replace(".", ",");
}

export function formatPoints(value: number, minPoints: number): string {
    return formatNumberDe(value, decimalsFor(minPoints));
}

export function formatRange(bound: GradeBound, minPoints: number): string {
    return `${formatPoints(bound.lowerBound, minPoints)} - ${formatPoints(bound.upperBound, minPoints)}`;
}
