/* ---------------------------------------------------------------------------
   Types for TPV Tienda POS — Ferreteria Workflow
   ------------------------------------------------------------------------- */

export type UnidadMedida = "ud" | "m" | "kg";
export type PaymentMethod = "cash" | "card" | "bizum" | "split";
export type DocumentType = "ticket" | "factura" | "albarán";

export interface POSProduct {
  id: string;
  codigo: string;
  nombre: string;
  precio_venta: number;
  stock: number | null;
  familia_nombre: string | null;
  tipo_iva_nombre: string | null;
  tipo_iva_porcentaje: number | null;
  imagen_url: string | null;
  // Ferreteria: unit of measure
  unidad_medida: UnidadMedida;
}

export interface CustomerInfo {
  id: string;
  nombre: string;
  nif: string;
  direccion: string;
  descuento: number; // percentage, e.g. 5 = 5%
}

export interface CartItem {
  product: POSProduct;
  quantity: number;
  discount_percent: number;
  unidad_medida: UnidadMedida;
  unit_price: number; // precio_venta after customer discount applied at add time
}

export interface IVABreakdownEntry {
  name: string;
  percent: number;
  base: number;
  tax: number;
}

export interface CartTotals {
  subtotal: number;
  discount_total: number;
  tax_total: number;
  total: number;
  iva_breakdown: IVABreakdownEntry[];
}

export type PaymentType = "cash" | "card" | "bizum" | "split";

export interface PaymentInfo {
  method: "cash" | "card" | "bizum" | "split";
  cash_amount?: number;
  card_amount?: number;
  bizum_amount?: number;
  change?: number;
}

export interface TerminalHealth {
  terminal_id: string;
  db_size_bytes: number;
  pending_sales_count: number;
  pending_closures_count: number;
  online: boolean;
  app_version: string;
}

export interface OfflineSalePayment {
  id: string;
  sale_id: string;
  metodo_pago: string;
  amount: number;
  created_at: string;
}

export interface OfflineSaleItem {
  id: string;
  sale_id: string;
  product_id: string;
  codigo: string;
  nombre: string;
  cantidad: number;
  precio_unitario: number;
  discount_percent: number;
  tipo_iva_porcentaje: number;
}

export interface OfflineSale {
  id: string;
  terminal_serial: string;
  sequence: number;
  subtotal: number;
  tax_total: number;
  discount_total: number;
  total: number;
  status: string;
  firma_registro: string;
  hash_anterior: string;
  created_at: string;
  sync_status: string;
  items: OfflineSaleItem[];
  payments: OfflineSalePayment[];
}

/* ---------------------------------------------------------------------------
   Pure cart math — uses unit_price from CartItem (already discounted)
   ------------------------------------------------------------------------- */
export function computeCartTotals(items: CartItem[]): CartTotals {
  let subtotal = 0;
  let discount_total = 0;
  const taxGroups = new Map<
    string,
    { name: string; percent: number; base: number }
  >();

  for (const item of items) {
    const lineBase = item.unit_price * item.quantity;
    const lineDiscount = lineBase * (item.discount_percent / 100);
    const lineDiscounted = lineBase - lineDiscount;

    subtotal += lineBase;
    discount_total += lineDiscount;

    const pct = item.product.tipo_iva_porcentaje ?? 0;
    const name = item.product.tipo_iva_nombre ?? `IVA ${pct}%`;
    const key = name;

    if (!taxGroups.has(key)) {
      taxGroups.set(key, { name, percent: pct, base: 0 });
    }
    taxGroups.get(key)!.base += lineDiscounted;
  }

  let tax_total = 0;
  const iva_breakdown: IVABreakdownEntry[] = [];

  for (const group of taxGroups.values()) {
    const tax = group.base * (group.percent / 100);
    tax_total += tax;
    iva_breakdown.push({
      name: group.name,
      percent: group.percent,
      base: Math.round(group.base * 100) / 100,
      tax: Math.round(tax * 100) / 100,
    });
  }

  return {
    subtotal: Math.round(subtotal * 100) / 100,
    discount_total: Math.round(discount_total * 100) / 100,
    tax_total: Math.round(tax_total * 100) / 100,
    total: Math.round((subtotal - discount_total + tax_total) * 100) / 100,
    iva_breakdown,
  };
}
