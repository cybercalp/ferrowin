import { useEffect } from "react";
import { usePOS } from "./PosContext";

interface NumericKeypadProps {
  visible: boolean;
}

export function NumericKeypad({ visible }: NumericKeypadProps) {
  const { state, dispatch } = usePOS();
  const { keypadBuffer, activeQtyProductId } = state;

  const handleDigit = (digit: string) => {
    dispatch({
      type: "SET_KEYPAD_BUFFER",
      payload: keypadBuffer + digit,
    });
  };

  const handleBackspace = () => {
    dispatch({
      type: "SET_KEYPAD_BUFFER",
      payload: keypadBuffer.slice(0, -1),
    });
  };

  const handleClear = () => {
    dispatch({ type: "SET_KEYPAD_BUFFER", payload: "" });
  };

  const handleConfirm = () => {
    if (keypadBuffer && activeQtyProductId) {
      const parsed = parseFloat(keypadBuffer);
      if (!isNaN(parsed) && parsed > 0) {
        dispatch({
          type: "UPDATE_QUANTITY",
          payload: { productId: activeQtyProductId, quantity: parsed },
        });
      }
    }
    dispatch({ type: "SET_ACTIVE_QTY_PRODUCT", payload: null });
    dispatch({ type: "SET_KEYPAD_BUFFER", payload: "" });
  };

  /** Dismiss keypad — also fired on backdrop click and Escape key */
  const handleDismiss = () => {
    dispatch({ type: "SET_ACTIVE_QTY_PRODUCT", payload: null });
    dispatch({ type: "SET_KEYPAD_BUFFER", payload: "" });
  };

  /** Prevent backdrop-dismiss when tapping inside the keypad card */
  const handleCardClick = (e: React.MouseEvent) => {
    e.stopPropagation();
  };

  // Keyboard support: Enter→OK, Backspace→del, Escape/C→clear, digits→type
  useEffect(() => {
    if (!visible) return;

    const onKeyDown = (e: KeyboardEvent) => {
      // Don't intercept if user is typing in the barcode input
      if (
        document.activeElement?.tagName === "INPUT" ||
        document.activeElement?.tagName === "TEXTAREA"
      ) {
        // Still handle Enter/Backspace/Escape even if input is focused
        if (e.key !== "Enter" && e.key !== "Backspace" && e.key !== "Escape") {
          return;
        }
      }

      switch (e.key) {
        case "Enter":
          e.preventDefault();
          handleConfirm();
          break;
        case "Backspace":
          e.preventDefault();
          handleBackspace();
          break;
        case "Escape":
        case "c":
        case "C":
          e.preventDefault();
          handleDismiss();
          break;
        default:
          // Single digits
          if (/^[0-9]$/.test(e.key)) {
            e.preventDefault();
            handleDigit(e.key);
          }
      }
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [visible, keypadBuffer, activeQtyProductId]);

  if (!visible) {
    return (
      <div className="numeric-keypad nk-hidden" aria-hidden={true} />
    );
  }

  return (
    <div
      className="numeric-keypad nk-visible"
      onClick={handleDismiss}
    >
      <div className="nk-card-wrapper" onClick={handleCardClick}>
        <div className="nk-display">
          {keypadBuffer || "0"}
        </div>
        <div className="nk-grid">
          <button className="nk-btn" onClick={() => handleDigit("1")} aria-label="1">
            1
          </button>
          <button className="nk-btn" onClick={() => handleDigit("2")} aria-label="2">
            2
          </button>
          <button className="nk-btn" onClick={() => handleDigit("3")} aria-label="3">
            3
          </button>
        <button className="nk-btn" onClick={() => handleDigit("4")} aria-label="4">
          4
        </button>
        <button className="nk-btn" onClick={() => handleDigit("5")} aria-label="5">
          5
        </button>
        <button className="nk-btn" onClick={() => handleDigit("6")} aria-label="6">
          6
        </button>
        <button className="nk-btn" onClick={() => handleDigit("7")} aria-label="7">
          7
        </button>
        <button className="nk-btn" onClick={() => handleDigit("8")} aria-label="8">
          8
        </button>
        <button className="nk-btn" onClick={() => handleDigit("9")} aria-label="9">
          9
        </button>
        <button
          className="nk-btn nk-clear"
          onClick={handleClear}
          aria-label="Limpiar"
        >
          C
        </button>
        <button className="nk-btn" onClick={() => handleDigit("0")} aria-label="0">
          0
        </button>
        <button
          className="nk-btn nk-backspace"
          onClick={handleBackspace}
          aria-label="Retroceso"
        >
          &larr;
        </button>
        <button
          className="nk-btn nk-ok"
          onClick={handleConfirm}
          aria-label="Confirmar cantidad"
        >
          &#10003; OK
        </button>
      </div>
    </div>
    </div>
  );
}
