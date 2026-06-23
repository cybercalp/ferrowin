import { useState, useEffect, useRef } from "react";
import { invoke } from "@tauri-apps/api/core";
import { useNavigate } from "react-router";
import type { Entidad } from "./EntityDetailPanel";

export type { Entidad };

type RoleFilter = "all" | "cliente" | "proveedor" | "ambos";

export function EntityManager() {
  const navigate = useNavigate();
  const [entidades, setEntidades] = useState<Entidad[]>([]);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState("");
  const [roleFilter, setRoleFilter] = useState<RoleFilter>("all");
  const searchRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    fetchEntidades();
  }, []);

  async function fetchEntidades() {
    setLoading(true);
    try {
      const data: Entidad[] = await invoke("get_clientes");
      setEntidades(data);
    } catch (err: any) {
      console.error("Error al cargar entidades:", err);
    } finally {
      setLoading(false);
    }
  }

  const filtered = entidades.filter((e) => {
    const q = search.toLowerCase();
    const matchesSearch =
      !search ||
      e.nombre.toLowerCase().includes(q) ||
      (e.nif && e.nif.toLowerCase().includes(q)) ||
      (e.email && e.email.toLowerCase().includes(q));
    if (!matchesSearch) return false;
    if (roleFilter === "all") return true;
    const roles = (e.roles ?? "").toUpperCase();
    if (roleFilter === "cliente") return roles.includes("CLIENTE") && !roles.includes("PROVEEDOR");
    if (roleFilter === "proveedor") return roles.includes("PROVEEDOR") && !roles.includes("CLIENTE");
    if (roleFilter === "ambos") return roles.includes("CLIENTE") && roles.includes("PROVEEDOR");
    return true;
  });

  const TABS: { key: RoleFilter; label: string; icon: string; color: string }[] = [
    { key: "all",       label: "Todos",       icon: "⬡", color: "var(--text-muted)" },
    { key: "cliente",   label: "Clientes",    icon: "●", color: "var(--accent-default)" },
    { key: "proveedor", label: "Proveedores", icon: "●", color: "var(--color-warning)" },
    { key: "ambos",     label: "Ambos roles", icon: "●", color: "#7c3aed" },
  ];

  const counts = {
    all: entidades.length,
    cliente: entidades.filter(e => { const r = (e.roles ?? "").toUpperCase(); return r.includes("CLIENTE") && !r.includes("PROVEEDOR"); }).length,
    proveedor: entidades.filter(e => { const r = (e.roles ?? "").toUpperCase(); return r.includes("PROVEEDOR") && !r.includes("CLIENTE"); }).length,
    ambos: entidades.filter(e => { const r = (e.roles ?? "").toUpperCase(); return r.includes("CLIENTE") && r.includes("PROVEEDOR"); }).length,
  };

  return (
    <div style={rootStyle}>
      {/* ── HERO HEADER ─────────────────────────────────────────────────── */}
      <div style={heroStyle}>
        <div style={heroInnerStyle}>
          <div style={{ display: "flex", alignItems: "center", gap: "12px" }}>
            <button onClick={() => navigate("/")} style={backBtnStyle} title="Volver al POS">
              ← POS
            </button>
            <div>
              <h1 style={heroTitleStyle}>Gestión de Entidades</h1>
              <p style={heroSubtitleStyle}>Clientes, proveedores y contactos de tu empresa</p>
            </div>
          </div>
          <button onClick={fetchEntidades} disabled={loading} style={refreshBtnStyle} title="Actualizar lista">
            <span style={{ display: "inline-block", animation: loading ? "spin 1s linear infinite" : "none" }}>↻</span>
            &nbsp;Actualizar
          </button>
        </div>

        {/* Search bar floats inside the hero */}
        <div style={searchBarStyle}>
          <span style={searchIconStyle}>⌕</span>
          <input
            ref={searchRef}
            type="text"
            placeholder="Buscar por nombre, NIF o email..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={searchInputStyle}
          />
          {search && (
            <button onClick={() => { setSearch(""); searchRef.current?.focus(); }} style={clearBtnStyle}>
              ✕
            </button>
          )}
        </div>
      </div>

      {/* ── ROLE FILTER PILLS ──────────────────────────────────────────── */}
      <div style={filterBarStyle}>
        <div style={filterPillsStyle}>
          {TABS.map((tab) => {
            const active = roleFilter === tab.key;
            return (
              <button
                key={tab.key}
                onClick={() => setRoleFilter(tab.key)}
                style={{
                  ...filterPillBase,
                  background: active ? tab.color : "transparent",
                  color: active ? "#fff" : "var(--text-muted)",
                  borderColor: active ? tab.color : "var(--border-default)",
                  boxShadow: active ? `0 2px 12px ${tab.color}55` : "none",
                  transform: active ? "translateY(-1px)" : "none",
                }}
              >
                {tab.icon}&nbsp;{tab.label}
                <span style={{
                  ...countChipStyle,
                  background: active ? "rgba(255,255,255,0.25)" : "var(--bg-page)",
                  color: active ? "#fff" : "var(--text-placeholder)",
                }}>
                  {counts[tab.key]}
                </span>
              </button>
            );
          })}
        </div>
        <span style={resultsLabelStyle}>
          {filtered.length === entidades.length
            ? `${filtered.length} entidades`
            : `${filtered.length} de ${entidades.length}`}
        </span>
      </div>

      {/* ── TABLE ──────────────────────────────────────────────────────── */}
      <div style={tableCardStyle}>
        {loading ? (
          <div style={skeletonWrapStyle}>
            {[...Array(5)].map((_, i) => (
              <div key={i} style={{ ...skeletonRowStyle, opacity: 1 - i * 0.15 }} />
            ))}
          </div>
        ) : filtered.length === 0 ? (
          <div style={emptyStateStyle}>
            <p style={{ fontSize: "48px", margin: "0 0 12px" }}>🔍</p>
            <p style={{ fontWeight: "700", fontSize: "16px", color: "var(--text-secondary)", margin: "0 0 4px" }}>
              Sin resultados
            </p>
            <p style={{ color: "var(--text-placeholder)", fontSize: "14px", margin: 0 }}>
              {search || roleFilter !== "all"
                ? "Ninguna entidad coincide con los filtros activos."
                : "No hay entidades registradas todavía."}
            </p>
          </div>
        ) : (
          <table style={tableStyle}>
            <thead>
              <tr>
                <th style={thStyle}>Entidad</th>
                <th style={thStyle}>NIF</th>
                <th style={thStyle}>Email</th>
                <th style={thStyle}>Roles</th>
                <th style={{ ...thStyle, textAlign: "center" }}>Estado</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((e) => (
                <EntityRow
                  key={e.id}
                  entity={e}
                  onClick={() => navigate(`/entities/${e.id}`)}
                />
              ))}
            </tbody>
          </table>
        )}
      </div>

      <style>{`
        @keyframes spin { to { transform: rotate(360deg); } }
        @keyframes fadeInRow { from { opacity: 0; transform: translateX(-8px); } to { opacity: 1; transform: none; } }
        @keyframes shimmer {
          0%   { background-position: -400px 0; }
          100% { background-position: 400px 0; }
        }
      `}</style>
    </div>
  );
}

// ── Row sub-component ──────────────────────────────────────────────────────
function EntityRow({
  entity,
  onClick,
}: {
  entity: Entidad;
  onClick: () => void;
}) {
  const [hovered, setHovered] = useState(false);
  const roles = parseRoles(entity.roles);

  return (
    <tr
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        borderBottom: "1px solid var(--border-default)",
        cursor: "pointer",
        backgroundColor: hovered ? "var(--bg-page)" : "transparent",
        transition: "background-color 0.12s",
        animation: "fadeInRow 0.2s ease both",
      }}
    >
      {/* Entity name + avatar */}
      <td style={{ ...tdStyle, fontWeight: "600" }}>
        <div style={{ display: "flex", alignItems: "center", gap: "12px" }}>
          <div style={avatarStyle(entity.nombre)}>
            {entity.nombre.charAt(0).toUpperCase()}
          </div>
          <span style={{ color: "var(--text-primary)" }}>{entity.nombre}</span>
        </div>
      </td>
      <td style={{ ...tdStyle, color: "var(--text-muted)", fontFamily: "monospace", fontSize: "13px" }}>
        {entity.nif || <span style={nullStyle}>—</span>}
      </td>
      <td style={{ ...tdStyle, color: "var(--text-muted)", fontSize: "13px" }}>
        {entity.email || <span style={nullStyle}>—</span>}
      </td>
      <td style={tdStyle}>
        <div style={{ display: "flex", gap: "5px", flexWrap: "wrap" }}>
          {roles.length === 0
            ? <span style={nullStyle}>—</span>
            : roles.map((r) => <RolePill key={r} role={r} />)}
        </div>
      </td>
      <td style={{ ...tdStyle, textAlign: "center" }}>
        <span style={entity.activo ? activoPillStyle : inactivoPillStyle}>
          {entity.activo ? "● Activo" : "○ Inactivo"}
        </span>
      </td>
    </tr>
  );
}

// ── Role pill ──────────────────────────────────────────────────────────────
function RolePill({ role }: { role: string }) {
  const MAP: Record<string, { bg: string; fg: string; glow: string }> = {
    CLIENTE:   { bg: "var(--status-info-bg)", fg: "var(--accent-default)", glow: "var(--accent-ring)" },
    PROVEEDOR: { bg: "var(--status-warning-bg)", fg: "var(--color-warning)", glow: "var(--color-warning)" },
  };
  const c = MAP[role] ?? { bg: "var(--status-info-bg)", fg: "var(--accent-default)", glow: "var(--accent-ring)" };
  return (
    <span style={{
      backgroundColor: c.bg,
      color: c.fg,
      fontSize: "11px",
      fontWeight: "700",
      padding: "3px 10px",
      borderRadius: "20px",
      letterSpacing: "0.04em",
      whiteSpace: "nowrap",
    }}>
      {role}
    </span>
  );
}

// ── Helpers ────────────────────────────────────────────────────────────────
function parseRoles(roles?: string): string[] {
  if (!roles) return [];
  return roles.split(",").map((r) => r.trim()).filter(Boolean);
}

const AVATAR_COLORS = [
  ["#3b82f6","#1d4ed8"],
  ["#8b5cf6","#6d28d9"],
  ["#ec4899","#be185d"],
  ["#14b8a6","#0f766e"],
  ["#f59e0b","#b45309"],
  ["#10b981","#047857"],
];
function avatarStyle(name: string): React.CSSProperties {
  const [from, to] = AVATAR_COLORS[name.charCodeAt(0) % AVATAR_COLORS.length];
  return {
    width: "34px",
    height: "34px",
    borderRadius: "10px",
    background: `linear-gradient(135deg, ${from}, ${to})`,
    color: "#fff",
    fontWeight: "800",
    fontSize: "15px",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    flexShrink: 0,
    boxShadow: `0 2px 8px ${from}66`,
  };
}

// ── Styles ─────────────────────────────────────────────────────────────────
const rootStyle: React.CSSProperties = {
  fontFamily: "'Inter', system-ui, sans-serif",
  maxWidth: "980px",
  margin: "0 auto 40px",
  textAlign: "left",
};

const heroStyle: React.CSSProperties = {
  background: "var(--gradient-hero)",
  borderRadius: "20px",
  padding: "32px 32px 0",
  marginBottom: "20px",
  position: "relative",
  overflow: "visible",
};

const heroInnerStyle: React.CSSProperties = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "flex-start",
  marginBottom: "24px",
};

const heroTitleStyle: React.CSSProperties = {
  fontSize: "26px",
  fontWeight: "800",
  color: "#fff",
  margin: "0 0 4px",
  letterSpacing: "-0.02em",
};

const heroSubtitleStyle: React.CSSProperties = {
  fontSize: "14px",
  color: "rgba(255,255,255,0.55)",
  margin: 0,
};

const refreshBtnStyle: React.CSSProperties = {
  backgroundColor: "rgba(255,255,255,0.12)",
  color: "#fff",
  border: "1px solid rgba(255,255,255,0.2)",
  borderRadius: "10px",
  padding: "9px 16px",
  fontWeight: "600",
  fontSize: "13px",
  cursor: "pointer",
  backdropFilter: "blur(6px)",
  whiteSpace: "nowrap",
  transition: "background 0.15s",
};

const backBtnStyle: React.CSSProperties = {
  backgroundColor: "rgba(255,255,255,0.15)",
  color: "#fff",
  border: "1px solid rgba(255,255,255,0.25)",
  borderRadius: "10px",
  padding: "9px 16px",
  fontWeight: "600",
  fontSize: "13px",
  cursor: "pointer",
  backdropFilter: "blur(6px)",
  whiteSpace: "nowrap",
  transition: "background 0.15s",
};

const searchBarStyle: React.CSSProperties = {
  position: "relative",
  display: "flex",
  alignItems: "center",
  backgroundColor: "rgba(255,255,255,0.10)",
  border: "1px solid rgba(255,255,255,0.18)",
  borderRadius: "12px 12px 0 0",
  backdropFilter: "blur(8px)",
  marginLeft: "-32px",
  marginRight: "-32px",
  padding: "0 16px",
};

const searchIconStyle: React.CSSProperties = {
  color: "rgba(255,255,255,0.5)",
  fontSize: "20px",
  marginRight: "8px",
  flexShrink: 0,
  userSelect: "none",
};

const searchInputStyle: React.CSSProperties = {
  flex: 1,
  border: "none",
  background: "transparent",
  padding: "16px 0",
  fontSize: "15px",
  color: "#fff",
  outline: "none",
};

const clearBtnStyle: React.CSSProperties = {
  background: "none",
  border: "none",
  color: "rgba(255,255,255,0.45)",
  cursor: "pointer",
  fontSize: "13px",
  padding: "4px 8px",
  borderRadius: "6px",
};

const filterBarStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  flexWrap: "wrap",
  gap: "12px",
  marginBottom: "16px",
};

const filterPillsStyle: React.CSSProperties = {
  display: "flex",
  gap: "8px",
  flexWrap: "wrap",
};

const filterPillBase: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: "6px",
  border: "1px solid",
  borderRadius: "999px",
  padding: "7px 14px",
  fontSize: "13px",
  fontWeight: "600",
  cursor: "pointer",
  transition: "all 0.18s",
};

const countChipStyle: React.CSSProperties = {
  borderRadius: "999px",
  fontSize: "11px",
  fontWeight: "700",
  padding: "1px 7px",
  minWidth: "20px",
  textAlign: "center",
};

const resultsLabelStyle: React.CSSProperties = {
  fontSize: "13px",
  color: "var(--text-placeholder)",
  fontWeight: "500",
};

const tableCardStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-card)",
  border: "1px solid var(--border-default)",
  borderRadius: "16px",
  overflow: "hidden",
  boxShadow: "var(--shadow-md)",
};

const skeletonWrapStyle: React.CSSProperties = {
  padding: "12px",
  display: "flex",
  flexDirection: "column",
  gap: "8px",
};

const skeletonRowStyle: React.CSSProperties = {
  height: "52px",
  borderRadius: "10px",
  background: "linear-gradient(90deg, var(--bg-page) 25%, var(--border-default) 50%, var(--bg-page) 75%)",
  backgroundSize: "800px 100%",
  animation: "shimmer 1.4s infinite linear",
};

const emptyStateStyle: React.CSSProperties = {
  textAlign: "center",
  padding: "72px 20px",
};

const tableStyle: React.CSSProperties = {
  width: "100%",
  borderCollapse: "collapse",
  fontSize: "14px",
};

const thStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-page)",
  color: "var(--text-muted)",
  fontWeight: "700",
  padding: "14px 20px",
  textAlign: "left",
  borderBottom: "1px solid var(--border-default)",
  fontSize: "11px",
  textTransform: "uppercase",
  letterSpacing: "0.06em",
};

const tdStyle: React.CSSProperties = {
  padding: "14px 20px",
  verticalAlign: "middle",
};

const nullStyle: React.CSSProperties = {
  color: "var(--border-input)",
};

const activoPillStyle: React.CSSProperties = {
  backgroundColor: "var(--status-success-bg)",
  color: "var(--status-success-text)",
  fontSize: "11px",
  fontWeight: "700",
  padding: "4px 11px",
  borderRadius: "999px",
};

const inactivoPillStyle: React.CSSProperties = {
  backgroundColor: "var(--bg-page)",
  color: "var(--text-placeholder)",
  fontSize: "11px",
  fontWeight: "700",
  padding: "4px 11px",
  borderRadius: "999px",
};
