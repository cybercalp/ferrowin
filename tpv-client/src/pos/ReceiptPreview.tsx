import { useMemo, useCallback } from "react";
import { usePOS } from "./PosContext";
import { buildReceiptLines } from "./payment-utils";

export function ReceiptPreview() {
  const { state, dispatch } = usePOS();
  const { cartTotals, cart, lastReceiptData } = state;

  const close = useCallback(() => {
    dispatch({ type: "CLOSE_RECEIPT_MODAL" });
    dispatch({ type: "CLEAR_CART" });
  }, [dispatch]);

  const handleReprint = useCallback(async () => {
    if (!lastReceiptData) return;
    try {
      const { invoke } = await import("@tauri-apps/api/core");
      await invoke<string>("generate_receipt_pdf", {
        saleId: lastReceiptData, // fallback: reuses stored data
      });
    } catch (err) {
      console.error("Reprint failed:", err);
    }
  }, [lastReceiptData]);

  const receiptLines = useMemo(() => {
    return buildReceiptLines({
      terminalId: "TPV-001",
      sequence: Math.floor(Math.random() * 9000) + 1000,
      date: new Date().toLocaleString("es-AR"),
      items: cart.map((item) => ({
        name: item.product.nombre,
        qty: item.quantity,
        price: item.product.precio_venta,
        total: item.product.precio_venta * item.quantity,
      })),
      subtotal: cartTotals.subtotal,
      discountTotal: cartTotals.discount_total,
      taxTotal: cartTotals.tax_total,
      total: cartTotals.total,
      payments: [{ method: "CASH", amount: cartTotals.total }],
      hash: `HASH-${Date.now().toString(16).toUpperCase()}`,
    });
  }, [cart, cartTotals, lastReceiptData]);

  return (
    <div className="modal-overlay">
      <div className="modal-panel receipt-modal animate-fade-in">
        <div className="modal-header">
          <h2 className="modal-title">Comprobante</h2>
          <button className="modal-close" onClick={close}>
            ✕
          </button>
        </div>

        <div className="receipt-content">
          <pre className="receipt-text">{receiptLines.join("\n")}</pre>
        </div>

        <div className="receipt-actions">
          <button className="payment-btn-secondary" onClick={close}>
            Cerrar
          </button>
          <button className="payment-btn-primary" onClick={handleReprint}>
            Re-imprimir (Reprint)
          </button>
        </div>
      </div>
    </div>
  );
}
