import { useState } from "react";
import { usePOS } from "./PosContext";
import type { POSProduct } from "./types";
import type { POSAction } from "./PosContext";
import { useColumnResize } from "./useColumnResize";
import type { ColResizeApi } from "./useColumnResize";

// ── Shared thead ────────────────────────────────────────────────────────────
function TableHead({ api: { getStyle, createHandler } }: { api: ColResizeApi }) {
  return (
    <thead>
      <tr>
        <th className="col-imagen" style={getStyle("imagen")}>
          <div className="col-resize-handle" onMouseDown={createHandler("imagen")} />
        </th>
        <th className="col-codigo" style={getStyle("codigo")}>
          Codigo
          <div className="col-resize-handle" onMouseDown={createHandler("codigo")} />
        </th>
        <th className="col-nombre">
          Nombre
          <div className="col-resize-handle" onMouseDown={createHandler("nombre")} />
        </th>
        <th className="col-ud" style={getStyle("ud")}>
          Ud.
          <div className="col-resize-handle" onMouseDown={createHandler("ud")} />
        </th>
        <th className="col-precio" style={getStyle("precio")}>
          Precio
          <div className="col-resize-handle" onMouseDown={createHandler("precio")} />
        </th>
        <th className="col-add" style={getStyle("add")} />
      </tr>
    </thead>
  );
}

// ── Shared row ──────────────────────────────────────────────────────────────
function ProductRow({
  product,
  dispatch,
}: {
  product: POSProduct;
  dispatch: React.Dispatch<POSAction>;
}) {
  const [imgError, setImgError] = useState(false);
  const showImg = !!product.imagen_url && !imgError;

  return (
    <tr
      className="product-list-row"
      onClick={() =>
        dispatch({ type: "ADD_TO_CART", payload: product })
      }
    >
      <td className="col-imagen">
        {showImg ? (
          <img
            className="product-thumb"
            src={product.imagen_url!}
            alt={product.nombre}
            onError={() => setImgError(true)}
          />
        ) : (
          <div className="product-thumb-placeholder" />
        )}
      </td>
      <td className="col-codigo">
        <span className="product-code">{product.codigo}</span>
      </td>
      <td className="col-nombre">
        <span className="product-name">{product.nombre}</span>
      </td>
      <td className="col-ud">
        <span className="product-ud-badge">
          {product.unidad_medida || "ud"}
        </span>
      </td>
      <td className="col-precio">
        <span className="product-price">
          {product.precio_venta.toFixed(4)} €
        </span>
      </td>
      <td className="col-add">
        <button
          className="product-add-btn"
          onClick={(e) => {
            e.stopPropagation();
            dispatch({
              type: "ADD_TO_CART",
              payload: product,
            });
          }}
          title="Agregar al carrito"
        >
          +
        </button>
      </td>
    </tr>
  );
}

// ── Grid/card mode (mode táctil) ────────────────────────────────────────────
function ProductGrid({
  products,
  dispatch,
  gridColumns,
}: {
  products: POSProduct[];
  dispatch: React.Dispatch<POSAction>;
  gridColumns: number;
}) {
  return (
    <div
      className="product-grid"
      style={{ gridTemplateColumns: `repeat(${gridColumns}, 1fr)` }}
    >
      {products.map((product) => (
        <GridCard key={product.id} product={product} dispatch={dispatch} />
      ))}
    </div>
  );
}

function GridCard({
  product,
  dispatch,
}: {
  product: POSProduct;
  dispatch: React.Dispatch<POSAction>;
}) {
  const [imgError, setImgError] = useState(false);
  const showImg = !!product.imagen_url && !imgError;

  return (
    <div
      className="product-card"
      onClick={() =>
        dispatch({ type: "ADD_TO_CART", payload: product })
      }
    >
      <div className="product-card-img">
        {showImg ? (
          <img src={product.imagen_url!} alt={product.nombre} onError={() => setImgError(true)} />
        ) : (
          <div className="product-card-img-placeholder">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <rect x="3" y="3" width="18" height="18" rx="2" />
              <circle cx="8.5" cy="8.5" r="1.5" />
              <path d="m21 15-5-5L5 21" />
            </svg>
          </div>
        )}
      </div>
      <div className="product-card-info">
        <span className="product-card-name">{product.nombre}</span>
        <span className="product-card-price">
          {product.precio_venta.toFixed(4)} €
        </span>
      </div>
    </div>
  );
}

// ── Main component ──────────────────────────────────────────────────────────
export function ProductList() {
  const { state, dispatch } = usePOS();
  const colResize = useColumnResize();
  const {
    products,
    searchResults,
    searchQuery,
    isLoadingProducts,
    isSearching,
    selectedFamily,
    viewMode,
  } = state;

  const isSearchActive = searchQuery.trim().length > 2;

  // ── Loading state ──
  if (isLoadingProducts) {
    return (
      <div className="product-list-loading">
        <div className="loading-spinner" />
        <span>Cargando productos...</span>
      </div>
    );
  }

  // Helper — renders the products in either table or grid mode
  function renderItems(items: POSProduct[]) {
    if (viewMode === "grid") {
      return <ProductGrid products={items} dispatch={dispatch} gridColumns={state.gridColumns} />;
    }

    return (
      <div className="product-list-table-wrapper">
        <table className="product-list-table">
          <colgroup>
            <col style={colResize.getStyle("imagen")} />
            <col style={colResize.getStyle("codigo")} />
            <col /> {/* nombre — flexible */}
            <col style={colResize.getStyle("ud")} />
            <col style={colResize.getStyle("precio")} />
            <col style={colResize.getStyle("add")} />
          </colgroup>
          <TableHead api={colResize} />
          <tbody>
            {items.map((product) => (
              <ProductRow
                key={product.id}
                product={product}
                dispatch={dispatch}
              />
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  // ── Search mode ──
  if (isSearchActive || isSearching) {
    const items = searchResults;

    if (isSearching) {
      return (
        <div className="product-list-loading">
          <div className="loading-spinner" />
          <span>Buscando &ldquo;{searchQuery}&rdquo;&hellip;</span>
        </div>
      );
    }

    if (items.length === 0) {
      return (
        <div className="product-list-empty">
          Sin resultados para &ldquo;{searchQuery}&rdquo;
        </div>
      );
    }

    return (
      <div className="product-list-container">
        <div className="product-list-subtitle">
          Resultados de b&uacute;squeda: &ldquo;{searchQuery}&rdquo;
          <span className="product-list-count">
            {items.length} art&iacute;culo{items.length !== 1 ? "s" : ""}
          </span>
        </div>
        {renderItems(items)}
      </div>
    );
  }

  // ── Normal mode ──
  if (products.length === 0) {
    return (
      <div className="product-list-empty">
        {selectedFamily
          ? `No hay productos en la familia "${selectedFamily}"`
          : "No hay productos disponibles"}
      </div>
    );
  }

  return (
    <div className="product-list-container">
      <div className="product-list-subtitle">
        {selectedFamily
          ? `Familia: ${selectedFamily}`
          : "Todos los productos"}
        <span className="product-list-count">
          {products.length} artículo{products.length !== 1 ? "s" : ""}
        </span>
      </div>
      {renderItems(products)}
    </div>
  );
}
