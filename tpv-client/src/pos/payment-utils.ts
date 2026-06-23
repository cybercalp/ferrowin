/* ---------------------------------------------------------------------------
   Pure payment math — reusable outside React, fully testable
   ------------------------------------------------------------------------- */

import type { IVABreakdownEntry } from "./types";

/**
 * Compute change = tender - total. Negative means insufficient.
 */
export function computeChange(total: number, tender: number): number {
  return Math.round((tender - total) * 100) / 100;
}

/**
 * A cash payment is valid when tender >= total.
 */
export function isCashValid(total: number, tender: number): boolean {
  return tender >= total;
}

/**
 * A split payment is valid when sum of amounts === total exactly.
 */
export function isSplitValid(
  total: number,
  amounts: number[],
): boolean {
  const sum = Math.round(amounts.reduce((a, b) => a + b, 0) * 100) / 100;
  return Math.abs(sum - total) < 0.005;
}

/**
 * Proportionally split IVA across payment methods in a split payment.
 * Returns the tax amount allocated to each method.
 */
export function prorateTax(
  ivaBreakdown: IVABreakdownEntry[],
  amounts: number[],
  total: number,
): number[] {
  if (total <= 0) return amounts.map(() => 0);
  const totalTax = ivaBreakdown.reduce((sum, e) => sum + e.tax, 0);
  const result = amounts.map((amt) => {
    const ratio = Math.min(1, Math.max(0, amt / total));
    return Math.round(totalTax * ratio * 100) / 100;
  });
  // Fix rounding drift
  const sum = result.reduce((a, b) => a + b, 0);
  const drift = Math.round((totalTax - sum) * 100) / 100;
  if (drift !== 0 && result.length > 0) {
    result[result.length - 1] = Math.round((result[result.length - 1] + drift) * 100) / 100;
  }
  return result;
}

/**
 * Build a human-readable receipt lines array from sale data.
 */
export function buildReceiptLines(params: {
  terminalId: string;
  sequence: number;
  date: string;
  items: { name: string; qty: number; price: number; total: number }[];
  subtotal: number;
  discountTotal: number;
  taxTotal: number;
  total: number;
  payments: { method: string; amount: number }[];
  hash: string;
}): string[] {
  const lines: string[] = [];
  const { terminalId, sequence, date, items, subtotal, discountTotal, taxTotal, total, payments, hash } = params;

  lines.push("\u2554\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2557");
  lines.push("\u2551      FERROWIN TPV - TIENDA       \u2551");
  lines.push("\u2560\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2563");
  lines.push(`  Terminal: ${terminalId}`);
  lines.push(`  Ticket #: ${sequence}`);
  lines.push(`  Fecha:    ${date}`);
  lines.push("\u2560\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2563");
  lines.push("");

  for (const item of items) {
    lines.push(`  ${item.name}`);
    lines.push(`    ${item.qty} x $${item.price.toFixed(2)}  =  $${item.total.toFixed(2)}`);
  }

  lines.push("");
  lines.push("\u2560\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2563");
  lines.push(`  Subtotal:          $${subtotal.toFixed(2)}`);
  if (discountTotal > 0) {
    lines.push(`  Descuento:        -$${discountTotal.toFixed(2)}`);
  }
  lines.push(`  IVA:               $${taxTotal.toFixed(2)}`);
  lines.push(`  TOTAL:             $${total.toFixed(2)}`);
  lines.push("");

  for (const pay of payments) {
    let label = pay.method === "CASH" ? "Efectivo" : pay.method === "CARD" ? "Tarjeta" : "Bizum";
    lines.push(`  ${label}: $${pay.amount.toFixed(2)}`);
  }

  lines.push("");
  lines.push("\u2560\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2563");
  lines.push(`  Hash: ${hash.slice(0, 20)}...`);
  lines.push("\u255a\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u255d");

  return lines;
}
