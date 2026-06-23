import { useState, useCallback } from "react";
import { invoke } from "@tauri-apps/api/core";
import { usePOS } from "./PosContext";
import { CashPayment } from "./CashPayment";
import { CardPayment } from "./CardPayment";
import { SplitPayment } from "./SplitPayment";
import type { PaymentType, DocumentType } from "./types";

export function PaymentModal() {
  const { state, dispatch } = usePOS();
  const { cartTotals, cart, customer } = state;
  const [method, setMethod] = useState<PaymentType | null>(null);
  const [docType, setDocType] = useState<DocumentType>(
    customer ? "factura" : "ticket",
  );
  const [processing, setProcessing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showDocTypeWarning, setShowDocTypeWarning] = useState(false);

  const close = useCallback(() => {
    dispatch({ type: "CLOSE_PAYMENT_MODAL" });
  }, [dispatch]);

  const handleBackdrop = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      if (e.target === e.currentTarget) close();
    },
    [close],
  );

  // Build common payment payload
  const buildPaymentPayload = useCallback(
    (
      payments: { metodo_pago: string; amount: number }[],
    ) => {
      const payload: Record<string, unknown> = {
        items: cart.map((item) => ({
          product_id: item.product.id,
          codigo: item.product.codigo,
          nombre: item.product.nombre,
          cantidad: item.quantity,
          precio_unitario: item.unit_price,
          discount_percent: item.discount_percent,
          tipo_iva_porcentaje: item.product.tipo_iva_porcentaje ?? 0,
          unidad_medida: item.unidad_medida,
        })),
        payments,
      };

      // Add optional fields — backend may ignore extras if not supported
      payload.document_type = docType;
      if (customer) {
        payload.customer_id = customer.id;
        payload.customer_nombre = customer.nombre;
        payload.customer_nif = customer.nif;
        payload.customer_descuento = customer.descuento;
      }

      return payload;
    },
    [cart, customer, docType],
  );

  const executePayment = useCallback(
    async (payments: { metodo_pago: string; amount: number }[]) => {
      setProcessing(true);
      setError(null);

      try {
        const result = await invoke<{ sale_id: string }>(
          "registrar_cobro_pago",
          buildPaymentPayload(payments),
        );

        const receiptData = await invoke<string>("generate_receipt_pdf", {
          saleId: result.sale_id,
        });

        dispatch({ type: "SET_RECEIPT_DATA", payload: receiptData });
        dispatch({ type: "OPEN_RECEIPT_MODAL" });
      } catch (err) {
        console.error("Payment failed:", err);
        // Try fallback without optional fields
        try {
          const fallbackPayload = {
            items: cart.map((item) => ({
              product_id: item.product.id,
              codigo: item.product.codigo,
              nombre: item.product.nombre,
              cantidad: item.quantity,
              precio_unitario: item.unit_price,
              discount_percent: item.discount_percent,
              tipo_iva_porcentaje: item.product.tipo_iva_porcentaje ?? 0,
            })),
            payments,
          };
          const result = await invoke<{ sale_id: string }>(
            "registrar_cobro_pago",
            fallbackPayload,
          );

          const receiptData = await invoke<string>("generate_receipt_pdf", {
            saleId: result.sale_id,
          });

          dispatch({ type: "SET_RECEIPT_DATA", payload: receiptData });
          dispatch({ type: "OPEN_RECEIPT_MODAL" });
        } catch (fallbackErr) {
          setError(
            "Error al procesar el pago. Verifica la conexion e intenta nuevamente.",
          );
          setProcessing(false);
        }
      }
    },
    [cart, dispatch, buildPaymentPayload],
  );

  // Payment method handlers
  const handleCashConfirm = useCallback(
    async (tender: number) => {
      await executePayment([{ metodo_pago: "CASH", amount: tender }]);
    },
    [executePayment],
  );

  const handleCardConfirm = useCallback(async () => {
    await executePayment([{ metodo_pago: "CARD", amount: cartTotals.total }]);
  }, [executePayment, cartTotals.total]);

  const handleBizumConfirm = useCallback(async () => {
    await executePayment([{ metodo_pago: "BIZUM", amount: cartTotals.total }]);
  }, [executePayment, cartTotals.total]);

  const handleSplitConfirm = useCallback(
    async (cashAmount: number, cardAmount: number, bizumAmount: number) => {
      const payments: { metodo_pago: string; amount: number }[] = [];
      if (cashAmount > 0) payments.push({ metodo_pago: "CASH", amount: cashAmount });
      if (cardAmount > 0) payments.push({ metodo_pago: "CARD", amount: cardAmount });
      if (bizumAmount > 0) payments.push({ metodo_pago: "BIZUM", amount: bizumAmount });
      await executePayment(payments);
    },
    [executePayment],
  );

  // Document type toggle
  const handleDocTypeChange = useCallback(
    (newType: DocumentType) => {
      if (customer && newType === "ticket") {
        setShowDocTypeWarning(true);
      } else {
        setShowDocTypeWarning(false);
        setDocType(newType);
      }
    },
    [customer],
  );

  const confirmDocTypeOverride = useCallback(() => {
    setDocType("ticket");
    setShowDocTypeWarning(false);
  }, []);

  return (
    <div className="modal-overlay" onClick={handleBackdrop}>
      <div className="modal-panel payment-modal animate-fade-in">
        <div className="modal-header">
          <h2 className="modal-title">Cobrar</h2>
          <button className="modal-close" onClick={close}>
            &times;
          </button>
        </div>

        {/* Total display */}
        <div className="payment-total-display">
          <span className="payment-total-label">Total a cobrar</span>
          <span className="payment-total-value">
            {cartTotals.total.toFixed(2)} €
          </span>
        </div>

        {/* Customer info + discount summary */}
        {customer && (
          <div className="payment-customer-info">
            <span className="payment-customer-name">{customer.nombre}</span>
            {customer.nif && (
              <span className="payment-customer-nif">{customer.nif}</span>
            )}
            {customer.descuento > 0 && (
              <span className="payment-customer-discount">
                Descuento: -{customer.descuento}%
              </span>
            )}
          </div>
        )}

        {/* Document type toggle */}
        <div className="payment-doctype-toggle">
          <span className="doctype-label">Tipo de documento:</span>
          <div className="doctype-buttons">
            <button
              className={`doctype-btn ${docType === "ticket" ? "doctype-active" : ""}`}
              onClick={() => handleDocTypeChange("ticket")}
              disabled={processing}
            >
              Ticket
            </button>
            <button
              className={`doctype-btn ${docType === "factura" ? "doctype-active" : ""}`}
              onClick={() => handleDocTypeChange("factura")}
              disabled={processing}
            >
              Factura
            </button>
            <button
              className={`doctype-btn ${docType === "albarán" ? "doctype-active" : ""}`}
              onClick={() => handleDocTypeChange("albarán")}
              disabled={processing}
            >
              Albar&aacute;n
            </button>
          </div>
        </div>

        {/* Document type override warning */}
        {showDocTypeWarning && (
          <div className="payment-warning">
            <p>El cliente no aparecera en el documento</p>
            <div className="payment-warning-actions">
              <button
                className="payment-btn-danger"
                onClick={confirmDocTypeOverride}
              >
                Confirmar Ticket
              </button>
              <button
                className="payment-btn-secondary"
                onClick={() => setShowDocTypeWarning(false)}
              >
                Cancelar
              </button>
            </div>
          </div>
        )}

        {/* Error display */}
        {error && (
          <div className="payment-error">
            <p>{error}</p>
            <button
              className="payment-btn-primary"
              onClick={() => setError(null)}
            >
              Reintentar
            </button>
          </div>
        )}

        {/* Processing overlay */}
        {processing && (
          <div className="payment-processing">
            <div className="processing-spinner" />
            <span>Procesando pago...</span>
          </div>
        )}

        {/* Method selector / payment form */}
        {!processing && !error && !showDocTypeWarning && (
          <>
            {!method ? (
              <div className="payment-method-selector">
                <button
                  className="payment-method-btn cash"
                  onClick={() => setMethod("cash")}
                >
                  <span className="method-icon method-icon-cash" />
                  <span className="method-name">Efectivo</span>
                  <span className="method-desc">
                    Pago en efectivo con vuelto
                  </span>
                </button>
                <button
                  className="payment-method-btn card"
                  onClick={() => setMethod("card")}
                >
                  <span className="method-icon method-icon-card" />
                  <span className="method-name">Tarjeta</span>
                  <span className="method-desc">
                    Debito / Credito - monto exacto
                  </span>
                </button>
                <button
                  className="payment-method-btn bizum"
                  onClick={() => {
                    setMethod("bizum");
                    handleBizumConfirm();
                  }}
                >
                  <span className="method-icon method-icon-bizum" />
                  <span className="method-name">Bizum</span>
                  <span className="method-desc">
                    Pago por Bizum - monto exacto
                  </span>
                </button>
                <button
                  className="payment-method-btn split"
                  onClick={() => setMethod("split")}
                >
                  <span className="method-icon method-icon-split" />
                  <span className="method-name">Dividido</span>
                  <span className="method-desc">
                    Efectivo + Tarjeta + Bizum combinado
                  </span>
                </button>
              </div>
            ) : (
              <>
                {method === "cash" && (
                  <CashPayment
                    total={cartTotals.total}
                    onConfirm={handleCashConfirm}
                    onCancel={() => setMethod(null)}
                  />
                )}
                {method === "card" && (
                  <CardPayment
                    total={cartTotals.total}
                    onConfirm={handleCardConfirm}
                    onCancel={() => setMethod(null)}
                  />
                )}
                {method === "split" && (
                  <SplitPayment
                    total={cartTotals.total}
                    ivaBreakdown={cartTotals.iva_breakdown}
                    onConfirm={handleSplitConfirm}
                    onCancel={() => setMethod(null)}
                  />
                )}
              </>
            )}
          </>
        )}
      </div>
    </div>
  );
}
