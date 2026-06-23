import { usePOS } from "./PosContext";

export function QuickActions() {
  const { state, dispatch } = usePOS();
  const hasCartItems = state.cart.length > 0;

  return (
    <div className="quick-actions">
      <h3 className="quick-actions-title">Acciones Rápidas</h3>
      <div className="quick-actions-grid">
        <button
          className="quick-action-btn"
          disabled={!hasCartItems}
          onClick={() => dispatch({ type: "OPEN_PAYMENT_MODAL" })}
        >
          💳 Cobrar
        </button>
        <button
          className="quick-action-btn"
          disabled={!hasCartItems}
          onClick={() => dispatch({ type: "OPEN_PAYMENT_MODAL" })}
        >
          🖨️ Imprimir
        </button>
        <button
          className="quick-action-btn"
          onClick={() => dispatch({ type: "OPEN_CLOSURE_PANEL" })}
        >
          📋 Corte
        </button>
        <button
          className="quick-action-btn"
          onClick={() => dispatch({ type: "OPEN_SETTINGS_PANEL" })}
        >
          ⚙️ Ajustes
        </button>
      </div>
    </div>
  );
}
