import { useState, useEffect } from "react";

interface ShareDocumentModalProps {
  numero: string;
  total: number;
  fecha: string;
  estado: string;
  clientName: string;
  onClose: () => void;
  onLog: (msg: string) => void;
}

export function ShareDocumentModal({
  numero,
  total,
  fecha,
  estado,
  clientName,
  onClose,
  onLog,
}: ShareDocumentModalProps) {
  const [phone, setPhone] = useState("");
  const [receiptText, setReceiptText] = useState("");
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    // Generate formatted receipt text
    const formattedDate = new Date(fecha).toLocaleString();
    const text = `📄 *COMPROBANTE DE FACTURA - FERROWIN*
----------------------------------------
*Nro. Factura:* ${numero}
*Fecha:* ${formattedDate}
*Cliente:* ${clientName}
*Total:* ${total.toFixed(2)} €
*Estado de Pago:* ${estado === "Cobrada" || estado === "Paid" ? "✅ Cobrada / Pagada" : "⚠️ Pendiente"}
----------------------------------------
¡Muchas gracias por su confianza!`;
    setReceiptText(text);
  }, [numero, total, fecha, estado, clientName]);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(receiptText);
      setCopied(true);
      onLog(`Texto del comprobante ${numero} copiado al portapapeles.`);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      onLog(`Error al copiar al portapapeles: ${err}`);
    }
  };

  const handleWhatsAppShare = () => {
    if (!phone) {
      alert("Por favor, ingrese un número de teléfono para enviar por WhatsApp.");
      return;
    }

    // Clean phone number (remove spaces, dashes, etc.)
    const cleanPhone = phone.replace(/\D/g, "");
    
    // URL encode the text
    const urlencodedText = encodeURIComponent(receiptText);
    
    // Open deep link
    const waUrl = `https://wa.me/${cleanPhone}?text=${urlencodedText}`;
    window.open(waUrl, "_blank");
    onLog(`Comprobante ${numero} compartido por WhatsApp al número: ${cleanPhone}`);
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
          <h3 style={{ margin: 0, fontSize: "18px", fontWeight: "700" }}>📤 Compartir Comprobante</h3>
          <button onClick={onClose} style={btnCloseXStyle}>✕</button>
        </div>

        <div style={contentStyle}>
          {/* Formatted Text Preview */}
          <div style={previewGroupStyle}>
            <label style={labelStyle}>Vista Previa del Mensaje</label>
            <textarea
              value={receiptText}
              onChange={(e) => setReceiptText(e.target.value)}
              style={textareaStyle}
              rows={8}
            />
            <button onClick={handleCopy} style={btnCopyStyle}>
              {copied ? "✅ ¡Copiado!" : "📋 Copiar al Portapapeles"}
            </button>
          </div>

          <hr style={hrStyle} />

          {/* WhatsApp sharing */}
          <div style={formGroupStyle}>
            <label style={labelStyle}>Enviar por WhatsApp</label>
            <div style={phoneInputRowStyle}>
              <input
                type="tel"
                placeholder="Ej: +34600123456"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                style={inputStyle}
              />
              <button onClick={handleWhatsAppShare} style={btnWhatsAppStyle}>
                💬 Enviar WhatsApp
              </button>
            </div>
            <span style={helpTextStyle}>
              Ingrese el número con código de país (ej: 34 para España) sin espacios ni símbolos.
            </span>
          </div>

          {/* Footer Close */}
          <div style={actionsRowStyle}>
            <button onClick={onClose} style={btnCancelStyle}>
              Cerrar
            </button>
          </div>
        </div>
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
  backgroundColor: "rgba(15, 23, 42, 0.4)",
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

const contentStyle: React.CSSProperties = {
  padding: "20px",
};

const previewGroupStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: "6px",
  textAlign: "left",
};

const labelStyle: React.CSSProperties = {
  fontSize: "13px",
  fontWeight: "600",
  color: "var(--text-secondary)",
};

const textareaStyle: React.CSSProperties = {
  padding: "10px 12px",
  borderRadius: "10px",
  border: "1px solid var(--border-input)",
  fontSize: "13px",
  fontFamily: "monospace",
  outline: "none",
  resize: "vertical",
  backgroundColor: "var(--bg-page)",
  lineHeight: "1.5",
};

const btnCopyStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-page)",
  color: "var(--accent-default)",
  border: "1px solid var(--border-input)",
  borderRadius: "8px",
  padding: "8px 12px",
  fontSize: "13px",
  fontWeight: "600",
  cursor: "pointer",
  marginTop: "4px",
  alignSelf: "flex-end",
};

const hrStyle: React.CSSProperties = {
  border: "0",
  borderTop: "1px solid var(--border-default)",
  margin: "16px 0",
};

const formGroupStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: "6px",
  textAlign: "left",
};

const phoneInputRowStyle: React.CSSProperties = {
  display: "flex",
  gap: "8px",
};

const inputStyle: React.CSSProperties = {
  flex: 1,
  padding: "10px 12px",
  borderRadius: "10px",
  border: "1px solid var(--border-input)",
  fontSize: "14px",
  outline: "none",
};

const btnWhatsAppStyle: React.CSSProperties = {
  backgroundColor: "#25d366", // WhatsApp Green
  color: "white",
  border: "none",
  borderRadius: "10px",
  padding: "10px 16px",
  fontWeight: "600",
  fontSize: "13px",
  cursor: "pointer",
};

const helpTextStyle: React.CSSProperties = {
  fontSize: "11px",
  color: "var(--text-muted)",
  marginTop: "2px",
};

const actionsRowStyle: React.CSSProperties = {
  display: "flex",
  justifyContent: "flex-end",
  marginTop: "20px",
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
