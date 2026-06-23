import { useState, useEffect, useRef, useCallback } from "react";
import { invoke } from "@tauri-apps/api/core";
import { usePOS } from "./PosContext";
import type { POSProduct } from "./types";

const DEBOUNCE_MS = 300;

/*
 * Barcode scanner detection
 *
 * A USB wedge scanner sends keystrokes extremely fast (<15ms between keys)
 * followed by Enter. Human typing is much slower per-key (>80ms).
 *
 * Strategy: track inter-key timing globally. When we see a burst of rapid
 * keystrokes, accumulate them in a buffer. On Enter, if the buffer has
 * enough chars, treat it as a barcode code and look up the product.
 *
 * Individual keystrokes are NOT prevented from reaching the focused element.
 * This means scanner characters may briefly appear in whatever input has
 * focus — but the product is correctly added to the cart, and the next
 * interaction clears them. This is the standard behavior in POS systems.
 */

let globalBuf = "";
let globalLastKeyTime = 0;
let globalHandleCode: ((code: string) => void) | null = null;

function processGlobalCode(code: string) {
  if (globalHandleCode) {
    globalHandleCode(code);
    globalBuf = "";
  }
}

if (typeof window !== "undefined") {
  window.addEventListener("keydown", (e: KeyboardEvent) => {
    const now = Date.now();
    const elapsed = now - globalLastKeyTime;

    if (e.key === "Enter") {
      if (globalBuf.length >= 4) {
        e.preventDefault();
        processGlobalCode(globalBuf);
        return;
      }
      globalBuf = "";
    } else if (e.key.length === 1 && elapsed < 50 && elapsed > 0) {
      globalBuf += e.key;
    } else if (e.key.length === 1) {
      globalBuf = e.key;
    } else if (e.key === "Escape") {
      globalBuf = "";
    }

    globalLastKeyTime = now;
  }, true);
}

export function BarcodeInput() {
  const { dispatch } = usePOS();
  const [value, setValue] = useState("");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [searching, setSearching] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearError = useCallback(() => setErrorMsg(null), []);

  // Register/unregister the global barcode handler
  useEffect(() => {
    const handler = async (code: string) => {
      try {
        const product: POSProduct | null = await invoke(
          "get_product_by_code",
          { codigo: code },
        );
        if (product) {
          dispatch({ type: "ADD_TO_CART", payload: product });
          setValue("");
          setErrorMsg(null);
        } else {
          setErrorMsg(`Código no encontrado: ${code}`);
        }
      } catch (err) {
        console.error("Barcode lookup failed:", err);
        setErrorMsg(`Error al buscar código: ${code}`);
      }
    };
    globalHandleCode = handler;
    return () => {
      globalHandleCode = null;
    };
  }, [dispatch]);

  // F5 shortcut to focus the barcode input
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "F5") {
        e.preventDefault();
        inputRef.current?.focus();
        inputRef.current?.select();
      }
    };
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, []);

  // Submit manually typed code (from the input field)
  const handleSubmitCode = useCallback(
    async (code: string) => {
      dispatch({ type: "SET_SEARCH_QUERY", payload: code });
      try {
        const product: POSProduct | null = await invoke(
          "get_product_by_code",
          { codigo: code },
        );
        if (product) {
          dispatch({ type: "ADD_TO_CART", payload: product });
          setValue("");
          setErrorMsg(null);
          return;
        }
        // Not a code — try as a search query
        const results: POSProduct[] = await invoke("search_products", {
          query: code,
        });
        if (results.length === 1) {
          dispatch({ type: "ADD_TO_CART", payload: results[0] });
          setValue("");
          setErrorMsg(null);
        } else if (results.length > 1) {
          dispatch({ type: "SET_SEARCH_RESULTS", payload: results });
        } else {
          setErrorMsg(`Sin resultados: ${code}`);
        }
      } catch (err) {
        console.error("Lookup failed:", err);
        setErrorMsg(`Error al buscar: ${code}`);
      }
    },
    [dispatch],
  );

  // Keep searchQuery in sync with input value so ProductList can detect
  // search mode and show searchResults instead of family products.
  useEffect(() => {
    dispatch({ type: "SET_SEARCH_QUERY", payload: value.trim() });
  }, [value, dispatch]);

  // Debounced FTS5 search when user types in the input field
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);

    const trimmed = value.trim();
    if (trimmed.length <= 2) {
      dispatch({ type: "SET_SEARCH_RESULTS", payload: [] });
      return;
    }

    debounceRef.current = setTimeout(async () => {
      setSearching(true);
      try {
        const results: POSProduct[] = await invoke("search_products", {
          query: trimmed,
        });
        dispatch({ type: "SET_SEARCH_RESULTS", payload: results });
      } catch (err) {
        console.error("Product search failed:", err);
        dispatch({ type: "SET_SEARCH_RESULTS", payload: [] });
      } finally {
        setSearching(false);
      }
    }, DEBOUNCE_MS);

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [value, dispatch]);

  return (
    <div className="barcode-input-wrapper">
      <input
        ref={inputRef}
        type="text"
        className="barcode-input"
        value={value}
        onChange={(e) => {
          setValue(e.target.value);
          clearError();
        }}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            const trimmed = value.trim();
            if (trimmed) {
              e.preventDefault();
              handleSubmitCode(trimmed);
            }
          }
          if (e.key === "Escape") {
            setValue("");
            setErrorMsg(null);
          }
        }}
        placeholder="Código de barras o nombre... (F5)"
        autoFocus
        autoComplete="off"
        spellCheck={false}
      />
      {searching && <span className="barcode-spinner" />}
      {errorMsg && <span className="barcode-error">{errorMsg}</span>}
    </div>
  );
}
