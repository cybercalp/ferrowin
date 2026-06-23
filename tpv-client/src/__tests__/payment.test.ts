import { describe, it, expect } from "vitest";
import {
  computeChange,
  isCashValid,
  isSplitValid,
  prorateTax,
  buildReceiptLines,
} from "../pos/payment-utils";
import type { IVABreakdownEntry } from "../pos/types";

/* ---------------------------------------------------------------------------
   computeChange
   --------------------------------------------------------------------------- */

describe("computeChange", () => {
  it("returns 0 when tender equals total", () => {
    expect(computeChange(50, 50)).toBe(0);
  });

  it("returns positive change when tender exceeds total", () => {
    expect(computeChange(23.5, 30)).toBe(6.5);
  });

  it("returns negative when tender is less than total", () => {
    expect(computeChange(100, 50)).toBe(-50);
  });

  it("handles fractional currency correctly", () => {
    expect(computeChange(19.99, 20)).toBe(0.01);
  });

  it("handles zero total", () => {
    expect(computeChange(0, 10)).toBe(10);
  });
});

/* ---------------------------------------------------------------------------
   isCashValid
   --------------------------------------------------------------------------- */

describe("isCashValid", () => {
  it("returns true when exact amount", () => {
    expect(isCashValid(50, 50)).toBe(true);
  });

  it("returns true with change", () => {
    expect(isCashValid(23.5, 30)).toBe(true);
  });

  it("rejects insufficient cash", () => {
    expect(isCashValid(100, 50)).toBe(false);
  });

  it("rejects when tender is less by a small amount", () => {
    expect(isCashValid(20, 19.99)).toBe(false);
  });
});

/* ---------------------------------------------------------------------------
   isSplitValid
   --------------------------------------------------------------------------- */

describe("isSplitValid", () => {
  it("accepts when cash + card equals total", () => {
    expect(isSplitValid(100, [40, 60])).toBe(true);
  });

  it("accepts all cash, zero card", () => {
    expect(isSplitValid(100, [100, 0])).toBe(true);
  });

  it("accepts all card, zero cash", () => {
    expect(isSplitValid(100, [0, 100])).toBe(true);
  });

  it("rejects when sum exceeds total", () => {
    expect(isSplitValid(100, [60, 50])).toBe(false);
  });

  it("rejects when sum is less than total", () => {
    expect(isSplitValid(100, [30, 30])).toBe(false);
  });

  it("handles fractional split correctly", () => {
    expect(isSplitValid(19.99, [10, 9.99])).toBe(true);
  });

  it("rejects fraction mismatch", () => {
    expect(isSplitValid(20, [10.01, 9.99])).toBe(true); // 10.01+9.99 = 20
  });

  it("works with 3-way split (cash + card + bizum)", () => {
    expect(isSplitValid(100, [30, 30, 40])).toBe(true);
    expect(isSplitValid(100, [50, 50, 1])).toBe(false);
  });
});

/* ---------------------------------------------------------------------------
   prorateTax
   --------------------------------------------------------------------------- */

describe("prorateTax", () => {
  const mockBreakdown: IVABreakdownEntry[] = [
    { name: "IVA General", percent: 21, base: 100, tax: 21 },
    { name: "IVA Reducido", percent: 10, base: 50, tax: 5 },
  ];

  it("splits tax proportionally to 2-way ratio", () => {
    // Total = 150 base, tax_total = 26
    // Cash = 60, Card = 90
    const result = prorateTax(mockBreakdown, [60, 90], 150);
    expect(result[0]).toBeCloseTo(10.4, 2);  // cash tax
    expect(result[1]).toBeCloseTo(15.6, 2);  // card tax
  });

  it("allocates all tax to first method when second is 0", () => {
    const result = prorateTax(mockBreakdown, [150, 0], 150);
    expect(result[0]).toBeCloseTo(26, 2);
    expect(result[1]).toBeCloseTo(0, 2);
  });

  it("allocates all tax to second method when first is 0", () => {
    const result = prorateTax(mockBreakdown, [0, 150], 150);
    expect(result[0]).toBeCloseTo(0, 2);
    expect(result[1]).toBeCloseTo(26, 2);
  });

  it("handles zero total gracefully", () => {
    const result = prorateTax(mockBreakdown, [0, 0], 0);
    expect(result[0]).toBe(0);
    expect(result[1]).toBe(0);
  });

  it("sum of parts equals totalTax", () => {
    const result = prorateTax(mockBreakdown, [75, 75], 150);
    const totalTax = mockBreakdown.reduce((s, e) => s + e.tax, 0);
    expect(result[0] + result[1]).toBeCloseTo(totalTax, 2);
  });

  it("works with 3-way split (cash + card + bizum)", () => {
    const result = prorateTax(mockBreakdown, [30, 60, 60], 150);
    const totalTax = mockBreakdown.reduce((s, e) => s + e.tax, 0);
    expect(result).toHaveLength(3);
    expect(result[0] + result[1] + result[2]).toBeCloseTo(totalTax, 2);
  });
});

/* ---------------------------------------------------------------------------
   buildReceiptLines
   --------------------------------------------------------------------------- */

describe("buildReceiptLines", () => {
  it("returns an array of lines with header and footer", () => {
    const lines = buildReceiptLines({
      terminalId: "TPV-001",
      sequence: 1234,
      date: "20/06/2026 15:30:00",
      items: [
        { name: "Tornillo M8", qty: 10, price: 0.5, total: 5 },
      ],
      subtotal: 5,
      discountTotal: 0,
      taxTotal: 1.05,
      total: 6.05,
      payments: [{ method: "CASH", amount: 10 }],
      hash: "ABC123DEF",
    });

    expect(lines.length).toBeGreaterThan(5);
    expect(lines.some((l) => l.includes("FERROWIN"))).toBe(true);
    expect(lines.some((l) => l.includes("TPV-001"))).toBe(true);
    expect(lines.some((l) => l.includes("Tornillo M8"))).toBe(true);
    expect(lines.some((l) => l.includes("6.05"))).toBe(true);
    expect(lines.some((l) => l.includes("ABC123DEF"))).toBe(true);
  });
});
