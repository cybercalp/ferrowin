import { useEffect, useState } from "react";
import { listen } from "@tauri-apps/api/event";

interface SyncStatusPayload {
  online: boolean;
  pending_sync_count: number;
}

export function SyncWarningBanner() {
  const [online, setOnline] = useState(true);
  const [pendingSyncCount, setPendingSyncCount] = useState(0);

  useEffect(() => {
    let unlisten: (() => void) | undefined;

    async function setupListener() {
      try {
        const unsub = await listen<SyncStatusPayload>(
          "sync-status-changed",
          (event) => {
            setOnline(event.payload.online);
            setPendingSyncCount(event.payload.pending_sync_count);
          }
        );
        unlisten = unsub;
      } catch (err) {
        console.error("Failed to listen to sync-status-changed", err);
      }
    }

    setupListener();

    return () => {
      if (unlisten) {
        unlisten();
      }
    };
  }, []);

  if (online && pendingSyncCount === 0) {
    return null;
  }

  return (
    <div style={containerStyle}>
      <style>{`
        @keyframes slideDown {
          from { transform: translateY(-20px); opacity: 0; }
          to { transform: translateY(0); opacity: 1; }
        }
        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
      `}</style>
      {!online ? (
        <div style={warningBannerStyle}>
          <div style={iconStyle}>⚠️</div>
          <div style={contentStyle}>
            <strong style={titleStyle}>Modo Offline Activo</strong>
            <span style={messageStyle}>
              Se ha perdido la conexión con el servidor. La facturación sigue estando permitida y las ventas se guardarán de forma local.
            </span>
            {pendingSyncCount > 0 && (
              <span style={badgeStyle}>
                {pendingSyncCount} {pendingSyncCount === 1 ? "registro pendiente" : "registros pendientes"} de sincronización
              </span>
            )}
          </div>
        </div>
      ) : (
        <div style={syncingNotificationStyle}>
          <div style={spinnerStyle}>🔄</div>
          <div style={contentStyle}>
            <strong style={titleStyle}>Sincronizando Datos</strong>
            <span style={messageStyle}>
              La conexión ha sido restablecida. Se están subiendo {pendingSyncCount} {pendingSyncCount === 1 ? "registro pendiente" : "registros pendientes"} al servidor central.
            </span>
          </div>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Inline styles for high fidelity modern look
// ---------------------------------------------------------------------------

const containerStyle: React.CSSProperties = {
  width: "100%",
  padding: "10px 16px",
  boxSizing: "border-box",
  animation: "slideDown 0.3s ease-out",
};

const warningBannerStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: "12px",
  backgroundColor: "var(--status-warning-bg)",
  border: "1px solid var(--status-warning-bg)",
  borderRadius: "12px",
  padding: "14px 20px",
  boxShadow: "var(--shadow-md)",
  color: "var(--status-warning-text)",
};

const syncingNotificationStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: "12px",
  backgroundColor: "var(--status-success-bg)",
  border: "1px solid var(--status-success-bg)",
  borderRadius: "12px",
  padding: "14px 20px",
  boxShadow: "var(--shadow-md)",
  color: "var(--status-success-text)",
};

const iconStyle: React.CSSProperties = {
  fontSize: "24px",
};

const spinnerStyle: React.CSSProperties = {
  fontSize: "20px",
  animation: "spin 2s linear infinite",
};

const contentStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  alignItems: "flex-start",
  textAlign: "left",
  gap: "2px",
};

const titleStyle: React.CSSProperties = {
  fontSize: "15px",
  fontWeight: "600",
};

const messageStyle: React.CSSProperties = {
  fontSize: "13px",
  opacity: 0.9,
};

const badgeStyle: React.CSSProperties = {
  display: "inline-block",
  marginTop: "4px",
  padding: "2px 8px",
  backgroundColor: "var(--status-warning-bg)",
  borderRadius: "6px",
  fontSize: "11px",
  fontWeight: "bold",
  textTransform: "uppercase",
};
