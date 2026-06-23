import { useState, useCallback } from "react";
import { usePOS } from "./PosContext";

export function DailyClosurePanel() {
  const { state, dispatch } = usePOS();
  const { todaySales } = state;
  const [cashDeclared, setCashDeclared] = useState("");
  const [loading, setLoading] = useState(false);
  const [closureMessage, setClosureMessage] = useState<string | null>(null);

  const close = useCallback(() => {
    dispatch({ type: "CLOSE_CLOSURE_PANEL" });
  }, [dispatch]);

  const fetchTodaySales = useCallback(async () => {
    try {
      const { invoke } = await import("@tauri-apps/api/core");
      const sales = await invoke<any[]>("get_today_sales");
      dispatch({ type: "SET_TODAY_SALES", payload: sales });
    } catch (err) {
      console.error("Failed to fetch today sales:", err);
    }
  }, [dispatch]);

  const handleXReport = useCallback(async () => {
    setLoading(true);
    try {
      await fetchTodaySales();
      setClosureMessage("Reporte X generado (sin corte de secuencia)");
    } finally {
      setLoading(false);
    }
  }, [fetchTodaySales]);

  const handleZReport = useCallback(async () => {
    setLoading(true);
    try {
      const { invoke } = await import("@tauri-apps/api/core");
      await fetchTodaySales();
      await invoke("save_offline_closure");
      setClosureMessage("Reporte Z generado — secuencia reiniciada");
    } catch (err) {
      console.error("Z report failed:", err);
      setClosureMessage("Error al generar reporte Z");
    } finally {
      setLoading(false);
    }
  }, [fetchTodaySales]);

  const handleRegisterOpen = useCallback(async () => {
    const amount = parseFloat(cashDeclared) || 0;
    try {
      const { invoke } = await import("@tauri-apps/api/core");
      await invoke("registrar_apertura", { amount });
      setClosureMessage(`Apertura registrada: ${amount.toFixed(2)} €`);
      setCashDeclared("");
    } catch (err) {
      console.error("Register open failed:", err);
      setClosureMessage("Error al registrar apertura");
    }
  }, [cashDeclared]);

  const handlePettyCashIn = useCallback(async () => {
    const amount = parseFloat(cashDeclared) || 0;
    if (amount <= 0) return;
    try {
      const { invoke } = await import("@tauri-apps/api/core");
      await invoke("registrar_ingreso_caja", { concepto: "Ingreso manual", amount });
      setClosureMessage(`Ingreso de caja: ${amount.toFixed(2)} €`);
      setCashDeclared("");
    } catch (err) {
      console.error("Petty cash in failed:", err);
    }
  }, [cashDeclared]);

  const handlePettyCashOut = useCallback(async () => {
    const amount = parseFloat(cashDeclared) || 0;
    if (amount <= 0) return;
    try {
      const { invoke } = await import("@tauri-apps/api/core");
      await invoke("registrar_retiro_caja", { concepto: "Retiro manual", amount });
      setClosureMessage(`Retiro de caja: ${amount.toFixed(2)} €`);
      setCashDeclared("");
    } catch (err) {
      console.error("Petty cash out failed:", err);
    }
  }, [cashDeclared]);

  const totalToday = todaySales.reduce((sum, s) => sum + s.total, 0);
  const salesCount = todaySales.length;

  return (
    <div className="modal-overlay">
      <div className="modal-panel closure-modal animate-fade-in">
        <div className="modal-header">
          <h2 className="modal-title">Corte de Caja</h2>
          <button className="modal-close" onClick={close}>
            ✕
          </button>
        </div>

        <div className="closure-content">
          {/* Summary */}
          <div className="closure-summary">
            <div className="closure-stat">
              <span className="closure-stat-label">Ventas hoy</span>
              <span className="closure-stat-value">{salesCount}</span>
            </div>
            <div className="closure-stat">
              <span className="closure-stat-label">Total hoy</span>
              <span className="closure-stat-value">{totalToday.toFixed(2)} €</span>
            </div>
          </div>

          {/* X/Z Report buttons */}
          <div className="closure-reports">
            <button
              className="closure-report-btn x-report"
              onClick={handleXReport}
              disabled={loading}
            >
              {loading ? "Generando..." : "📊 Reporte X"}
            </button>
            <button
              className="closure-report-btn z-report"
              onClick={handleZReport}
              disabled={loading}
            >
              {loading ? "Generando..." : "📋 Reporte Z"}
            </button>
          </div>

          {/* Cash declaration */}
          <div className="closure-cash-declaration">
            <h4 className="closure-section-title">Declaración de efectivo</h4>
            <div className="cash-input-wrapper">
              <input
                type="text"
                inputMode="decimal"
                className="cash-input"
                value={cashDeclared}
                onChange={(e) => {
                  const raw = e.target.value.replace(/[^0-9.,]/g, "").replace(",", ".");
                  setCashDeclared(raw);
                }}
                onFocus={() => setCashDeclared("")}
                placeholder="0.00"
              />
              <span className="cash-currency">€</span>
            </div>
            <div className="closure-cash-actions">
              <button
                className="payment-btn-secondary"
                onClick={handleRegisterOpen}
              >
                Abrir Caja
              </button>
              <button
                className="payment-btn-primary"
                onClick={handlePettyCashIn}
                disabled={!cashDeclared || parseFloat(cashDeclared) <= 0}
              >
                Ingreso
              </button>
              <button
                className="payment-btn-danger"
                onClick={handlePettyCashOut}
                disabled={!cashDeclared || parseFloat(cashDeclared) <= 0}
              >
                Retiro
              </button>
            </div>
          </div>

          {/* Today's sales list */}
          <div className="closure-sales-list">
            <h4 className="closure-section-title">Ventas del día</h4>
            {todaySales.length === 0 ? (
              <p className="closure-no-sales">
                {loading ? "Cargando..." : "No hay ventas hoy. Presiona Reporte X o Z para cargar."}
              </p>
            ) : (
              <div className="closure-sales-table">
                <div className="closure-sales-header">
                  <span>Ticket</span>
                  <span>Monto</span>
                  <span>Método</span>
                  <span>Hora</span>
                </div>
                {todaySales.map((sale) => (
                  <div key={sale.id} className="closure-sales-row">
                    <span>{sale.sequence}</span>
                    <span>{sale.total.toFixed(2)} €</span>
                    <span>
                      {sale.payments?.map((p) => p.metodo_pago).join(", ") || "—"}
                    </span>
                    <span>
                      {new Date(sale.created_at).toLocaleTimeString("es-AR", {
                        hour: "2-digit",
                        minute: "2-digit",
                      })}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Message */}
          {closureMessage && (
            <div className="closure-message">{closureMessage}</div>
          )}
        </div>
      </div>
    </div>
  );
}
