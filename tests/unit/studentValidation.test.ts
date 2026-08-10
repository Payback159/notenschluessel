import { describe, expect, it } from "vitest";
import { validateStudentPoints } from "../../src/core/studentValidation";

describe("validateStudentPoints", () => {
    it("accepts points up to and including the maximum", () => {
        const d = validateStudentPoints(
            [
                { name: "Alice", points: 100, sourceRow: 2 },
                { name: "Bob", points: 0, sourceRow: 3 }
            ],
            100
        );
        expect(d).toHaveLength(0);
    });

    it("rejects a typo above the maximum and names the row", () => {
        const d = validateStudentPoints([{ name: "Tippfehler", points: 855, sourceRow: 4 }], 100);
        expect(d).toHaveLength(1);
        expect(d[0]?.severity).toBe("error");
        expect(d[0]?.code).toBe("points-exceed-max");
        expect(d[0]?.row).toBe(4);
        expect(d[0]?.message).toContain("855");
    });

    it("collects every offending row so they can be fixed in one pass", () => {
        const d = validateStudentPoints(
            [
                { name: "A", points: 855, sourceRow: 2 },
                { name: "B", points: 50, sourceRow: 3 },
                { name: "C", points: 101, sourceRow: 4 }
            ],
            100
        );
        expect(d).toHaveLength(2);
        expect(d.map((x) => x.row)).toEqual([2, 4]);
    });

    it("falls back to row 0 when the source row is unknown", () => {
        const d = validateStudentPoints([{ name: "A", points: 200 }], 100);
        expect(d).toHaveLength(1);
        expect(d[0]?.row).toBe(0);
    });
});
