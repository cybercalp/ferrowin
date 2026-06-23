import { describe, it, expect } from "vitest";
import { computeCartTotals } from "../pos/types";
import type { CartItem, POSProduct } from "../pos/types";

function makeProduct(overrides: Partial<POSProduct> = {}): POSProduct {
  return {
    id: "p1",
    codigo: "001",
    nombre: "Producto de prueba",
    precio_venta: 100,
    stock: 10,
    familia_nombre: null,
    tipo_iva_nombre: "IVA General",
    tipo_iva_porcentaje: 21,
    imagen_url: null,
    unidad_medida: "ud",
    ...overrides,
  };
}

function ci(overrides: Partial<CartItem> = {}): CartItem {
  return {
    product: makeProduct(),
    quantity: 1,
    discount_percent: 0,
    unidad_medida: "ud",
    unit_price: 100,
    ...overrides,
  };
}

describe("computeCartTotals", () => {
  it("returns zero totals for an empty cart", () => {
    const result = computeCartTotals([]);
    expect(result.subtotal).toBe(0);
    expect(result.discount_total).toBe(0);
    expect(result.tax_total).toBe(0);
    expect(result.total).toBe(0);
    expect(result.iva_breakdown).toEqual([]);
  });

  it("adds a single item with correct IVA", () => {
    const items: CartItem[] = [
      ci({ product: makeProduct({ precio_venta: 100 }), quantity: 2, unit_price: 100 }),
    ];
    const result = computeCartTotals(items);
    expect(result.subtotal).toBe(200);
    expect(result.tax_total).toBe(42); // 200 * 0.21
    expect(result.total).toBe(242); // 200 + 42
  });

  it("applies a percentage discount correctly", () => {
    const items: CartItem[] = [
      ci({ product: makeProduct({ precio_venta: 100 }), quantity: 2, discount_percent: 10, unit_price: 100 }),
    ];
    const result = computeCartTotals(items);
    expect(result.subtotal).toBe(200);
    expect(result.discount_total).toBe(20); // 200 * 0.10
    expect(result.tax_total).toBe(37.8); // 180 * 0.21
    expect(result.total).toBe(217.8); // 180 + 37.8
  });

  it("handles multiple items with different IVA rates", () => {
    const items: CartItem[] = [
      ci({ product: makeProduct({ id: "p1", precio_venta: 100, tipo_iva_nombre: "IVA General", tipo_iva_porcentaje: 21 }), quantity: 1, unit_price: 100 }),
      ci({ product: makeProduct({ id: "p2", precio_venta: 50, tipo_iva_nombre: "IVA Reducido", tipo_iva_porcentaje: 10 }), quantity: 2, unit_price: 50 }),
    ];
    const result = computeCartTotals(items);
    expect(result.subtotal).toBe(200); // 100 + 100
    expect(result.tax_total).toBe(31); // 21 + 10
    expect(result.total).toBe(231);

    expect(result.iva_breakdown).toHaveLength(2);

    const general = result.iva_breakdown.find((e) => e.name === "IVA General");
    expect(general).toBeDefined();
    expect(general!.base).toBe(100);
    expect(general!.tax).toBe(21);

    const reducido = result.iva_breakdown.find((e) => e.name === "IVA Reducido");
    expect(reducido).toBeDefined();
    expect(reducido!.base).toBe(100);
    expect(reducido!.tax).toBe(10);
  });

  it("handles items without IVA data", () => {
    const items: CartItem[] = [
      ci({ product: makeProduct({ precio_venta: 100, tipo_iva_nombre: null, tipo_iva_porcentaje: null }), unit_price: 100 }),
    ];
    const result = computeCartTotals(items);
    expect(result.subtotal).toBe(100);
    expect(result.tax_total).toBe(0);
    expect(result.total).toBe(100);
    expect(result.iva_breakdown[0].name).toBe("IVA 0%");
    expect(result.iva_breakdown[0].percent).toBe(0);
    expect(result.iva_breakdown[0].tax).toBe(0);
  });

  it("groups items with same IVA rate", () => {
    const items: CartItem[] = [
      ci({ product: makeProduct({ id: "p1", precio_venta: 50, tipo_iva_nombre: "IVA General", tipo_iva_porcentaje: 21 }), quantity: 2, unit_price: 50 }),
      ci({ product: makeProduct({ id: "p2", precio_venta: 30, tipo_iva_nombre: "IVA General", tipo_iva_porcentaje: 21 }), quantity: 3, unit_price: 30 }),
    ];
    const result = computeCartTotals(items);
    expect(result.iva_breakdown).toHaveLength(1);
    expect(result.iva_breakdown[0].base).toBe(190); // 100 + 90
    expect(result.iva_breakdown[0].tax).toBe(39.9); // 190 * 0.21
  });

  it("removing an item recalculates totals", () => {
    const items: CartItem[] = [
      ci({ product: makeProduct({ id: "p1", precio_venta: 100 }), unit_price: 100 }),
      ci({ product: makeProduct({ id: "p2", precio_venta: 50 }), quantity: 3, unit_price: 50 }),
    ];
    const result = computeCartTotals(items);
    expect(result.subtotal).toBe(250);
    expect(result.total).toBe(302.5); // 250 + 52.5
  });

  it("discount rounds correctly with fractional values", () => {
    const items: CartItem[] = [
      ci({ product: makeProduct({ precio_venta: 9.99 }), quantity: 3, discount_percent: 15, unit_price: 9.99 }),
    ];
    const result = computeCartTotals(items);
    // subtotal: 29.97
    // discount: 4.4955 -> 4.50 (rounded)
    // discounted: 25.47
    // tax: 5.3487 -> 5.35 (rounded)
    // total: 30.82
    expect(result.subtotal).toBe(29.97);
    expect(result.discount_total).toBe(4.5);
    expect(result.total).toBe(30.82);
  });
});
