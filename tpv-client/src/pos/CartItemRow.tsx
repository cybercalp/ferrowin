import { useCallback } from "react";
import { usePOS } from "./PosContext";
import type { CartItem } from "./types";

interface CartItemRowProps {
  item: CartItem;
}

export function CartItemRow({ item }: CartItemRowProps) {
  const { state, dispatch } = usePOS();
  const { activeQtyProductId, keypadBuffer } = state;

  const isActive = activeQtyProductId === item.product.id;
  const displayQty = isActive ? (keypadBuffer || "0") : String(item.quantity);

  const lineTotal =
    item.unit_price * item.quantity * (1 - item.discount_percent / 100);

  const handleQtyClick = useCallback(() => {
    if (isActive) {
      dispatch({ type: "SET_ACTIVE_QTY_PRODUCT", payload: null });
      dispatch({ type: "SET_KEYPAD_BUFFER", payload: "" });
    } else {
      dispatch({ type: "SET_ACTIVE_QTY_PRODUCT", payload: item.product.id });
      dispatch({ type: "SET_KEYPAD_BUFFER", payload: String(item.quantity) });
    }
  }, [dispatch, item.product.id, item.quantity, isActive]);

  return (
    <tr className={`cart-item-row ${isActive ? "cart-row-active" : ""}`}>
      <td className="cart-col-codigo">{item.product.codigo}</td>
      <td className="cart-col-descripcion">
        <span className="cart-item-desc">{item.product.nombre}</span>
      </td>
      <td className="cart-col-ud">
        <span className="ud-badge">{item.unidad_medida}</span>
      </td>
      <td className="cart-col-precio">{item.unit_price.toFixed(4)} €</td>
      <td className="cart-col-dto">
        <div className="cart-dto-wrap">
          <input
            type="number"
            min="0"
            max="100"
            value={item.discount_percent}
            onChange={(e) =>
              dispatch({
                type: "UPDATE_DISCOUNT",
                payload: {
                  productId: item.product.id,
                  discount_percent: Number(e.target.value) || 0,
                },
              })
            }
            className="dto-input"
            aria-label="Porcentaje de descuento"
          />
          <span>%</span>
        </div>
      </td>
      <td className="cart-col-cantidad">
        <span
          className={`qty-value qty-clickable ${isActive ? "qty-active" : ""}`}
          onClick={handleQtyClick}
          title="Click para editar cantidad"
        >
          {displayQty}
        </span>
      </td>
      <td className="cart-col-total">{lineTotal.toFixed(2)} €</td>
      <td className="cart-col-remove">
        <button
          className="cart-remove-btn"
          onClick={() =>
            dispatch({ type: "REMOVE_FROM_CART", payload: item.product.id })
          }
          aria-label="Eliminar articulo"
        >
          &times;
        </button>
      </td>
    </tr>
  );
}
