import { GradeBound, GradeBoundsValidationResult } from "../types";

/**
 * Berechnet die Notengrenzen basierend auf der österreichischen 1–5-Skala.
 * 
 * Der Algorithmus verwendet einen "Knickpunkt" (Bestehensschwelle), der durch `breakPointPercent` definiert wird.
 * Der Bereich zwischen dem Knickpunkt und den Maximalpunkten wird in vier gleich große Segmente unterteilt,
 * die jeweils einer Note zugeordnet werden. Alle Grenzen werden auf das nächste `minPoints`-Intervall gerundet.
 * 
 * @param maxPoints - Die maximale Punktzahl (z. B. 100).
 * @param minPoints - Das Intervall für die Rundung der Notengrenzen.
 * @param breakPointPercent - Prozentsatz der Maximalpunktzahl, an dem die Bestehensschwelle (Note 4) liegt.
 * @returns Ein Array von `GradeBound`-Objekten, die den Bereich für jede Note definieren.
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
 * Validiert die Integrität eines Arrays von Notengrenzen.
 * 
 * Prüft, ob alle fünf Noten vorhanden sind, ob die Bereiche innerhalb der Grenzen liegen
 * und ob es Überschneidungen oder invertierte Intervalle zwischen den Noten gibt.
 * 
 * @param gradeBounds - Das Array der zu validierenden `GradeBound`-Objekte.
 * @returns Ein Objekt mit dem Validierungsstatus (`valid`) und einer Fehlermeldung (`reason`), falls ungültig.
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