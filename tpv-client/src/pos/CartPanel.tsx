import { usePOS } from "./PosContext";
import { CartItemRow } from "./CartItemRow";

export function CartPanel() {
  const { state, dispatch } = usePOS();
  const { cart, cartTotals, customer } = state;

  const hasItems = cart.length > 0;

  return (
    <div className="cart-panel">
      {/* Header */}
      <div className="cart-header">
        <h2 className="cart-title">Carrito</h2>
        <div className="cart-header-badges">
          <button
            className={`cart-doctype-badge doctype-${state.documentType === "albarán" ? "albaran" : state.documentType}`}
            onClick={() => {
              const next: Record<string, string> = {
                ticket: "factura",
                factura: "albarán",
                "albarán": "ticket",
              };
              dispatch({
                type: "SET_DOCUMENT_TYPE",
                payload: next[state.documentType] as "ticket" | "factura" | "albarán",
              });
            }}
            title="Cambiar tipo de documento"
          >
            {state.documentType === "ticket" && "Ticket"}
            {state.documentType === "factura" && "Factura"}
            {state.documentType === "albarán" && "Albarán"}
          </button>
          {hasItems && (
            <span className="cart-count">
              {cart.reduce((s, i) => s + i.quantity, 0)} ud.
            </span>
          )}
        </div>
      </div>

      {/* Customer discount banner */}
      {customer && customer.descuento > 0 && (
        <div className="cart-customer-banner">
          <span className="cart-customer-name">{customer.nombre}</span>
          <span className="cart-customer-discount-badge">
            -{customer.descuento}%
          </span>
        </div>
      )}

      {/* Items table */}
      <div className="cart-items">
        {!hasItems ? (
          <div className="cart-empty">
            Selecciona productos para agregar al carrito
          </div>
        ) : (
          <table className="cart-table">
            <thead>
              <tr>
                <th className="cart-th-codigo">Código</th>
                <th className="cart-th-descripcion">Descripción</th>
                <th className="cart-th-ud">Ud</th>
                <th className="cart-th-precio">Precio</th>
                <th className="cart-th-dto">%Dto</th>
                <th className="cart-th-cantidad">Cant</th>
                <th className="cart-th-total">Total</th>
                <th className="cart-th-remove"></th>
              </tr>
            </thead>
            <tbody>
              {cart.map((item) => (
                <CartItemRow key={item.product.id} item={item} />
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Totals section */}
      {hasItems && (
        <>
          <div className="iva-breakdown">
            <h3 className="iva-breakdown-title">Desglose de IVA</h3>
            {cartTotals.iva_breakdown.length === 0 ? (
              <div className="iva-row">
                <span className="iva-row-label">Sin IVA</span>
              </div>
            ) : (
              cartTotals.iva_breakdown.map((entry) => (
                <div key={entry.name} className="iva-row">
                  <span className="iva-row-label">
                    {entry.name} ({entry.percent}%)
                  </span>
                  <span className="iva-row-base">
                    {entry.base.toFixed(2)} €
                  </span>
                  <span className="iva-row-tax">
                    IVA: {entry.tax.toFixed(2)} €
                  </span>
                </div>
              ))
            )}
          </div>

          <div className="cart-totals">
            <div className="total-row">
              <span>Subtotal</span>
              <span>{cartTotals.subtotal.toFixed(2)} €</span>
            </div>
            {cartTotals.discount_total > 0 && (
              <div className="total-row total-discount">
                <span>Descuento</span>
                <span>-{cartTotals.discount_total.toFixed(2)} €</span>
              </div>
            )}
            <div className="total-row">
              <span>IVA</span>
              <span>{cartTotals.tax_total.toFixed(2)} €</span>
            </div>
            <div className="total-row total-grand">
              <span>Total</span>
              <span>{cartTotals.total.toFixed(2)} €</span>
            </div>
          </div>
        </>
      )}

      {/* Cobrar button */}
      <div className="cart-cobrar-section">
        {!hasItems && (
          <div className="cart-cobrar-empty-msg">
            Agrega productos al carrito para cobrar
          </div>
        )}
        <button
          className="btn-cobrar"
          disabled={!hasItems || cartTotals.total <= 0}
          onClick={() => dispatch({ type: "OPEN_PAYMENT_MODAL" })}
        >
          <span className="btn-cobrar-label">Cobrar</span>
          <span className="btn-cobrar-total">
            {cartTotals.total.toFixed(2)} €
          </span>
        </button>
        {hasItems && (
          <button
            className="btn-clear-cart"
            onClick={() => dispatch({ type: "CLEAR_CART" })}
            title="Limpiar carrito"
          >
            Limpiar
          </button>
        )}
      </div>
    </div>
  );
}
