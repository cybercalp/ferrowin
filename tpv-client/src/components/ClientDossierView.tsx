import { useState, useEffect } from "react";
import { invoke } from "@tauri-apps/api/core";
import { useParams, useNavigate } from "react-router";

export interface RecentSaleDossier {
  id_factura: string;
  cliente_id: string;
  fecha: string;
  numero: string;
  total: number;
  estado: string;
}

export interface ClientStatsDossier {
  cliente_id: string;
  saldo_pendiente: number;
  limite_credito: number;
  articulos_mas_comprados_json: string;
}

export interface PendingInvoiceDossier {
  id_factura: string;
  cliente_id: string;
  numero_factura: string;
  importe_pendiente: number;
  fecha_emision: string;
}

export interface ClientDossier {
  cliente: { id: string; nombre: string; nif?: string; email?: string };
  estadisticas?: ClientStatsDossier | null;
  ventas_recientes: RecentSaleDossier[];
  facturas_pendientes: PendingInvoiceDossier[];
}

export function ClientDossierView() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [dossier, setDossier] = useState<ClientDossier | null>(null);
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");

  useEffect(() => {
    if (id) fetchDossier(id);
  }, [id]);

  async function fetchDossier(clientId: string) {
    setLoading(true);
    setErrorMsg("");
    try {
      const data: ClientDossier = await invoke("get_cliente_dossier", { clientId });
      setDossier(data);
    } catch (err: any) {
      setErrorMsg(`Error al cargar expediente: ${err}`);
    } finally {
      setLoading(false);
    }
  }

  // Format date helper
  const formatDate = (isoString: string) => {
    try {
      const d = new Date(isoString);
      return d.toLocaleDateString() + " " + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } catch {
      return isoString;
    }
  };

  if (loading) {
    return <div style={loadingStyle}>🔄 Cargando expediente de cliente...</div>;
  }

  if (errorMsg) {
    return (
      <div style={containerStyle}>
        <div style={errorStyle}>{errorMsg}</div>
        <button onClick={() => navigate(`/entities/${id}`)} style={backBtnStyle}>
          ← Volver a entidad
        </button>
      </div>
    );
  }

  if (!dossier) {
    return <div style={emptyStyle}>Seleccione un cliente para ver su ficha.</div>;
  }

  const { cliente, estadisticas, ventas_recientes, facturas_pendientes } = dossier;
  const limiteCredito = estadisticas?.limite_credito ?? 0;
  const saldoPendiente = estadisticas?.saldo_pendiente ?? 0;
  const creditoDisponible = Math.max(0, limiteCredito - saldoPendiente);

  return (
    <div style={containerStyle}>
      {/* Back button */}
      <div style={navBarStyle}>
        <button onClick={() => navigate(`/entities/${id}`)} style={backBtnStyle}>
          ← Volver a entidad
        </button>
        <button
          onClick={() => navigate(`/clients/${id}/collection`)}
          style={collectBtnStyle}
        >
          💳 Registrar Cobro
        </button>
      </div>

      {/* Header card */}
      <div style={headerCardStyle}>
        <div style={avatarStyle}>👤</div>
        <div>
          <h2 style={clientNameStyle}>{cliente.nombre}</h2>
          <div style={clientMetaStyle}>
            {cliente.nif && <span>NIF: <strong>{cliente.nif}</strong></span>}
            {cliente.email && <span style={{ marginLeft: "15px" }}>Email: <strong>{cliente.email}</strong></span>}
          </div>
        </div>
      </div>

      {/* Credit status cards */}
      <div style={statsGridStyle}>
        <div style={statCardStyle}>
          <span style={statLabelStyle}>LÍMITE DE CRÉDITO</span>
          <span style={statValueStyle}>{limiteCredito.toFixed(2)} €</span>
        </div>
        <div style={{ ...statCardStyle, borderLeftColor: "var(--color-danger)" }}>
          <span style={statLabelStyle}>SALDO PENDIENTE</span>
          <span style={{ ...statValueStyle, color: "var(--color-danger)" }}>{saldoPendiente.toFixed(2)} €</span>
        </div>
        <div style={{ ...statCardStyle, borderLeftColor: "var(--color-success)" }}>
          <span style={statLabelStyle}>CRÉDITO DISPONIBLE</span>
          <span style={{ ...statValueStyle, color: "var(--color-success)" }}>{creditoDisponible.toFixed(2)} €</span>
        </div>
      </div>

      {/* Main split details */}
      <div style={detailsGridStyle}>
        {/* Unpaid Invoices */}
        <div style={sectionBlockStyle}>
          <h4 style={sectionTitleStyle}>💸 Facturas Pendientes (Impagas)</h4>
          <div style={listContainerStyle}>
            {facturas_pendientes.length === 0 ? (
              <div style={emptyListStyle}>No hay facturas pendientes de cobro.</div>
            ) : (
              facturas_pendientes.map((f) => (
                <div key={f.id_factura} style={listItemStyle}>
                  <div style={itemMetaStyle}>
                    <span style={itemTitleStyle}>{f.numero_factura}</span>
                    <span style={itemDateStyle}>{formatDate(f.fecha_emision)}</span>
                  </div>
                  <div style={itemActionStyle}>
                    <span style={itemAmountStyle}>{f.importe_pendiente.toFixed(2)} €</span>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Recent sales history */}
        <div style={sectionBlockStyle}>
          <h4 style={sectionTitleStyle}>📜 Historial de Ventas Recientes</h4>
          <div style={listContainerStyle}>
            {ventas_recientes.length === 0 ? (
              <div style={emptyListStyle}>No hay historial de ventas.</div>
            ) : (
              ventas_recientes.map((v) => (
                <div key={v.id_factura} style={listItemStyle}>
                  <div style={itemMetaStyle}>
                    <span style={itemTitleStyle}>{v.numero}</span>
                    <span style={itemDateStyle}>{formatDate(v.fecha)}</span>
                    <span
                      style={{
                        ...badgeStyle,
                        backgroundColor: v.estado === "Cobrada" || v.estado === "Paid" ? "var(--status-success-bg)" : "var(--status-error-bg)",
                        color: v.estado === "Cobrada" || v.estado === "Paid" ? "var(--status-success-text)" : "var(--status-error-text)",
                      }}
                    >
                      {v.estado}
                    </span>
                  </div>
                  <div style={itemActionStyle}>
                    <span style={{ ...itemAmountStyle, color: "var(--text-primary)" }}>{v.total.toFixed(2)} €</span>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Premium styling matching Ferrowin aesthetics
// ---------------------------------------------------------------------------
const containerStyle: React.CSSProperties = {
  background: "var(--glass-bg)",
  backdropFilter: "blur(12px)",
  borderRadius: "16px",
  padding: "24px",
  margin: "20px auto",
  maxWidth: "800px",
  boxShadow: "var(--shadow-md)",
  border: "1px solid var(--glass-border)",
  textAlign: "left",
};

const navBarStyle: React.CSSProperties = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  marginBottom: "16px",
};

const backBtnStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-page)",
  color: "var(--text-secondary)",
  border: "1px solid var(--border-input)",
  borderRadius: "8px",
  padding: "8px 14px",
  fontSize: "13px",
  fontWeight: "600",
  cursor: "pointer",
};

const collectBtnStyle: React.CSSProperties = {
  backgroundColor: "var(--accent-default)",
  color: "#fff",
  border: "none",
  borderRadius: "8px",
  padding: "8px 14px",
  fontSize: "13px",
  fontWeight: "600",
  cursor: "pointer",
};

const loadingStyle: React.CSSProperties = {
  textAlign: "center",
  padding: "40px",
  color: "var(--text-muted)",
  fontWeight: "500",
};

const errorStyle: React.CSSProperties = {
  backgroundColor: "var(--status-error-bg)",
  color: "var(--status-error-text)",
  border: "1px solid var(--status-error-bg)",
  borderRadius: "12px",
  padding: "16px",
  marginBottom: "16px",
  textAlign: "center",
};

const emptyStyle: React.CSSProperties = {
  textAlign: "center",
  padding: "40px",
  color: "var(--text-placeholder)",
  background: "var(--bg-page)",
  border: "2px dashed var(--border-input)",
  borderRadius: "16px",
  maxWidth: "800px",
  margin: "20px auto",
};

const headerCardStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: "16px",
  backgroundColor: "var(--bg-page)",
  padding: "16px 20px",
  borderRadius: "12px",
  border: "1px solid var(--border-default)",
  marginBottom: "20px",
};

const avatarStyle: React.CSSProperties = {
  fontSize: "32px",
  backgroundColor: "var(--border-default)",
  width: "56px",
  height: "56px",
  borderRadius: "50%",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
};

const clientNameStyle: React.CSSProperties = {
  fontSize: "20px",
  fontWeight: "700",
  color: "var(--text-primary)",
  margin: 0,
};

const clientMetaStyle: React.CSSProperties = {
  fontSize: "13px",
  color: "var(--text-muted)",
  marginTop: "4px",
};

const statsGridStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))",
  gap: "16px",
  marginBottom: "24px",
};

const statCardStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-card)",
  border: "1px solid var(--border-default)",
  borderLeft: "4px solid var(--accent-default)",
  borderRadius: "12px",
  padding: "16px",
  display: "flex",
  flexDirection: "column",
  boxShadow: "var(--shadow-sm)",
};

const statLabelStyle: React.CSSProperties = {
  fontSize: "11px",
  fontWeight: "700",
  color: "var(--text-muted)",
  letterSpacing: "0.05em",
  marginBottom: "4px",
};

const statValueStyle: React.CSSProperties = {
  fontSize: "22px",
  fontWeight: "800",
  color: "var(--text-primary)",
};

const detailsGridStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(340px, 1fr))",
  gap: "20px",
};

const sectionBlockStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-card)",
  border: "1px solid var(--border-default)",
  borderRadius: "12px",
  padding: "18px",
};

const sectionTitleStyle: React.CSSProperties = {
  fontSize: "15px",
  fontWeight: "700",
  color: "var(--text-primary)",
  margin: "0 0 14px",
};

const listContainerStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: "10px",
  maxHeight: "240px",
  overflowY: "auto",
};

const emptyListStyle: React.CSSProperties = {
  textAlign: "center",
  padding: "20px",
  color: "var(--text-placeholder)",
  fontSize: "13px",
};

const listItemStyle: React.CSSProperties = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  padding: "10px 12px",
  borderRadius: "8px",
  backgroundColor: "var(--bg-page)",
  border: "1px solid var(--border-default)",
};

const itemMetaStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: "2px",
  alignItems: "flex-start",
};

const itemTitleStyle: React.CSSProperties = {
  fontWeight: "600",
  color: "var(--text-secondary)",
  fontSize: "13px",
};

const itemDateStyle: React.CSSProperties = {
  fontSize: "11px",
  color: "var(--text-muted)",
};

const badgeStyle: React.CSSProperties = {
  fontSize: "10px",
  fontWeight: "700",
  padding: "2px 6px",
  borderRadius: "4px",
  marginTop: "2px",
};

const itemActionStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: "12px",
};

const itemAmountStyle: React.CSSProperties = {
  fontWeight: "700",
  fontSize: "14px",
  color: "var(--color-danger)",
};
