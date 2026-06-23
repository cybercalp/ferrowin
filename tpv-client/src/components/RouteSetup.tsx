import { useState, useEffect } from "react";
import { invoke } from "@tauri-apps/api/core";

export interface Cliente {
  id: string;
  nombre: string;
  nif?: string;
  email?: string;
  updated_at: string;
  activo: boolean;
}

interface RouteSetupProps {
  onRoutePrepared: (clientIds: string[]) => void;
  onLog: (msg: string) => void;
}

export function RouteSetup({ onRoutePrepared, onLog }: RouteSetupProps) {
  const [clientes, setClientes] = useState<Cliente[]>([]);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(false);
  const [statusMsg, setStatusMsg] = useState("");

  useEffect(() => {
    fetchClientes();
  }, []);

  async function fetchClientes() {
    try {
      const data: Cliente[] = await invoke("get_clientes");
      setClientes(data);
      onLog(`Cargados ${data.length} clientes del catálogo local.`);
    } catch (err: any) {
      onLog(`Error al cargar clientes: ${err}`);
    }
  }

  const toggleSelect = (id: string) => {
    setSelectedIds((prev) =>
      prev.includes(id) ? prev.filter((item) => item !== id) : [...prev, id]
    );
  };

  const handleSelectAll = () => {
    const filtered = filteredClientes.map(c => c.id);
    const allSelected = filtered.every(id => selectedIds.includes(id));
    if (allSelected) {
      setSelectedIds(prev => prev.filter(id => !filtered.includes(id)));
    } else {
      setSelectedIds(prev => Array.from(new Set([...prev, ...filtered])));
    }
  };

  const handlePrepareRoute = async () => {
    if (selectedIds.length === 0) {
      setStatusMsg("Por favor, seleccione al menos un cliente.");
      return;
    }

    setLoading(true);
    setStatusMsg("Sincronizando catálogo y descargando expedientes...");
    onLog(`Iniciando preparación de ruta para ${selectedIds.length} clientes.`);

    try {
      // 1. Sync catalog delta first
      await invoke("sync_catalog");
      onLog("Sincronización de catálogo delta completada.");

      // 2. Download dossiers
      await invoke("download_dossiers", { clientIds: selectedIds });
      onLog(`Expedientes descargados correctamente para: ${selectedIds.join(", ")}`);

      setStatusMsg("¡Ruta preparada correctamente!");
      onRoutePrepared(selectedIds);
    } catch (err: any) {
      setStatusMsg(`Error al preparar ruta: ${err}`);
      onLog(`Error al preparar ruta: ${err}`);
    } finally {
      setLoading(false);
    }
  };

  const filteredClientes = clientes.filter(
    (c) =>
      c.nombre.toLowerCase().includes(search.toLowerCase()) ||
      (c.nif && c.nif.toLowerCase().includes(search.toLowerCase()))
  );

  return (
    <div style={panelStyle}>
      <h3 style={headerStyle}>📍 Configuración de Ruta de Clientes</h3>
      <p style={subHeaderStyle}>
        Seleccione los clientes que visitará en su ruta de ventas ambulante para descargar sus fichas de crédito y facturas impagas de manera local.
      </p>

      <div style={filterRowStyle}>
        <input
          type="text"
          placeholder="Buscar por nombre o NIF..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={inputStyle}
        />
        <button onClick={fetchClientes} style={btnRefreshStyle} disabled={loading}>
          🔄 Actualizar
        </button>
      </div>

      <div style={tableContainerStyle}>
        <table style={tableStyle}>
          <thead>
            <tr>
              <th style={{ ...thStyle, width: "40px" }}>
                <input
                  type="checkbox"
                  checked={filteredClientes.length > 0 && filteredClientes.every(c => selectedIds.includes(c.id))}
                  onChange={handleSelectAll}
                  style={checkboxStyle}
                />
              </th>
              <th style={thStyle}>Nombre Cliente</th>
              <th style={thStyle}>NIF</th>
              <th style={thStyle}>Email</th>
            </tr>
          </thead>
          <tbody>
            {filteredClientes.length === 0 ? (
              <tr>
                <td colSpan={4} style={emptyTdStyle}>
                  No se encontraron clientes activos.
                </td>
              </tr>
            ) : (
              filteredClientes.map((c) => {
                const isSelected = selectedIds.includes(c.id);
                return (
                  <tr
                    key={c.id}
                    onClick={() => toggleSelect(c.id)}
                    style={{
                      ...trStyle,
                      backgroundColor: isSelected ? "rgba(37, 99, 235, 0.08)" : "transparent",
                    }}
                  >
                    <td style={tdStyle} onClick={(e) => e.stopPropagation()}>
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={() => toggleSelect(c.id)}
                        style={checkboxStyle}
                      />
                    </td>
                    <td style={{ ...tdStyle, fontWeight: "600" }}>{c.nombre}</td>
                    <td style={tdStyle}>{c.nif || "-"}</td>
                    <td style={tdStyle}>{c.email || "-"}</td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      <div style={actionsRowStyle}>
        <span style={selectedCountStyle}>
          {selectedIds.length} {selectedIds.length === 1 ? "cliente seleccionado" : "clientes seleccionados"}
        </span>
        <button
          onClick={handlePrepareRoute}
          disabled={loading || selectedIds.length === 0}
          style={{
            ...btnPrepareStyle,
            opacity: loading || selectedIds.length === 0 ? 0.6 : 1,
            cursor: loading || selectedIds.length === 0 ? "not-allowed" : "pointer",
          }}
        >
          {loading ? "Preparando..." : "🚀 Preparar Ruta"}
        </button>
      </div>

      {statusMsg && (
        <div
          style={{
            ...statusBoxStyle,
            backgroundColor: statusMsg.includes("Error") ? "var(--status-error-bg)" : "var(--status-success-bg)",
            color: statusMsg.includes("Error") ? "var(--status-error-text)" : "var(--status-success-text)",
            border: `1px solid ${statusMsg.includes("Error") ? "var(--status-error-bg)" : "var(--status-success-bg)"}`,
          }}
        >
          {statusMsg}
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Premium styling matching Ferrowin aesthetics
// ---------------------------------------------------------------------------
const panelStyle: React.CSSProperties = {
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

const headerStyle: React.CSSProperties = {
  fontSize: "20px",
  fontWeight: "700",
  color: "var(--text-primary)",
  margin: "0 0 8px 0",
};

const subHeaderStyle: React.CSSProperties = {
  fontSize: "14px",
  color: "var(--text-muted)",
  lineHeight: "1.5",
  margin: "0 0 20px 0",
};

const filterRowStyle: React.CSSProperties = {
  display: "flex",
  gap: "12px",
  marginBottom: "16px",
};

const inputStyle: React.CSSProperties = {
  flex: 1,
  padding: "10px 14px",
  borderRadius: "10px",
  border: "1px solid var(--border-input)",
  fontSize: "14px",
  outline: "none",
  transition: "border-color 0.2s",
};

const btnRefreshStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-page)",
  color: "var(--text-secondary)",
  border: "1px solid var(--border-input)",
  borderRadius: "10px",
  padding: "10px 16px",
  fontWeight: "600",
  fontSize: "14px",
  cursor: "pointer",
};

const tableContainerStyle: React.CSSProperties = {
  maxHeight: "280px",
  overflowY: "auto",
  border: "1px solid var(--border-default)",
  borderRadius: "12px",
  backgroundColor: "var(--bg-card)",
};

const tableStyle: React.CSSProperties = {
  width: "100%",
  borderCollapse: "collapse",
  fontSize: "14px",
};

const thStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-page)",
  color: "var(--text-muted)",
  fontWeight: "600",
  padding: "12px",
  textAlign: "left",
  borderBottom: "1px solid var(--border-default)",
  position: "sticky",
  top: 0,
  zIndex: 10,
};

const trStyle: React.CSSProperties = {
  cursor: "pointer",
  transition: "background-color 0.2s",
  borderBottom: "1px solid var(--border-default)",
};

const tdStyle: React.CSSProperties = {
  padding: "12px",
  color: "var(--text-secondary)",
};

const emptyTdStyle: React.CSSProperties = {
  padding: "24px",
  textAlign: "center",
  color: "var(--text-placeholder)",
};

const checkboxStyle: React.CSSProperties = {
  width: "16px",
  height: "16px",
  cursor: "pointer",
};

const actionsRowStyle: React.CSSProperties = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  marginTop: "20px",
};

const selectedCountStyle: React.CSSProperties = {
  fontSize: "14px",
  fontWeight: "600",
  color: "var(--accent-default)",
};

const btnPrepareStyle: React.CSSProperties = {
  backgroundColor: "var(--accent-default)",
  color: "white",
  border: "none",
  borderRadius: "10px",
  padding: "12px 24px",
  fontWeight: "600",
  fontSize: "14px",
  boxShadow: "var(--shadow-md)",
  transition: "transform 0.1s, box-shadow 0.1s",
};

const statusBoxStyle: React.CSSProperties = {
  marginTop: "16px",
  padding: "12px 16px",
  borderRadius: "10px",
  fontSize: "14px",
  fontWeight: "500",
  textAlign: "center",
};
