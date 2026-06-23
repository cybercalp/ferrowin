import { useState, useEffect, useRef, useCallback } from "react";
import { invoke } from "@tauri-apps/api/core";
import { usePOS } from "./PosContext";
import type { POSProduct } from "./types";

const DEBOUNCE_MS = 300;
const BARCODE_GAP_MS = 100;

export function ProductSearchBar() {
  const { state, dispatch } = usePOS();
  const [localQuery, setLocalQuery] = useState(state.searchQuery);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastKeyTimeRef = useRef(0);
  const barcodeBufRef = useRef("");

  // Sync local query when state changes from outside
  useEffect(() => {
    setLocalQuery(state.searchQuery);
  }, [state.searchQuery]);

  // Debounced search via Tauri IPC
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);

    if (!localQuery.trim()) {
      dispatch({ type: "SET_SEARCH_RESULTS", payload: [] });
      return;
    }

    debounceRef.current = setTimeout(async () => {
      dispatch({ type: "SET_IS_SEARCHING", payload: true });
      dispatch({ type: "SET_SEARCH_QUERY", payload: localQuery });
      try {
        const results: POSProduct[] = await invoke("search_products", {
          query: localQuery,
        });
        dispatch({ type: "SET_SEARCH_RESULTS", payload: results });
      } catch (err) {
        console.error("Product search failed:", err);
        dispatch({ type: "SET_SEARCH_RESULTS", payload: [] });
      }
    }, DEBOUNCE_MS);

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [localQuery, dispatch]);

  // Barcode wedge: rapid keystrokes followed by Enter
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      const now = Date.now();
      const elapsed = now - lastKeyTimeRef.current;

      if (elapsed < BARCODE_GAP_MS && e.key.length === 1) {
        barcodeBufRef.current += e.key;
      } else if (e.key === "Enter" && barcodeBufRef.current.length > 0) {
        e.preventDefault();
        const code = barcodeBufRef.current;
        barcodeBufRef.current = "";
        setLocalQuery(code);
      } else if (e.key.length === 1) {
        barcodeBufRef.current = e.key;
      } else {
        barcodeBufRef.current = "";
      }

      lastKeyTimeRef.current = now;
    },
    [],
  );

  return (
    <div className="product-search-bar">
      <input
        type="text"
        className="search-input"
        value={localQuery}
        onChange={(e) => setLocalQuery(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="Buscar producto por nombre, código o código de barras..."
        autoFocus
      />
      {state.isSearching && <span className="search-spinner">⟳</span>}
    </div>
  );
}
