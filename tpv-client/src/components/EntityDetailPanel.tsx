import { useState, useEffect } from "react";
import { invoke } from "@tauri-apps/api/core";
import { useParams, useNavigate } from "react-router";

// Local interface — extends the base Cliente shape with the roles field.
export interface Entidad {
  id: string;
  nombre: string;
  nif?: string;
  email?: string;
  updated_at: string;
  activo: boolean;
  roles?: string; // e.g. "CLIENTE", "PROVEEDOR", "CLIENTE,PROVEEDOR"
}

interface Direccion {
  id: string;
  entidad_id: string;
  tipo_direccion: string;
  calle: string;
  ciudad: string;
  provincia: string;
  codigo_postal: string;
  pais: string;
}

interface Contacto {
  id: string;
  entidad_id: string;
  nombre: string;
  puesto?: string;
  email?: string;
  telefono?: string;
}

interface Nota {
  id: string;
  entidad_id: string;
  nota: string;
  creado_en: string;
}

type TabKey = "datos" | "direcciones" | "contactos" | "notas";

export function EntityDetailPanel() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [entity, setEntity] = useState<Entidad | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [activeTab, setActiveTab] = useState<TabKey>("datos");

  // Direcciones state
  const [direcciones, setDirecciones] = useState<Direccion[]>([]);
  const [loadingDirecciones, setLoadingDirecciones] = useState(false);
  const [showDireccionForm, setShowDireccionForm] = useState(false);
  const [direccionForm, setDireccionForm] = useState<Omit<Direccion, "id" | "entidad_id">>({
    tipo_direccion: "FISCAL",
    calle: "",
    ciudad: "",
    provincia: "",
    codigo_postal: "",
    pais: "España",
  });
  const [savingDireccion, setSavingDireccion] = useState(false);
  const [direccionMsg, setDireccionMsg] = useState<{ text: string; ok: boolean } | null>(null);

  // Contactos state
  const [contactos, setContactos] = useState<Contacto[]>([]);
  const [loadingContactos, setLoadingContactos] = useState(false);
  const [showContactoForm, setShowContactoForm] = useState(false);
  const [contactoForm, setContactoForm] = useState<Omit<Contacto, "id" | "entidad_id">>({
    nombre: "",
    puesto: "",
    email: "",
    telefono: "",
  });
  const [savingContacto, setSavingContacto] = useState(false);
  const [contactoMsg, setContactoMsg] = useState<{ text: string; ok: boolean } | null>(null);

  // Notas state
  const [notas, setNotas] = useState<Nota[]>([]);
  const [loadingNotas, setLoadingNotas] = useState(false);
  const [notaText, setNotaText] = useState("");
  const [savingNota, setSavingNota] = useState(false);
  const [notaMsg, setNotaMsg] = useState<{ text: string; ok: boolean } | null>(null);

  // Fetch entity by ID
  useEffect(() => {
    if (!id) return;
    fetchEntity(id);
  }, [id]);

  async function fetchEntity(entityId: string) {
    setLoading(true);
    setError("");
    try {
      const data: Entidad = await invoke("get_entidad_by_id", { id: entityId });
      setEntity(data);
    } catch (err: any) {
      setError(`Error al cargar entidad: ${err}`);
    } finally {
      setLoading(false);
    }
  }

  // Load data when tab changes
  useEffect(() => {
    if (!entity) return;
    if (activeTab === "direcciones" && direcciones.length === 0) {
      fetchDirecciones();
    } else if (activeTab === "contactos" && contactos.length === 0) {
      fetchContactos();
    } else if (activeTab === "notas" && notas.length === 0) {
      fetchNotas();
    }
  }, [activeTab, entity]);

  async function fetchDirecciones() {
    if (!entity) return;
    setLoadingDirecciones(true);
    try {
      const data: Direccion[] = await invoke("get_direcciones", { entidadId: entity.id });
      setDirecciones(data);
    } catch (err: any) {
      console.error("Error al cargar direcciones:", err);
    } finally {
      setLoadingDirecciones(false);
    }
  }

  async function fetchContactos() {
    if (!entity) return;
    setLoadingContactos(true);
    try {
      const data: Contacto[] = await invoke("get_contactos", { entidadId: entity.id });
      setContactos(data);
    } catch (err: any) {
      console.error("Error al cargar contactos:", err);
    } finally {
      setLoadingContactos(false);
    }
  }

  async function fetchNotas() {
    if (!entity) return;
    setLoadingNotas(true);
    try {
      const data: Nota[] = await invoke("get_notas", { entidadId: entity.id });
      const sorted = [...data].sort(
        (a, b) => new Date(b.creado_en).getTime() - new Date(a.creado_en).getTime()
      );
      setNotas(sorted);
    } catch (err: any) {
      console.error("Error al cargar notas:", err);
    } finally {
      setLoadingNotas(false);
    }
  }

  async function handleSaveDireccion(e: React.FormEvent) {
    e.preventDefault();
    if (!entity) return;
    setSavingDireccion(true);
    setDireccionMsg(null);
    const newDireccion: Direccion = {
      id: crypto.randomUUID(),
      entidad_id: entity.id,
      ...direccionForm,
    };
    try {
      await invoke("save_direccion", { direccion: newDireccion });
      setDirecciones((prev) => [...prev, newDireccion]);
      setDireccionForm({ tipo_direccion: "FISCAL", calle: "", ciudad: "", provincia: "", codigo_postal: "", pais: "España" });
      setShowDireccionForm(false);
      setDireccionMsg({ text: "Dirección guardada correctamente.", ok: true });
    } catch (err: any) {
      setDireccionMsg({ text: `Error: ${err}`, ok: false });
    } finally {
      setSavingDireccion(false);
    }
  }

  async function handleSaveContacto(e: React.FormEvent) {
    e.preventDefault();
    if (!entity) return;
    if (!contactoForm.nombre.trim()) {
      setContactoMsg({ text: "El nombre del contacto es obligatorio.", ok: false });
      return;
    }
    setSavingContacto(true);
    setContactoMsg(null);
    const newContacto: Contacto = {
      id: crypto.randomUUID(),
      entidad_id: entity.id,
      ...contactoForm,
    };
    try {
      await invoke("save_contacto", { contacto: newContacto });
      setContactos((prev) => [...prev, newContacto]);
      setContactoForm({ nombre: "", puesto: "", email: "", telefono: "" });
      setShowContactoForm(false);
      setContactoMsg({ text: "Contacto guardado correctamente.", ok: true });
    } catch (err: any) {
      setContactoMsg({ text: `Error: ${err}`, ok: false });
    } finally {
      setSavingContacto(false);
    }
  }

  async function handleSaveNota(e: React.FormEvent) {
    e.preventDefault();
    if (!entity) return;
    if (!notaText.trim()) {
      setNotaMsg({ text: "La nota no puede estar vacía.", ok: false });
      return;
    }
    setSavingNota(true);
    setNotaMsg(null);
    const newNota: Nota = {
      id: crypto.randomUUID(),
      entidad_id: entity.id,
      nota: notaText.trim(),
      creado_en: new Date().toISOString(),
    };
    try {
      await invoke("save_nota", { nota: newNota });
      setNotas((prev) => [newNota, ...prev]);
      setNotaText("");
      setNotaMsg({ text: "Nota guardada correctamente.", ok: true });
    } catch (err: any) {
      setNotaMsg({ text: `Error: ${err}`, ok: false });
    } finally {
      setSavingNota(false);
    }
  }

  if (loading) {
    return (
      <div style={pageLoadingStyle}>
        <p>Cargando entidad...</p>
      </div>
    );
  }

  if (error || !entity) {
    return (
      <div style={pageErrorStyle}>
        <p>{error || "Entidad no encontrada"}</p>
        <button onClick={() => navigate("/entities")} style={saveBtnStyle}>
          Volver a entidades
        </button>
      </div>
    );
  }

  const roles = parseRoles(entity.roles);

  return (
    <div style={pageContainerStyle}>
      {/* Header */}
      <div style={panelHeaderStyle}>
        <button onClick={() => navigate("/entities")} style={backButtonStyle}>
          ← Entidades
        </button>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: "flex", alignItems: "center", gap: "10px", flexWrap: "wrap" }}>
            <h2 style={panelTitleStyle}>{entity.nombre}</h2>
            {roles.map((role) => (
              <span key={role} style={roleBadgeStyle(role)}>{role}</span>
            ))}
            <span style={entity.activo ? activoBadgeStyle : inactivoBadgeStyle}>
              {entity.activo ? "Activo" : "Inactivo"}
            </span>
          </div>
          {entity.nif && (
            <p style={panelSubtitleStyle}>NIF: {entity.nif}</p>
          )}
        </div>
        <div style={{ display: "flex", gap: "8px" }}>
          {roles.includes("CLIENTE") && (
            <>
              <button
                onClick={() => navigate(`/clients/${entity.id}/dossier`)}
                style={actionBtnStyle}
              >
                📋 Dossier
              </button>
              <button
                onClick={() => navigate(`/clients/${entity.id}/collection`)}
                style={actionBtnAccentStyle}
              >
                💳 Cobro
              </button>
            </>
          )}
        </div>
      </div>

      {/* Tab navigation */}
      <div style={tabNavStyle}>
        {(["datos", "direcciones", "contactos", "notas"] as TabKey[]).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            style={{ ...tabButtonStyle, ...(activeTab === tab ? tabActiveStyle : {}) }}
          >
            {TAB_LABELS[tab]}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div style={tabContentStyle}>
        {/* DATOS TAB */}
        {activeTab === "datos" && (
          <div style={datosGridStyle}>
            <DataField label="Nombre" value={entity.nombre} />
            <DataField label="NIF" value={entity.nif || "—"} />
            <DataField label="Email" value={entity.email || "—"} />
            <DataField label="Estado" value={entity.activo ? "Activo" : "Inactivo"} />
            <DataField label="Roles" value={entity.roles || "—"} />
            <DataField label="Última actualización" value={new Date(entity.updated_at).toLocaleString("es-ES")} />
          </div>
        )}

        {/* DIRECCIONES TAB */}
        {activeTab === "direcciones" && (
          <div>
            {loadingDirecciones ? (
              <p style={loadingStyle}>🔄 Cargando...</p>
            ) : (
              <>
                {direcciones.length === 0 && !showDireccionForm && (
                  <div style={emptyStateStyle}>
                    <p>Sin direcciones registradas.</p>
                  </div>
                )}
                {direcciones.map((d) => (
                  <div key={d.id} style={cardItemStyle}>
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
                      <span style={typeTagStyle}>{d.tipo_direccion}</span>
                    </div>
                    <p style={addressLineStyle}>{d.calle}</p>
                    <p style={addressLineStyle}>{d.codigo_postal} {d.ciudad}, {d.provincia}</p>
                    <p style={addressLineStyle}>{d.pais}</p>
                  </div>
                ))}
                {showDireccionForm && (
                  <form onSubmit={handleSaveDireccion} style={inlineFormStyle}>
                    <h4 style={formTitleStyle}>Nueva Dirección</h4>
                    <div style={formGridStyle}>
                      <div style={formFieldStyle}>
                        <label style={labelStyle}>Tipo</label>
                        <select
                          value={direccionForm.tipo_direccion}
                          onChange={(e) => setDireccionForm((f) => ({ ...f, tipo_direccion: e.target.value }))}
                          style={selectStyle}
                        >
                          <option value="FISCAL">FISCAL</option>
                          <option value="ENVIO">ENVIO</option>
                          <option value="OTRA">OTRA</option>
                        </select>
                      </div>
                      <div style={formFieldStyle}>
                        <label style={labelStyle}>Calle</label>
                        <input
                          required
                          value={direccionForm.calle}
                          onChange={(e) => setDireccionForm((f) => ({ ...f, calle: e.target.value }))}
                          placeholder="Calle, número..."
                          style={fieldInputStyle}
                        />
                      </div>
                      <div style={formFieldStyle}>
                        <label style={labelStyle}>Ciudad</label>
                        <input
                          required
                          value={direccionForm.ciudad}
                          onChange={(e) => setDireccionForm((f) => ({ ...f, ciudad: e.target.value }))}
                          placeholder="Ciudad"
                          style={fieldInputStyle}
                        />
                      </div>
                      <div style={formFieldStyle}>
                        <label style={labelStyle}>Provincia</label>
                        <input
                          required
                          value={direccionForm.provincia}
                          onChange={(e) => setDireccionForm((f) => ({ ...f, provincia: e.target.value }))}
                          placeholder="Provincia"
                          style={fieldInputStyle}
                        />
                      </div>
                      <div style={formFieldStyle}>
                        <label style={labelStyle}>Código Postal</label>
                        <input
                          required
                          value={direccionForm.codigo_postal}
                          onChange={(e) => setDireccionForm((f) => ({ ...f, codigo_postal: e.target.value }))}
                          placeholder="28001"
                          style={fieldInputStyle}
                        />
                      </div>
                      <div style={formFieldStyle}>
                        <label style={labelStyle}>País</label>
                        <input
                          required
                          value={direccionForm.pais}
                          onChange={(e) => setDireccionForm((f) => ({ ...f, pais: e.target.value }))}
                          placeholder="España"
                          style={fieldInputStyle}
                        />
                      </div>
                    </div>
                    <div style={formActionsStyle}>
                      <button type="button" onClick={() => setShowDireccionForm(false)} style={cancelBtnStyle}>
                        Cancelar
                      </button>
                      <button type="submit" disabled={savingDireccion} style={saveBtnStyle}>
                        {savingDireccion ? "Guardando..." : "Guardar Dirección"}
                      </button>
                    </div>
                  </form>
                )}
                {direccionMsg && (
                  <div style={feedbackStyle(direccionMsg.ok)}>{direccionMsg.text}</div>
                )}
                {!showDireccionForm && (
                  <button onClick={() => { setShowDireccionForm(true); setDireccionMsg(null); }} style={addBtnStyle}>
                    + Agregar Dirección
                  </button>
                )}
              </>
            )}
          </div>
        )}

        {/* CONTACTOS TAB */}
        {activeTab === "contactos" && (
          <div>
            {loadingContactos ? (
              <p style={loadingStyle}>🔄 Cargando...</p>
            ) : (
              <>
                {contactos.length === 0 && !showContactoForm && (
                  <div style={emptyStateStyle}>
                    <p>Sin contactos registrados.</p>
                  </div>
                )}
                {contactos.map((c) => (
                  <div key={c.id} style={cardItemStyle}>
                    <p style={{ margin: "0 0 4px", fontWeight: "700", color: "var(--text-primary)" }}>{c.nombre}</p>
                    {c.puesto && <p style={metaTextStyle}>{c.puesto}</p>}
                    {c.email && <p style={metaTextStyle}>✉ {c.email}</p>}
                    {c.telefono && <p style={metaTextStyle}>📞 {c.telefono}</p>}
                  </div>
                ))}
                {showContactoForm && (
                  <form onSubmit={handleSaveContacto} style={inlineFormStyle}>
                    <h4 style={formTitleStyle}>Nuevo Contacto</h4>
                    <div style={formGridStyle}>
                      <div style={formFieldStyle}>
                        <label style={labelStyle}>Nombre *</label>
                        <input
                          required
                          value={contactoForm.nombre}
                          onChange={(e) => setContactoForm((f) => ({ ...f, nombre: e.target.value }))}
                          placeholder="Nombre completo"
                          style={fieldInputStyle}
                        />
                      </div>
                      <div style={formFieldStyle}>
                        <label style={labelStyle}>Puesto</label>
                        <input
                          value={contactoForm.puesto}
                          onChange={(e) => setContactoForm((f) => ({ ...f, puesto: e.target.value }))}
                          placeholder="Director, Gerente..."
                          style={fieldInputStyle}
                        />
                      </div>
                      <div style={formFieldStyle}>
                        <label style={labelStyle}>Email</label>
                        <input
                          type="email"
                          value={contactoForm.email}
                          onChange={(e) => setContactoForm((f) => ({ ...f, email: e.target.value }))}
                          placeholder="contacto@empresa.com"
                          style={fieldInputStyle}
                        />
                      </div>
                      <div style={formFieldStyle}>
                        <label style={labelStyle}>Teléfono</label>
                        <input
                          value={contactoForm.telefono}
                          onChange={(e) => setContactoForm((f) => ({ ...f, telefono: e.target.value }))}
                          placeholder="+34 600 000 000"
                          style={fieldInputStyle}
                        />
                      </div>
                    </div>
                    <div style={formActionsStyle}>
                      <button type="button" onClick={() => setShowContactoForm(false)} style={cancelBtnStyle}>
                        Cancelar
                      </button>
                      <button type="submit" disabled={savingContacto} style={saveBtnStyle}>
                        {savingContacto ? "Guardando..." : "Guardar Contacto"}
                      </button>
                    </div>
                  </form>
                )}
                {contactoMsg && (
                  <div style={feedbackStyle(contactoMsg.ok)}>{contactoMsg.text}</div>
                )}
                {!showContactoForm && (
                  <button onClick={() => { setShowContactoForm(true); setContactoMsg(null); }} style={addBtnStyle}>
                    + Agregar Contacto
                  </button>
                )}
              </>
            )}
          </div>
        )}

        {/* NOTAS TAB */}
        {activeTab === "notas" && (
          <div>
            {loadingNotas ? (
              <p style={loadingStyle}>🔄 Cargando...</p>
            ) : (
              <>
                <form onSubmit={handleSaveNota} style={{ marginBottom: "20px" }}>
                  <textarea
                    value={notaText}
                    onChange={(e) => setNotaText(e.target.value)}
                    placeholder="Escribe una nota sobre esta entidad..."
                    rows={4}
                    style={textareaStyle}
                  />
                  <div style={{ display: "flex", justifyContent: "flex-end", marginTop: "8px" }}>
                    <button type="submit" disabled={savingNota || !notaText.trim()} style={{
                      ...saveBtnStyle,
                      opacity: !notaText.trim() ? 0.5 : 1,
                      cursor: !notaText.trim() ? "not-allowed" : "pointer",
                    }}>
                      {savingNota ? "Guardando..." : "Guardar Nota"}
                    </button>
                  </div>
                </form>
                {notaMsg && (
                  <div style={feedbackStyle(notaMsg.ok)}>{notaMsg.text}</div>
                )}
                {notas.length === 0 ? (
                  <div style={emptyStateStyle}>
                    <p>Sin notas registradas.</p>
                  </div>
                ) : (
                  notas.map((n) => (
                    <div key={n.id} style={cardItemStyle}>
                      <p style={{ margin: "0 0 8px", color: "var(--text-secondary)", lineHeight: "1.6", whiteSpace: "pre-wrap" }}>{n.nota}</p>
                      <p style={{ margin: 0, fontSize: "11px", color: "var(--text-placeholder)" }}>
                        {new Date(n.creado_en).toLocaleString("es-ES")}
                      </p>
                    </div>
                  ))
                )}
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Helper components
// ---------------------------------------------------------------------------
function DataField({ label, value }: { label: string; value: string }) {
  return (
    <div style={dataFieldStyle}>
      <span style={dataLabelStyle}>{label}</span>
      <span style={dataValueStyle}>{value}</span>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
const TAB_LABELS: Record<TabKey, string> = {
  datos: "📋 Datos",
  direcciones: "📍 Direcciones",
  contactos: "👤 Contactos",
  notas: "📝 Notas",
};

function parseRoles(roles?: string): string[] {
  if (!roles) return [];
  return roles.split(",").map((r) => r.trim()).filter(Boolean);
}

function roleBadgeStyle(role: string): React.CSSProperties {
  const colors: Record<string, { bg: string; color: string }> = {
    CLIENTE: { bg: "var(--accent-default)", color: "#fff" },
    PROVEEDOR: { bg: "var(--color-warning)", color: "#fff" },
  };
  const c = colors[role] ?? { bg: "var(--accent-default)", color: "#fff" };
  return {
    backgroundColor: c.bg,
    color: c.color,
    fontSize: "11px",
    fontWeight: "700",
    padding: "2px 8px",
    borderRadius: "20px",
    letterSpacing: "0.05em",
    textTransform: "uppercase" as const,
    whiteSpace: "nowrap" as const,
  };
}

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------
const pageContainerStyle: React.CSSProperties = {
  maxWidth: "980px",
  margin: "0 auto",
  padding: "20px",
  fontFamily: "'Inter', system-ui, sans-serif",
};

const pageLoadingStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  minHeight: "50vh",
  color: "var(--text-muted)",
  fontSize: "16px",
};

const pageErrorStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  alignItems: "center",
  justifyContent: "center",
  minHeight: "50vh",
  gap: "16px",
  color: "var(--text-muted)",
};

const panelHeaderStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "flex-start",
  gap: "12px",
  padding: "20px 24px",
  borderBottom: "1px solid var(--border-default)",
  backgroundColor: "var(--bg-card)",
  borderRadius: "16px 16px 0 0",
  border: "1px solid var(--border-default)",
};

const backButtonStyle: React.CSSProperties = {
  background: "none",
  border: "1px solid var(--border-default)",
  borderRadius: "8px",
  padding: "8px 14px",
  fontSize: "13px",
  fontWeight: "600",
  color: "var(--text-secondary)",
  cursor: "pointer",
  whiteSpace: "nowrap",
  flexShrink: 0,
};

const panelTitleStyle: React.CSSProperties = {
  fontSize: "18px",
  fontWeight: "800",
  color: "var(--text-primary)",
  margin: 0,
  lineHeight: "1.2",
};

const panelSubtitleStyle: React.CSSProperties = {
  margin: "4px 0 0",
  fontSize: "13px",
  color: "var(--text-muted)",
};

const actionBtnStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-page)",
  color: "var(--accent-default)",
  border: "1px solid var(--border-input)",
  borderRadius: "8px",
  padding: "8px 14px",
  fontSize: "13px",
  fontWeight: "600",
  cursor: "pointer",
  whiteSpace: "nowrap",
};

const actionBtnAccentStyle: React.CSSProperties = {
  backgroundColor: "var(--accent-default)",
  color: "#fff",
  border: "none",
  borderRadius: "8px",
  padding: "8px 14px",
  fontSize: "13px",
  fontWeight: "600",
  cursor: "pointer",
  whiteSpace: "nowrap",
};

const tabNavStyle: React.CSSProperties = {
  display: "flex",
  borderBottom: "1px solid var(--border-default)",
  backgroundColor: "var(--bg-card)",
  overflowX: "auto",
  borderLeft: "1px solid var(--border-default)",
  borderRight: "1px solid var(--border-default)",
};

const tabButtonStyle: React.CSSProperties = {
  background: "none",
  border: "none",
  borderBottom: "2px solid transparent",
  padding: "12px 16px",
  fontSize: "13px",
  fontWeight: "500",
  color: "var(--text-muted)",
  cursor: "pointer",
  whiteSpace: "nowrap",
  transition: "color 0.15s, border-color 0.15s",
};

const tabActiveStyle: React.CSSProperties = {
  color: "var(--accent-default)",
  borderBottomColor: "var(--accent-default)",
  fontWeight: "700",
};

const tabContentStyle: React.CSSProperties = {
  flex: 1,
  overflowY: "auto",
  padding: "20px 24px",
  backgroundColor: "var(--bg-card)",
  border: "1px solid var(--border-default)",
  borderTop: "none",
  borderRadius: "0 0 16px 16px",
};

const datosGridStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: "12px",
};

const dataFieldStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: "2px",
  padding: "12px 14px",
  backgroundColor: "var(--bg-page)",
  borderRadius: "10px",
  border: "1px solid var(--border-default)",
};

const dataLabelStyle: React.CSSProperties = {
  fontSize: "11px",
  fontWeight: "700",
  color: "var(--text-placeholder)",
  textTransform: "uppercase",
  letterSpacing: "0.06em",
};

const dataValueStyle: React.CSSProperties = {
  fontSize: "15px",
  fontWeight: "600",
  color: "var(--text-primary)",
};

const activoBadgeStyle: React.CSSProperties = {
  backgroundColor: "var(--status-success-bg)",
  color: "var(--status-success-text)",
  fontSize: "11px",
  fontWeight: "700",
  padding: "2px 8px",
  borderRadius: "20px",
  letterSpacing: "0.05em",
};

const inactivoBadgeStyle: React.CSSProperties = {
  backgroundColor: "var(--status-error-bg)",
  color: "var(--status-error-text)",
  fontSize: "11px",
  fontWeight: "700",
  padding: "2px 8px",
  borderRadius: "20px",
  letterSpacing: "0.05em",
};

const loadingStyle: React.CSSProperties = {
  color: "var(--text-muted)",
  textAlign: "center",
  padding: "24px",
};

const emptyStateStyle: React.CSSProperties = {
  textAlign: "center",
  padding: "32px 20px",
  color: "var(--text-placeholder)",
  border: "2px dashed var(--border-default)",
  borderRadius: "12px",
  fontSize: "14px",
  marginBottom: "16px",
};

const cardItemStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-page)",
  border: "1px solid var(--border-default)",
  borderRadius: "10px",
  padding: "14px 16px",
  marginBottom: "10px",
};

const typeTagStyle: React.CSSProperties = {
  backgroundColor: "var(--status-info-bg)",
  color: "var(--status-info-text)",
  fontSize: "11px",
  fontWeight: "700",
  padding: "2px 8px",
  borderRadius: "4px",
  letterSpacing: "0.05em",
  marginBottom: "8px",
  display: "inline-block",
};

const addressLineStyle: React.CSSProperties = {
  margin: "4px 0 0",
  fontSize: "14px",
  color: "var(--text-secondary)",
};

const metaTextStyle: React.CSSProperties = {
  margin: "4px 0 0",
  fontSize: "13px",
  color: "var(--text-muted)",
};

const inlineFormStyle: React.CSSProperties = {
  backgroundColor: "var(--status-info-bg)",
  border: "1px solid var(--status-info-bg)",
  borderRadius: "12px",
  padding: "16px",
  marginBottom: "16px",
};

const formTitleStyle: React.CSSProperties = {
  margin: "0 0 14px",
  fontSize: "14px",
  fontWeight: "700",
  color: "var(--text-primary)",
};

const formGridStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "1fr 1fr",
  gap: "10px",
};

const formFieldStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: "4px",
};

const labelStyle: React.CSSProperties = {
  fontSize: "11px",
  fontWeight: "600",
  color: "var(--text-secondary)",
  textTransform: "uppercase",
  letterSpacing: "0.04em",
};

const fieldInputStyle: React.CSSProperties = {
  padding: "10px 14px",
  borderRadius: "8px",
  border: "1px solid var(--border-input)",
  fontSize: "14px",
  backgroundColor: "var(--bg-card)",
  outline: "none",
};

const selectStyle: React.CSSProperties = {
  ...fieldInputStyle,
  cursor: "pointer",
  backgroundColor: "var(--bg-card)",
};

const formActionsStyle: React.CSSProperties = {
  display: "flex",
  justifyContent: "flex-end",
  gap: "10px",
  marginTop: "14px",
};

const cancelBtnStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-page)",
  color: "var(--text-secondary)",
  border: "1px solid var(--border-input)",
  borderRadius: "8px",
  padding: "9px 16px",
  fontSize: "14px",
  fontWeight: "600",
  cursor: "pointer",
};

const saveBtnStyle: React.CSSProperties = {
  backgroundColor: "var(--accent-default)",
  color: "#fff",
  border: "none",
  borderRadius: "8px",
  padding: "9px 18px",
  fontSize: "14px",
  fontWeight: "600",
  cursor: "pointer",
};

const addBtnStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-card)",
  color: "var(--accent-default)",
  border: "2px dashed var(--accent-ring)",
  borderRadius: "10px",
  padding: "10px 16px",
  fontSize: "14px",
  fontWeight: "600",
  cursor: "pointer",
  width: "100%",
  marginTop: "4px",
};

const textareaStyle: React.CSSProperties = {
  width: "100%",
  padding: "10px 14px",
  borderRadius: "10px",
  border: "1px solid var(--border-input)",
  fontSize: "14px",
  resize: "vertical",
  fontFamily: "inherit",
  outline: "none",
  boxSizing: "border-box",
};

function feedbackStyle(ok: boolean): React.CSSProperties {
  return {
    padding: "10px 14px",
    borderRadius: "8px",
    fontSize: "13px",
    fontWeight: "500",
    marginBottom: "12px",
    backgroundColor: ok ? "var(--status-success-bg)" : "var(--status-error-bg)",
    color: ok ? "var(--status-success-text)" : "var(--status-error-text)",
    border: `1px solid ${ok ? "var(--status-success-bg)" : "var(--status-error-bg)"}`,
  };
}
