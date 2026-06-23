import { useCallback, useEffect, useState } from "react";
import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import { useNavigate } from "react-router";
import { usePOS } from "./PosContext";
import { useAuth } from "../context/AuthContext";
import { BarcodeInput } from "./BarcodeInput";
import { CustomerSelector } from "./CustomerSelector";
import { FamilyNavBar } from "./FamilyNavBar";
import { ProductList } from "./ProductList";
import { CartPanel } from "./CartPanel";
import { NumericKeypad } from "./NumericKeypad";
import { PanelSplitter } from "./PanelSplitter";
import { PaymentModal } from "./PaymentModal";
import { ReceiptPreview } from "./ReceiptPreview";
import { DailyClosurePanel } from "./DailyClosurePanel";
import { TerminalSettings } from "./TerminalSettings";
import "../pos-ferreteria.css";

const RPW_KEY = "ferrowin:rightPanelWidth";
const RPW_DEFAULT = 500;
const RPW_MIN = 320;
const RPW_MAX = 700;

function loadRpw(): number {
  try {
    const v = localStorage.getItem(RPW_KEY);
    return v ? Math.min(RPW_MAX, Math.max(RPW_MIN, parseInt(v, 10))) : RPW_DEFAULT;
  } catch {
    return RPW_DEFAULT;
  }
}

export function POSMainLayout() {
  const { state, dispatch } = usePOS();
  const { logout, user } = useAuth();
  const navigate = useNavigate();
  const [rightPanelWidth, setRightPanelWidth] = useState<number>(loadRpw);

  // Save to localStorage only when the user finishes dragging
  const commitRpw = useCallback((w: number) => {
    localStorage.setItem(RPW_KEY, String(w));
  }, []);

  // Load terminal health on mount + sync catalog + listen for live sync-status updates
  useEffect(() => {
    let unlisten: (() => void) | undefined;

    (async () => {
      try {
        // Fetch initial health — get_terminal_health now reads the
        // real online status from the shared AtomicBool.
        const health = await invoke<any>("get_terminal_health");
        dispatch({ type: "SET_TERMINAL_HEALTH", payload: health });

        // Sync catalog from backend (products, families, tax types)
        try {
          await invoke("sync_catalog");
        } catch {
          // silent — catalog sync may fail if offline or no token
        }

        // Reactively keep terminalHealth.online in sync with the
        // background sync loop (every 30 s).
        const unsub = await listen<{ online: boolean }>(
          "sync-status-changed",
          async () => {
            try {
              const refreshed = await invoke<any>("get_terminal_health");
              dispatch({ type: "SET_TERMINAL_HEALTH", payload: refreshed });
            } catch {
              // silent
            }
          },
        );
        unlisten = unsub;
      } catch (err) {
        console.error("Failed to fetch terminal health:", err);
      }
    })();

    return () => {
      if (unlisten) unlisten();
    };
  }, [dispatch]);

  // Dismiss keypad when payment modal opens or active product leaves cart
  useEffect(() => {
    if (!state.activeQtyProductId) return;
    const shouldDismiss =
      state.paymentModalOpen ||
      state.cart.length === 0 ||
      !state.cart.some((i) => i.product.id === state.activeQtyProductId);
    if (shouldDismiss) {
      dispatch({ type: "SET_ACTIVE_QTY_PRODUCT", payload: null });
      dispatch({ type: "SET_KEYPAD_BUFFER", payload: "" });
    }
  }, [
    state.paymentModalOpen,
    state.cart,
    state.activeQtyProductId,
    dispatch,
  ]);

  return (
    <div className="pos-wrapper">
      {/* ── Header bar ── */}
      <header className="pos-header">
        <div className="pos-header-left">
          <span className="pos-title">
            <span
              className={`pos-status-dot ${state.terminalHealth?.online ? "dot-online" : "dot-offline"}`}
            />
            Ferrowin TPV
          </span>
        </div>
        <div className="pos-header-center">
          <BarcodeInput />
        </div>
        <div className="pos-header-right">
          <CustomerSelector />
          <button
            className="pos-header-btn"
            onClick={() =>
              dispatch({
                type: "SET_VIEW_MODE",
                payload: state.viewMode === "list" ? "grid" : "list",
              })
            }
            title={state.viewMode === "list" ? "Modo táctil" : "Modo lista"}
          >
            <span
              className={`header-btn-icon ${state.viewMode === "grid" ? "header-icon-grid-active" : "header-icon-grid"}`}
            />
            <span className="header-btn-label">
              {state.viewMode === "list" ? "Táctil" : "Lista"}
            </span>
          </button>
          {state.viewMode === "grid" && (
            <div className="grid-columns-selector" title="Columnas">
              <button
                className="grid-col-btn"
                onClick={() => {
                  const next = Math.max(2, state.gridColumns - 1);
                  localStorage.setItem("ferrowin:gridColumns", String(next));
                  dispatch({ type: "SET_GRID_COLUMNS", payload: next });
                }}
                disabled={state.gridColumns <= 2}
              >
                −
              </button>
              <span className="grid-col-value">{state.gridColumns}</span>
              <button
                className="grid-col-btn"
                onClick={() => {
                  const next = Math.min(8, state.gridColumns + 1);
                  localStorage.setItem("ferrowin:gridColumns", String(next));
                  dispatch({ type: "SET_GRID_COLUMNS", payload: next });
                }}
                disabled={state.gridColumns >= 8}
              >
                +
              </button>
            </div>
          )}
          <button
            className="pos-header-btn"
            onClick={() => dispatch({ type: "OPEN_CLOSURE_PANEL" })}
            title="Corte de caja"
          >
            <span className="header-btn-icon header-icon-closure" />
            <span className="header-btn-label">Corte</span>
          </button>
          <button
            className="pos-header-btn"
            onClick={() => navigate("/entities")}
            title="Gestión de entidades"
          >
            <span className="header-btn-icon header-icon-entities" />
            <span className="header-btn-label">Clientes</span>
          </button>
          <button
            className="pos-header-btn"
            onClick={() => dispatch({ type: "OPEN_SETTINGS_PANEL" })}
            title="Configuracion"
          >
            <span className="header-btn-icon header-icon-settings" />
            <span className="header-btn-label">Ajustes</span>
          </button>
          <div className="pos-header-separator" />
          <span className="pos-header-user">
            {user?.username || ""}
          </span>
          <button
            className="pos-header-btn pos-header-btn-logout"
            onClick={logout}
            title="Cerrar sesión"
          >
            <span className="header-btn-icon header-icon-logout" />
            <span className="header-btn-label">Salir</span>
          </button>
        </div>
      </header>

      {/* ── Body: family nav + product list | splitter | cart ── */}
      <div className="pos-body">
        {/* Left panel: family nav + product list */}
        <div className="pos-left-panel">
          <FamilyNavBar />
          <ProductList />
        </div>

        {/* Draggable vertical splitter */}
        <PanelSplitter
          onWidthChange={setRightPanelWidth}
          onCommit={commitRpw}
          min={RPW_MIN}
          max={RPW_MAX}
        />

        {/* Numeric keypad (floating overlay — absolutely positioned) */}
        <NumericKeypad visible={state.activeQtyProductId !== null} />

        {/* Right panel: cart */}
        <div
          className="pos-right-panel"
          style={{ width: rightPanelWidth }}
        >
          <CartPanel />
        </div>
      </div>

      {/* ── Status bar ── */}
      <footer className="pos-status-bar">
        <div className="status-bar-left">
          <span
            className={`status-indicator ${state.terminalHealth?.online ? "status-online" : "status-offline"}`}
          >
            {state.terminalHealth?.online ? "Online" : "Offline"}
          </span>
          <span className="status-separator">|</span>
          <span className="status-terminal">
            Terminal: {state.terminalHealth?.terminal_id || "TPV-001"}
          </span>
        </div>
        <div className="status-bar-right">
          <span className="status-sales-today">
            Ventas hoy: {state.todaySales
              .reduce((sum, s) => sum + s.total, 0)
              .toFixed(2)} €
          </span>
          {state.terminalHealth && state.terminalHealth.pending_sales_count > 0 && (
            <>
              <span className="status-separator">|</span>
              <span className="status-pending">
                Pendientes: {state.terminalHealth.pending_sales_count}
              </span>
            </>
          )}
        </div>
      </footer>

      {/* ── Modals & Panels ── */}
      {state.paymentModalOpen && <PaymentModal />}
      {state.receiptModalOpen && <ReceiptPreview />}
      {state.closurePanelOpen && <DailyClosurePanel />}
      {state.settingsPanelOpen && <TerminalSettings />}
    </div>
  );
}
