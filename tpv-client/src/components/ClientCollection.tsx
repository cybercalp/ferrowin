import { useState, useEffect } from "react";
import { invoke } from "@tauri-apps/api/core";
import { PendingInvoiceDossier } from "./ClientDossierView";

interface ClientCollectionProps {
  clientId: string;
  clientName: string;
  initialInvoice: PendingInvoiceDossier | null; // Selected invoice to pay, if any
  pendingInvoices: PendingInvoiceDossier[];      // List of all pending invoices for dropdown
  onClose: () => void;
  onSuccess: () => void;
  onLog: (msg: string) => void;
}

export function ClientCollection({
  clientId,
  clientName,
  initialInvoice,
  pendingInvoices,
  onClose,
  onSuccess,
  onLog,
}: ClientCollectionProps) {
  const [tipoCobro, setTipoCobro] = useState<"DEUDA" | "A_CUENTA">("DEUDA");
  const [selectedInvoiceId, setSelectedInvoiceId] = useState<string>("");
  const [importe, setImporte] = useState<number>(0);
  const [metodoPago, setMetodoPago] = useState<string>("Efectivo");
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");

  useEffect(() => {
    if (initialInvoice) {
      setTipoCobro("DEUDA");
      setSelectedInvoiceId(initialInvoice.id_factura);
      setImporte(initialInvoice.importe_pendiente);
    } else {
      setTipoCobro("A_CUENTA");
      setSelectedInvoiceId("");
      setImporte(0);
    }
  }, [initialInvoice]);

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
    if (importe <= 0) {
      setErrorMsg("El importe debe ser mayor a 0 €.");
      return;
    }

    if (tipoCobro === "DEUDA" && !selectedInvoiceId) {
      setErrorMsg("Debe seleccionar una factura para el cobro de deuda.");
      return;
    }

    setLoading(true);
    setErrorMsg("");
    const uuid = crypto.randomUUID();

    try {
      onLog(`Registrando cobro de ${importe} € para cliente ${clientName}. Tipo: ${tipoCobro}`);
      await invoke("registrar_cobro", {
        id: uuid,
        clienteId: clientId,
        facturaId: tipoCobro === "DEUDA" ? selectedInvoiceId : null,
        importe: Number(importe),
        metodoPago,
        tipoCobro,
      });

      onLog(`Cobro registrado correctamente de forma local. ID: ${uuid}`);
      onSuccess();
    } catch (err: any) {
      setErrorMsg(`Error al registrar cobro: ${err}`);
      onLog(`Error al registrar cobro: ${err}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={overlayStyle}>
      <style>{`
        @keyframes scaleIn {
          from { transform: scale(0.95); opacity: 0; }
          to { transform: scale(1); opacity: 1; }
        }
      `}</style>
      <div style={modalStyle}>
        <div style={modalHeaderStyle}>
          <h3 style={{ margin: 0, fontSize: "18px", fontWeight: "700" }}>💳 Registrar Cobro In Situ</h3>
          <button onClick={onClose} style={btnCloseXStyle}>✕</button>
        </div>

        <form onSubmit={handleSubmit} style={formStyle}>
          <div style={clientLabelRowStyle}>
            <span style={clientLabelTitleStyle}>Cliente:</span>
            <span style={clientLabelValueStyle}>{clientName}</span>
          </div>

          {/* Tipo de Cobro Toggle */}
          {!initialInvoice && (
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
          )}

          {/* Factura Selection */}
          {tipoCobro === "DEUDA" && (
            <div style={formGroupStyle}>
              <label style={labelStyle}>Factura Pendiente</label>
              {initialInvoice ? (
                <div style={lockedInvoiceStyle}>
                  📄 {initialInvoice.numero_factura} (Pendiente: {initialInvoice.importe_pendiente.toFixed(2)} €)
                </div>
              ) : (
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
              )}
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

          {/* Actions */}
          <div style={actionsRowStyle}>
            <button type="button" onClick={onClose} style={btnCancelStyle} disabled={loading}>
              Cancelar
            </button>
            <button type="submit" style={btnSubmitStyle} disabled={loading}>
              {loading ? "Registrando..." : "Confirmar Pago"}
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
const overlayStyle: React.CSSProperties = {
  position: "fixed",
  top: 0,
  left: 0,
  right: 0,
  bottom: 0,
  backgroundColor: "var(--overlay)",
  backdropFilter: "blur(4px)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  zIndex: 1000,
};

const modalStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-card)",
  borderRadius: "16px",
  width: "100%",
  maxWidth: "460px",
  boxShadow: "var(--shadow-lg)",
  border: "1px solid var(--border-default)",
  animation: "scaleIn 0.2s cubic-bezier(0.16, 1, 0.3, 1)",
  overflow: "hidden",
};

const modalHeaderStyle: React.CSSProperties = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  padding: "16px 20px",
  borderBottom: "1px solid var(--border-default)",
};

const btnCloseXStyle: React.CSSProperties = {
  border: "none",
  background: "none",
  fontSize: "16px",
  cursor: "pointer",
  color: "var(--text-placeholder)",
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

const lockedInvoiceStyle: React.CSSProperties = {
  padding: "10px 12px",
  borderRadius: "10px",
  backgroundColor: "var(--bg-page)",
  border: "1px solid var(--border-default)",
  color: "var(--text-secondary)",
  fontSize: "13px",
  fontWeight: "600",
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
