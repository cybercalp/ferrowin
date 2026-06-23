import { useState, useRef, useEffect, useCallback } from "react";
import { usePOS } from "./PosContext";

export function CustomerSelector() {
  const { state, dispatch } = usePOS();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);

  // Close dropdown on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const handleClear = useCallback(() => {
    dispatch({ type: "SET_CUSTOMER", payload: null });
    dispatch({ type: "SET_DOCUMENT_TYPE", payload: "ticket" });
    setOpen(false);
  }, [dispatch]);

  const hasCustomer = state.customer !== null;

  return (
    <div className="customer-selector" ref={containerRef}>
      <button
        className={`customer-selector-btn ${hasCustomer ? "customer-active" : ""}`}
        onClick={() => setOpen(!open)}
        title="Seleccionar cliente"
      >
        <span className="customer-btn-label">Cliente:</span>
        {hasCustomer ? (
          <>
            <span className="customer-btn-name">
              {state.customer!.nombre}
            </span>
            {state.customer!.descuento > 0 && (
              <span className="customer-discount-badge">
                -{state.customer!.descuento}%
              </span>
            )}
          </>
        ) : (
          <span className="customer-btn-placeholder">Seleccionar</span>
        )}
      </button>

      {open && (
        <div className="customer-dropdown">
          <div className="customer-dropdown-search">
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Buscar por nombre o NIF..."
              autoFocus
              autoComplete="off"
            />
          </div>
          <div className="customer-dropdown-list">
            <CustomerList
              search={search}
              onSelect={(c) => {
                dispatch({ type: "SET_CUSTOMER", payload: c });
                dispatch({ type: "SET_DOCUMENT_TYPE", payload: "factura" });
                setOpen(false);
                setSearch("");
              }}
            />
          </div>
          {hasCustomer && (
            <button
              className="customer-dropdown-clear"
              onClick={handleClear}
            >
              Quitar cliente
            </button>
          )}
        </div>
      )}
    </div>
  );
}

/* ---------------------------------------------------------------------------
   Inline customer search + list
   ------------------------------------------------------------------------- */
import { invoke } from "@tauri-apps/api/core";
import type { CustomerInfo } from "./types";

function CustomerList({
  search,
  onSelect,
}: {
  search: string;
  onSelect: (c: CustomerInfo) => void;
}) {
  const [customers, setCustomers] = useState<CustomerInfo[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!search.trim()) {
      setCustomers([]);
      return;
    }
    const t = setTimeout(async () => {
      setLoading(true);
      try {
        const results: CustomerInfo[] = await invoke("search_clients", {
          query: search,
        });
        setCustomers(results);
      } catch (err) {
        console.error("Customer search failed:", err);
        setCustomers([]);
      } finally {
        setLoading(false);
      }
    }, 250);
    return () => clearTimeout(t);
  }, [search]);

  if (loading) {
    return <div className="customer-list-loading">Buscando...</div>;
  }

  if (customers.length === 0) {
    return search.trim() ? (
      <div className="customer-list-empty">Sin resultados</div>
    ) : (
      <div className="customer-list-empty">
        Escribe para buscar clientes
      </div>
    );
  }

  return (
    <>
      {customers.map((c) => (
        <button
          key={c.id}
          className="customer-list-item"
          onClick={() => onSelect(c)}
        >
          <span className="customer-list-name">{c.nombre}</span>
          {c.nif && <span className="customer-list-nif">{c.nif}</span>}
          {c.descuento > 0 && (
            <span className="customer-list-discount">-{c.descuento}%</span>
          )}
        </button>
      ))}
    </>
  );
}
