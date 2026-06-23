import { useEffect, useRef, useCallback } from "react";
import { invoke } from "@tauri-apps/api/core";
import { usePOS } from "./PosContext";
import type { POSProduct } from "./types";

export function FamilyNavBar() {
  const { state, dispatch } = usePOS();
  const scrollRef = useRef<HTMLDivElement>(null);

  // Load families on mount
  useEffect(() => {
    (async () => {
      try {
        const families: string[] = await invoke("get_families");
        dispatch({ type: "SET_FAMILIES", payload: families });
      } catch (err) {
        console.error("Failed to load families:", err);
      }
    })();
  }, [dispatch]);

  // Load products when family changes
  const handleFamilyClick = useCallback(
    async (family: string | null) => {
      dispatch({ type: "SET_SELECTED_FAMILY", payload: family });
      dispatch({ type: "SET_LOADING_PRODUCTS", payload: true });

      try {
        const products: POSProduct[] = family
          ? await invoke("get_products_by_family", { familia: family })
          : await invoke("get_products_by_family", { familia: null });
        dispatch({ type: "SET_PRODUCTS", payload: products });
      } catch (err) {
        console.error("Failed to load products:", err);
        dispatch({ type: "SET_PRODUCTS", payload: [] });
      }
    },
    [dispatch],
  );

  // Load all products initially
  useEffect(() => {
    handleFamilyClick(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const allFamilies = ["TODO", ...state.families];

  return (
    <div className="family-navbar" ref={scrollRef}>
      <div className="family-navbar-scroll">
        {allFamilies.map((f) => {
          const isActive =
            f === "TODO"
              ? state.selectedFamily === null
              : state.selectedFamily === f;
          return (
            <button
              key={f}
              className={`family-tab ${isActive ? "family-tab-active" : ""}`}
              onClick={() => handleFamilyClick(f === "TODO" ? null : f)}
            >
              {f === "TODO" ? "TODO" : f}
            </button>
          );
        })}
      </div>
    </div>
  );
}
