import { useState, useCallback } from "react";

interface CardPaymentProps {
  total: number;
  onConfirm: () => void;
  onCancel: () => void;
}

export function CardPayment({ total, onConfirm, onCancel }: CardPaymentProps) {
  const [processing, setProcessing] = useState(false);
  const [done, setDone] = useState(false);

  const handleConfirm = useCallback(() => {
    setProcessing(true);
    // Simulate TPV terminal processing (actual Tauri command handles real processing)
    setTimeout(() => {
      setProcessing(false);
      setDone(true);
      onConfirm();
    }, 1500);
  }, [onConfirm]);

  return (
    <div className="card-payment">
      <h3 className="payment-subtitle">Pago con Tarjeta</h3>

      <div className="card-total-display">
        <span className="card-total-label">Total a cobrar</span>
        <span className="card-total-value">{total.toFixed(2)} €</span>
      </div>

      {processing && (
        <div className="card-processing">
          <div className="processing-spinner" />
          <span>Procesando pago con tarjeta...</span>
        </div>
      )}

      {done && (
        <div className="card-success">
          <span>Pago aprobado</span>
        </div>
      )}

      <div className="payment-actions">
        <button
          className="payment-btn-secondary"
          onClick={onCancel}
          disabled={processing}
        >
          Cancelar
        </button>
        <button
          className="payment-btn-primary"
          onClick={handleConfirm}
          disabled={processing || done}
        >
          {processing ? "Procesando..." : "Cobrar con Tarjeta"}
        </button>
      </div>
    </div>
  );
}
