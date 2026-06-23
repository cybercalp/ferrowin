import { useEffect, useCallback } from "react";
import { usePOS } from "./PosContext";

const HARDCODED_TERMINAL_ID = "TPV-001";

export function TerminalSettings() {
  const { state, dispatch } = usePOS();
  const { terminalHealth } = state;

  const close = useCallback(() => {
    dispatch({ type: "CLOSE_SETTINGS_PANEL" });
  }, [dispatch]);

  useEffect(() => {
    (async () => {
      try {
        const { invoke } = await import("@tauri-apps/api/core");
        const health = await invoke<any>("get_terminal_health");
        dispatch({ type: "SET_TERMINAL_HEALTH", payload: health });
      } catch (err) {
        console.error("Failed to fetch terminal health:", err);
        // Provide fallback data for development
        dispatch({
          type: "SET_TERMINAL_HEALTH",
          payload: {
            terminal_id: HARDCODED_TERMINAL_ID,
            db_size_bytes: 0,
            pending_sales_count: 0,
            pending_closures_count: 0,
            online: false,
            app_version: "0.1.0",
          },
        });
      }
    })();
  }, [dispatch]);

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
  };

  return (
    <div className="modal-overlay">
      <div className="modal-panel settings-modal animate-fade-in">
        <div className="modal-header">
          <h2 className="modal-title">Configuración del Terminal</h2>
          <button className="modal-close" onClick={close}>
            ✕
          </button>
        </div>

        <div className="settings-content">
          {/* Terminal ID */}
          <div className="settings-section">
            <h4 className="settings-section-title">Identificación</h4>
            <div className="settings-field">
              <span className="settings-field-label">Terminal ID</span>
              <span className="settings-field-value">
                {terminalHealth?.terminal_id || HARDCODED_TERMINAL_ID}
              </span>
            </div>
            <div className="settings-field">
              <span className="settings-field-label">Versión App</span>
              <span className="settings-field-value">
                {terminalHealth?.app_version || "—"}
              </span>
            </div>
          </div>

          {/* Health status */}
          <div className="settings-section">
            <h4 className="settings-section-title">Estado de Salud</h4>
            <div className="settings-field">
              <span className="settings-field-label">Estado</span>
              <span
                className={`settings-status ${terminalHealth?.online ? "status-online" : "status-offline"}`}
              >
                {terminalHealth?.online ? "🟢 Online" : "🔴 Offline"}
              </span>
            </div>
            <div className="settings-field">
              <span className="settings-field-label">Base de datos</span>
              <span className="settings-field-value">
                {terminalHealth ? formatBytes(terminalHealth.db_size_bytes) : "—"}
              </span>
            </div>
            <div className="settings-field">
              <span className="settings-field-label">Ventas pendientes</span>
              <span className="settings-field-value">
                {terminalHealth?.pending_sales_count ?? "—"}
              </span>
            </div>
            <div className="settings-field">
              <span className="settings-field-label">Cortes pendientes</span>
              <span className="settings-field-value">
                {terminalHealth?.pending_closures_count ?? "—"}
              </span>
            </div>
          </div>

          {/* Printer config placeholder */}
          <div className="settings-section">
            <h4 className="settings-section-title">Impresora</h4>
            <div className="settings-field">
              <span className="settings-field-label">Tipo</span>
              <span className="settings-field-value settings-placeholder">
                PDF (por defecto)
              </span>
            </div>
            <div className="settings-field">
              <span className="settings-field-label">Formato</span>
              <span className="settings-field-value settings-placeholder">
                Thermal (próximamente)
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
