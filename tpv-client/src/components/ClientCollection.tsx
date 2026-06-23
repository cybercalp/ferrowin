import { useState, useEffect } from "react";
import { invoke } from "@tauri-apps/api/core";
import { useParams, useNavigate } from "react-router";
import type { PendingInvoiceDossier } from "./ClientDossierView";

interface ClienteInfo {
  id: string;
  nombre: string;
  nif?: string;
}

export function ClientCollection() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [cliente, setCliente] = useState<ClienteInfo | null>(null);
  const [pendingInvoices, setPendingInvoices] = useState<PendingInvoiceDossier[]>([]);
  const [loading, setLoading] = useState(true);
  const [tipoCobro, setTipoCobro] = useState<"DEUDA" | "A_CUENTA">("DEUDA");
  const [selectedInvoiceId, setSelectedInvoiceId] = useState<string>("");
  const [importe, setImporte] = useState<number>(0);
  const [metodoPago, setMetodoPago] = useState<string>("Efectivo");
  const [submitting, setSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");
  const [successMsg, setSuccessMsg] = useState("");

  useEffect(() => {
    if (id) fetchData(id);
  }, [id]);

  async function fetchData(clientId: string) {
    setLoading(true);
    try {
      // Fetch client info
      const clienteData: ClienteInfo = await invoke("get_entidad_by_id", { id: clientId });
      setCliente(clienteData);

      // Fetch pending invoices
      try {
        const dossier = await invoke<any>("get_cliente_dossier", { clientId });
        setPendingInvoices(dossier.facturas_pendientes || []);
        if (dossier.facturas_pendientes?.length > 0) {
          setTipoCobro("DEUDA");
          setSelectedInvoiceId(dossier.facturas_pendientes[0].id_factura);
          setImporte(dossier.facturas_pendientes[0].importe_pendiente);
        } else {
          setTipoCobro("A_CUENTA");
        }
      } catch {
        // If dossier fails, default to A_CUENTA
        setTipoCobro("A_CUENTA");
      }
    } catch (err: any) {
      setErrorMsg(`Error al cargar datos: ${err}`);
    } finally {
      setLoading(false);
    }
  }

  // Handle invoice change in dropdown
  const handleInvoiceChange = (invoiceId: string) => {
    setSelectedInvoiceId(invoiceId);
    const invoice = pendingInvoices.find((i) => i.id_factura === invoiceId);
    if (invoice) {
      setImporte(invoice.importe_pendiente);
    } else {
      setImporte(0);
    }
  };

  const handleTipoCobroChange = (tipo: "DEUDA" | "A_CUENTA") => {
    setTipoCobro(tipo);
    if (tipo === "A_CUENTA") {
      setSelectedInvoiceId("");
      setImporte(0);
    } else if (pendingInvoices.length > 0) {
      setSelectedInvoiceId(pendingInvoices[0].id_factura);
      setImporte(pendingInvoices[0].importe_pendiente);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!id || !cliente) return;
    if (importe <= 0) {
      setErrorMsg("El importe debe ser mayor a 0 €.");
      return;
    }

    if (tipoCobro === "DEUDA" && !selectedInvoiceId) {
      setErrorMsg("Debe seleccionar una factura para el cobro de deuda.");
      return;
    }

    setSubmitting(true);
    setErrorMsg("");
    setSuccessMsg("");
    const uuid = crypto.randomUUID();

    try {
      await invoke("registrar_cobro", {
        id: uuid,
        clienteId: id,
        facturaId: tipoCobro === "DEUDA" ? selectedInvoiceId : null,
        importe: Number(importe),
        metodoPago,
        tipoCobro,
      });

      setSuccessMsg(`Cobro de ${importe.toFixed(2)} € registrado correctamente.`);
      // Reset form
      setImporte(0);
      setSelectedInvoiceId("");
      // Refresh invoices
      setTimeout(() => fetchData(id), 1000);
    } catch (err: any) {
      setErrorMsg(`Error al registrar cobro: ${err}`);
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return <div style={loadingStyle}>🔄 Cargando datos del cliente...</div>;
  }

  if (!cliente) {
    return (
      <div style={containerStyle}>
        <div style={errorStyle}>{errorMsg || "Cliente no encontrado"}</div>
        <button onClick={() => navigate("/entities")} style={backBtnStyle}>
          ← Entidades
        </button>
      </div>
    );
  }

  return (
    <div style={containerStyle}>
      {/* Navigation */}
      <div style={navBarStyle}>
        <button onClick={() => navigate(`/entities/${id}`)} style={backBtnStyle}>
          ← Volver a entidad
        </button>
        <button onClick={() => navigate(`/clients/${id}/dossier`)} style={dossierBtnStyle}>
          📋 Ver Dossier
        </button>
      </div>

      <div style={modalStyle}>
        <div style={modalHeaderStyle}>
          <h3 style={{ margin: 0, fontSize: "18px", fontWeight: "700" }}>💳 Registrar Cobro In Situ</h3>
        </div>

        <form onSubmit={handleSubmit} style={formStyle}>
          <div style={clientLabelRowStyle}>
            <span style={clientLabelTitleStyle}>Cliente:</span>
            <span style={clientLabelValueStyle}>{cliente.nombre}</span>
          </div>

          {/* Tipo de Cobro Toggle */}
          <div style={formGroupStyle}>
            <label style={labelStyle}>Tipo de Cobro</label>
            <div style={toggleButtonGroupStyle}>
              <button
                type="button"
                onClick={() => handleTipoCobroChange("DEUDA")}
                style={{
                  ...toggleButtonStyle,
                  backgroundColor: tipoCobro === "DEUDA" ? "var(--accent-default)" : "var(--bg-page)",
                  color: tipoCobro === "DEUDA" ? "#fff" : "var(--text-secondary)",
                }}
                disabled={pendingInvoices.length === 0}
              >
                Cobro de Deuda (Factura)
              </button>
              <button
                type="button"
                onClick={() => handleTipoCobroChange("A_CUENTA")}
                style={{
                  ...toggleButtonStyle,
                  backgroundColor: tipoCobro === "A_CUENTA" ? "var(--accent-default)" : "var(--bg-page)",
                  color: tipoCobro === "A_CUENTA" ? "#fff" : "var(--text-secondary)",
                }}
              >
                Entrega a Cuenta
              </button>
            </div>
          </div>

          {/* Factura Selection */}
          {tipoCobro === "DEUDA" && (
            <div style={formGroupStyle}>
              <label style={labelStyle}>Factura Pendiente</label>
              <select
                value={selectedInvoiceId}
                onChange={(e) => handleInvoiceChange(e.target.value)}
                style={selectStyle}
                required
              >
                <option value="">-- Seleccione una factura --</option>
                {pendingInvoices.map((inv) => (
                  <option key={inv.id_factura} value={inv.id_factura}>
                    {inv.numero_factura} (Pendiente: {inv.importe_pendiente.toFixed(2)} €)
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Importe */}
          <div style={formGroupStyle}>
            <label style={labelStyle}>Importe (€)</label>
            <input
              type="number"
              step="0.01"
              value={importe || ""}
              onChange={(e) => setImporte(parseFloat(e.target.value))}
              placeholder="0.00"
              style={inputStyle}
              required
              min="0.01"
            />
          </div>

          {/* Método de Pago */}
          <div style={formGroupStyle}>
            <label style={labelStyle}>Método de Pago</label>
            <select
              value={metodoPago}
              onChange={(e) => setMetodoPago(e.target.value)}
              style={selectStyle}
              required
            >
              <option value="Efectivo">💵 Efectivo</option>
              <option value="Tarjeta">💳 Tarjeta de Crédito/Débito</option>
              <option value="Transferencia">🏦 Transferencia Bancaria</option>
              <option value="Bizum">📱 Bizum</option>
            </select>
          </div>

          {errorMsg && <div style={errorBoxStyle}>{errorMsg}</div>}
          {successMsg && <div style={successBoxStyle}>{successMsg}</div>}

          {/* Actions */}
          <div style={actionsRowStyle}>
            <button type="button" onClick={() => navigate(`/entities/${id}`)} style={btnCancelStyle} disabled={submitting}>
              Cancelar
            </button>
            <button type="submit" style={btnSubmitStyle} disabled={submitting}>
              {submitting ? "Registrando..." : "Confirmar Pago"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Premium styling matching Ferrowin aesthetics
// ---------------------------------------------------------------------------
const containerStyle: React.CSSProperties = {
  maxWidth: "600px",
  margin: "20px auto",
  padding: "0 20px",
  fontFamily: "'Inter', system-ui, sans-serif",
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

const dossierBtnStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-page)",
  color: "var(--accent-default)",
  border: "1px solid var(--border-input)",
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

const modalStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-card)",
  borderRadius: "16px",
  boxShadow: "var(--shadow-lg)",
  border: "1px solid var(--border-default)",
  overflow: "hidden",
};

const modalHeaderStyle: React.CSSProperties = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  padding: "16px 20px",
  borderBottom: "1px solid var(--border-default)",
};

const formStyle: React.CSSProperties = {
  padding: "20px",
};

const clientLabelRowStyle: React.CSSProperties = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  backgroundColor: "var(--bg-page)",
  padding: "10px 14px",
  borderRadius: "10px",
  border: "1px solid var(--border-input)",
  marginBottom: "16px",
  fontSize: "14px",
};

const clientLabelTitleStyle: React.CSSProperties = {
  fontWeight: "500",
  color: "var(--text-muted)",
};

const clientLabelValueStyle: React.CSSProperties = {
  fontWeight: "700",
  color: "var(--text-primary)",
};

const formGroupStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: "6px",
  marginBottom: "16px",
  textAlign: "left",
};

const labelStyle: React.CSSProperties = {
  fontSize: "13px",
  fontWeight: "600",
  color: "var(--text-secondary)",
};

const toggleButtonGroupStyle: React.CSSProperties = {
  display: "flex",
  gap: "8px",
};

const toggleButtonStyle: React.CSSProperties = {
  flex: 1,
  border: "none",
  borderRadius: "8px",
  padding: "10px",
  fontSize: "12px",
  fontWeight: "600",
  cursor: "pointer",
  transition: "background-color 0.2s, color 0.2s",
};

const selectStyle: React.CSSProperties = {
  padding: "10px 12px",
  borderRadius: "10px",
  border: "1px solid var(--border-input)",
  fontSize: "14px",
  outline: "none",
  backgroundColor: "var(--bg-card)",
};

const inputStyle: React.CSSProperties = {
  padding: "10px 12px",
  borderRadius: "10px",
  border: "1px solid var(--border-input)",
  fontSize: "14px",
  outline: "none",
};

const errorBoxStyle: React.CSSProperties = {
  backgroundColor: "var(--status-error-bg)",
  color: "var(--status-error-text)",
  border: "1px solid var(--status-error-bg)",
  borderRadius: "8px",
  padding: "10px 14px",
  fontSize: "13px",
  marginBottom: "16px",
};

const successBoxStyle: React.CSSProperties = {
  backgroundColor: "var(--status-success-bg)",
  color: "var(--status-success-text)",
  border: "1px solid var(--status-success-bg)",
  borderRadius: "8px",
  padding: "10px 14px",
  fontSize: "13px",
  marginBottom: "16px",
};

const actionsRowStyle: React.CSSProperties = {
  display: "flex",
  justifyContent: "flex-end",
  gap: "10px",
  marginTop: "24px",
};

const btnCancelStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-page)",
  color: "var(--text-secondary)",
  border: "1px solid var(--border-input)",
  borderRadius: "8px",
  padding: "10px 16px",
  fontWeight: "600",
  fontSize: "14px",
  cursor: "pointer",
};

const btnSubmitStyle: React.CSSProperties = {
  backgroundColor: "var(--accent-default)",
  color: "white",
  border: "none",
  borderRadius: "8px",
  padding: "10px 20px",
  fontWeight: "600",
  fontSize: "14px",
  cursor: "pointer",
  boxShadow: "var(--shadow-md)",
};
