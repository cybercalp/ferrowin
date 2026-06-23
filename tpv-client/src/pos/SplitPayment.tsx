import { useState, useCallback, useMemo } from "react";
import { isSplitValid, prorateTax } from "./payment-utils";
import type { IVABreakdownEntry } from "./types";

interface SplitPaymentProps {
  total: number;
  ivaBreakdown: IVABreakdownEntry[];
  onConfirm: (cashAmount: number, cardAmount: number, bizumAmount: number) => void;
  onCancel: () => void;
}

export function SplitPayment({
  total,
  ivaBreakdown,
  onConfirm,
  onCancel,
}: SplitPaymentProps) {
  const [cashAmount, setCashAmount] = useState<string>("");
  const [cardAmount, setCardAmount] = useState<string>("");
  const [bizumAmount, setBizumAmount] = useState<string>("");

  const cashNum = parseFloat(cashAmount) || 0;
  const cardNum = parseFloat(cardAmount) || 0;
  const bizumNum = parseFloat(bizumAmount) || 0;
  const paid = Math.round((cashNum + cardNum + bizumNum) * 100) / 100;
  const remaining = Math.round((total - paid) * 100) / 100;
  const valid = isSplitValid(total, [cashNum, cardNum, bizumNum]);

  const taxProration = useMemo(
    () => prorateTax(ivaBreakdown, [cashNum, cardNum, bizumNum], total),
    [ivaBreakdown, cashNum, cardNum, bizumNum, total],
  );

  const handleAmountChange = useCallback(
    (
      setter: React.Dispatch<React.SetStateAction<string>>,
      e: React.ChangeEvent<HTMLInputElement>,
    ) => {
      const raw = e.target.value.replace(/[^0-9.,]/g, "").replace(",", ".");
      if (/^\d*\.?\d{0,2}$/.test(raw)) {
        setter(raw);
      }
    },
    [],
  );

  // Auto-fill remaining when one field is left empty after others are set
  const handleAutoFill = useCallback(
    (field: "cash" | "card" | "bizum") => {
      const otherSum = [cashNum, cardNum, bizumNum].reduce((a, b) => a + b, 0);
      const rest = Math.round((total - (otherSum || 0)) * 100) / 100;
      if (rest > 0) {
        const str = rest.toFixed(2);
        if (field === "cash") setCashAmount(str);
        else if (field === "card") setCardAmount(str);
        else setBizumAmount(str);
      }
    },
    [total, cashNum, cardNum, bizumNum],
  );

  return (
    <div className="split-payment">
      <h3 className="payment-subtitle">
        Pago Dividido (Efectivo + Tarjeta + Bizum)
      </h3>

      <div className="split-grid">
        {/* Cash */}
        <div className="split-col">
          <label className="split-label">Efectivo</label>
          <div className="cash-input-wrapper">
            <input
              type="text"
              inputMode="decimal"
              className="cash-input"
              value={cashAmount}
              onChange={(e) => handleAmountChange(setCashAmount, e)}
              placeholder="0.00"
              autoFocus
            />
            <span className="cash-currency">€</span>
          </div>
          <div className="split-tax-info">
            IVA: {taxProration[0].toFixed(2)} €
          </div>
          <button
            className="split-autofill-btn"
            onClick={() => handleAutoFill("cash")}
          >
            Restante
          </button>
        </div>

        {/* Card */}
        <div className="split-col">
          <label className="split-label">Tarjeta</label>
          <div className="cash-input-wrapper">
            <input
              type="text"
              inputMode="decimal"
              className="cash-input"
              value={cardAmount}
              onChange={(e) => handleAmountChange(setCardAmount, e)}
              placeholder="0.00"
            />
            <span className="cash-currency">€</span>
          </div>
          <div className="split-tax-info">
            IVA: {taxProration[1].toFixed(2)} €
          </div>
          <button
            className="split-autofill-btn"
            onClick={() => handleAutoFill("card")}
          >
            Restante
          </button>
        </div>

        {/* Bizum */}
        <div className="split-col">
          <label className="split-label">Bizum</label>
          <div className="cash-input-wrapper">
            <input
              type="text"
              inputMode="decimal"
              className="cash-input"
              value={bizumAmount}
              onChange={(e) => handleAmountChange(setBizumAmount, e)}
              placeholder="0.00"
            />
            <span className="cash-currency">€</span>
          </div>
          <div className="split-tax-info">
            IVA: {taxProration[2].toFixed(2)} €
          </div>
          <button
            className="split-autofill-btn"
            onClick={() => handleAutoFill("bizum")}
          >
            Restante
          </button>
        </div>
      </div>

      <div className="split-balance">
        <span className="split-balance-label">
          Total: {total.toFixed(2)} €
        </span>
        <span className="split-balance-label">
          Pagado: {paid.toFixed(2)} €
        </span>
        <span
          className={`split-balance-remaining ${remaining === 0 ? "split-ok" : remaining < 0 ? "split-over" : "split-under"}`}
        >
          {remaining === 0
            ? "Monto exacto"
            : remaining < 0
              ? `Excede en ${Math.abs(remaining).toFixed(2)} €`
              : `Faltan ${remaining.toFixed(2)} €`}
        </span>
      </div>

      <div className="payment-actions">
        <button className="payment-btn-secondary" onClick={onCancel}>
          Cancelar
        </button>
        <button
          className="payment-btn-primary"
          disabled={!valid || (cashNum <= 0 && cardNum <= 0 && bizumNum <= 0)}
          onClick={() => onConfirm(cashNum, cardNum, bizumNum)}
        >
          Cobrar {paid.toFixed(2)} €
        </button>
      </div>
    </div>
  );
}
