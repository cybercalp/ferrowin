import { useState, useCallback } from "react";
import { computeChange, isCashValid } from "./payment-utils";

const QUICK_AMOUNTS = [5, 10, 20, 50];

interface CashPaymentProps {
  total: number;
  onConfirm: (tender: number) => void;
  onCancel: () => void;
}

export function CashPayment({ total, onConfirm, onCancel }: CashPaymentProps) {
  const [tender, setTender] = useState<string>("");
  const [customMode, setCustomMode] = useState(false);

  const tenderNum = parseFloat(tender) || 0;
  const change = computeChange(total, tenderNum);
  const valid = isCashValid(total, tenderNum);

  const handleQuickAmount = useCallback(
    (amount: number) => {
      if (amount < total) {
        // If quick amount is less than total, set it and focus custom
        setTender(amount.toFixed(2));
        setCustomMode(true);
      } else {
        setTender(amount.toFixed(2));
        setCustomMode(false);
      }
    },
    [total],
  );

  const handleExact = useCallback(() => {
    setTender(total.toFixed(2));
    setCustomMode(false);
  }, [total]);

  const handleTenderChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const raw = e.target.value.replace(/[^0-9.,]/g, "").replace(",", ".");
      if (/^\d*\.?\d{0,2}$/.test(raw)) {
        setTender(raw);
      }
    },
    [],
  );

  return (
    <div className="cash-payment">
      <h3 className="payment-subtitle">Pago en Efectivo</h3>

      <div className="cash-tender-group">
        <label className="cash-label">Monto recibido</label>
        <div className="cash-input-wrapper">
          <input
            type="text"
            inputMode="decimal"
            className="cash-input"
            value={tender}
            onChange={handleTenderChange}
            placeholder="0.00"
            autoFocus
          />
          <span className="cash-currency">€</span>
        </div>
      </div>

      <div className="quick-amounts">
        {QUICK_AMOUNTS.map((amount) => (
          <button
            key={amount}
            className="quick-amount-btn"
            onClick={() => handleQuickAmount(amount)}
          >
            {amount} €
          </button>
        ))}
        <button className="quick-amount-btn quick-exact" onClick={handleExact}>
          Exacto
        </button>
        <button
          className={`quick-amount-btn ${customMode ? "quick-active" : ""}`}
          onClick={() => setCustomMode(!customMode)}
        >
          Otro
        </button>
      </div>

      {customMode && (
        <div className="custom-tender">
          <label className="cash-label">Otro monto</label>
          <div className="cash-input-wrapper">
            <input
              type="text"
              inputMode="decimal"
              className="cash-input"
              value={tender}
              onChange={handleTenderChange}
              placeholder="0.00"
            />
            <span className="cash-currency">€</span>
          </div>
        </div>
      )}

      <div className="change-display">
        <span className="change-label">Vuelto</span>
        <span
          className={`change-value ${change < 0 ? "change-negative" : "change-positive"}`}
        >
          {change < 0
            ? `Faltan ${Math.abs(change).toFixed(2)} €`
            : `${change.toFixed(2)} €`}
        </span>
      </div>

      <div className="payment-actions">
        <button className="payment-btn-secondary" onClick={onCancel}>
          Cancelar
        </button>
        <button
          className="payment-btn-primary"
          disabled={!valid || tenderNum <= 0}
          onClick={() => onConfirm(tenderNum)}
        >
          Cobrar {tenderNum.toFixed(2)} €
        </button>
      </div>
    </div>
  );
}
