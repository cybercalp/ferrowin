import {
  createContext,
  useContext,
  useReducer,
  type ReactNode,
} from "react";
import type {
  POSProduct,
  CartItem,
  CartTotals,
  PaymentType,
  TerminalHealth,
  OfflineSale,
  CustomerInfo,
  DocumentType,
  UnidadMedida,
} from "./types";
import { computeCartTotals } from "./types";

/* ---------------------------------------------------------------------------
   State
   ------------------------------------------------------------------------- */
interface POSState {
  searchQuery: string;
  searchResults: POSProduct[];
  isSearching: boolean;
  cart: CartItem[];
  selectedPayment: PaymentType | null;
  cartTotals: CartTotals;
  paymentModalOpen: boolean;
  receiptModalOpen: boolean;
  closurePanelOpen: boolean;
  settingsPanelOpen: boolean;
  lastReceiptData: string | null;
  todaySales: OfflineSale[];
  terminalHealth: TerminalHealth | null;

  // Ferreteria workflow fields
  customer: CustomerInfo | null;
  documentType: DocumentType;
  selectedFamily: string | null; // null = "TODO" (show all)
  families: string[];
  barcodeBuffer: string;
  products: POSProduct[];
  isLoadingProducts: boolean;
  viewMode: "list" | "grid";
  gridColumns: number;

  // Numeric keypad state
  activeQtyProductId: string | null;
  keypadBuffer: string;
}

/* ---------------------------------------------------------------------------
   Actions
   ------------------------------------------------------------------------- */
type POSAction =
  | { type: "SET_SEARCH_QUERY"; payload: string }
  | { type: "SET_SEARCH_RESULTS"; payload: POSProduct[] }
  | { type: "SET_IS_SEARCHING"; payload: boolean }
  | { type: "ADD_TO_CART"; payload: POSProduct }
  | { type: "REMOVE_FROM_CART"; payload: string }
  | {
      type: "UPDATE_QUANTITY";
      payload: { productId: string; quantity: number };
    }
  | {
      type: "UPDATE_DISCOUNT";
      payload: { productId: string; discount_percent: number };
    }
  | { type: "CLEAR_CART" }
  | { type: "CLEAR_CART_AND_RESET" }
  | { type: "SET_PAYMENT_METHOD"; payload: PaymentType | null }
  | { type: "OPEN_PAYMENT_MODAL" }
  | { type: "CLOSE_PAYMENT_MODAL" }
  | { type: "OPEN_RECEIPT_MODAL" }
  | { type: "CLOSE_RECEIPT_MODAL" }
  | { type: "OPEN_CLOSURE_PANEL" }
  | { type: "CLOSE_CLOSURE_PANEL" }
  | { type: "OPEN_SETTINGS_PANEL" }
  | { type: "CLOSE_SETTINGS_PANEL" }
  | { type: "SET_RECEIPT_DATA"; payload: string }
  | { type: "SET_TODAY_SALES"; payload: OfflineSale[] }
  | { type: "SET_TERMINAL_HEALTH"; payload: TerminalHealth }
  // Ferreteria actions
  | { type: "SET_CUSTOMER"; payload: CustomerInfo | null }
  | { type: "SET_DOCUMENT_TYPE"; payload: DocumentType }
  | { type: "SET_SELECTED_FAMILY"; payload: string | null }
  | { type: "SET_FAMILIES"; payload: string[] }
  | { type: "SET_PRODUCTS"; payload: POSProduct[] }
  | { type: "SET_LOADING_PRODUCTS"; payload: boolean }
  | { type: "SET_VIEW_MODE"; payload: "list" | "grid" }
  | { type: "SET_GRID_COLUMNS"; payload: number }
  // Numeric keypad actions
  | { type: "SET_ACTIVE_QTY_PRODUCT"; payload: string | null }
  | { type: "SET_KEYPAD_BUFFER"; payload: string };

/* ---------------------------------------------------------------------------
   Context value
   ------------------------------------------------------------------------- */
interface POSContextValue {
  state: POSState;
  dispatch: React.Dispatch<POSAction>;
}

const POSContext = createContext<POSContextValue | undefined>(undefined);

/* ---------------------------------------------------------------------------
   Reducer
   ------------------------------------------------------------------------- */
function computeUnitPrice(
  product: POSProduct,
  customer: CustomerInfo | null,
): number {
  if (customer && customer.descuento > 0) {
    return Math.round(
      product.precio_venta * (1 - customer.descuento / 100) * 100,
    ) / 100;
  }
  return product.precio_venta;
}

function cartReducer(state: POSState, action: POSAction): POSState {
  switch (action.type) {
    case "SET_SEARCH_QUERY":
      return { ...state, searchQuery: action.payload };

    case "SET_SEARCH_RESULTS":
      return { ...state, searchResults: action.payload, isSearching: false };

    case "SET_IS_SEARCHING":
      return { ...state, isSearching: action.payload };

    case "ADD_TO_CART": {
      const existing = state.cart.find(
        (item) => item.product.id === action.payload.id,
      );
      const unidad_medida: UnidadMedida = action.payload.unidad_medida || "ud";
      const unit_price = computeUnitPrice(action.payload, state.customer);

      let newCart: CartItem[];
      if (existing) {
        newCart = state.cart.map((item) =>
          item.product.id === action.payload.id
            ? { ...item, quantity: item.quantity + 1 }
            : item,
        );
      } else {
        newCart = [
          ...state.cart,
          {
            product: action.payload,
            quantity: 1,
            discount_percent: 0,
            unidad_medida,
            unit_price,
          },
        ];
      }
      return {
        ...state,
        cart: newCart,
        cartTotals: computeCartTotals(newCart),
      };
    }

    case "REMOVE_FROM_CART": {
      const newCart = state.cart.filter(
        (item) => item.product.id !== action.payload,
      );
      return {
        ...state,
        cart: newCart,
        cartTotals: computeCartTotals(newCart),
      };
    }

    case "UPDATE_QUANTITY": {
      const newCart = state.cart
        .map((item) =>
          item.product.id === action.payload.productId
            ? { ...item, quantity: Math.max(0, action.payload.quantity) }
            : item,
        )
        .filter((item) => item.quantity > 0);
      return {
        ...state,
        cart: newCart,
        cartTotals: computeCartTotals(newCart),
      };
    }

    case "UPDATE_DISCOUNT": {
      const newCart = state.cart.map((item) =>
        item.product.id === action.payload.productId
          ? {
              ...item,
              discount_percent: Math.max(
                0,
                Math.min(100, action.payload.discount_percent),
              ),
            }
          : item,
      );
      return {
        ...state,
        cart: newCart,
        cartTotals: computeCartTotals(newCart),
      };
    }

    case "CLEAR_CART":
      return {
        ...state,
        cart: [],
        cartTotals: computeCartTotals([]),
        documentType: "ticket",
      };

    case "CLEAR_CART_AND_RESET":
      return {
        ...state,
        cart: [],
        cartTotals: computeCartTotals([]),
        documentType: "ticket",
        // Keep customer
      };

    case "SET_PAYMENT_METHOD":
      return { ...state, selectedPayment: action.payload };

    case "OPEN_PAYMENT_MODAL":
      return { ...state, paymentModalOpen: true };
    case "CLOSE_PAYMENT_MODAL":
      return { ...state, paymentModalOpen: false };
    case "OPEN_RECEIPT_MODAL":
      return { ...state, receiptModalOpen: true, paymentModalOpen: false };
    case "CLOSE_RECEIPT_MODAL":
      return { ...state, receiptModalOpen: false, lastReceiptData: null };
    case "OPEN_CLOSURE_PANEL":
      return { ...state, closurePanelOpen: true };
    case "CLOSE_CLOSURE_PANEL":
      return { ...state, closurePanelOpen: false };
    case "OPEN_SETTINGS_PANEL":
      return { ...state, settingsPanelOpen: true };
    case "CLOSE_SETTINGS_PANEL":
      return { ...state, settingsPanelOpen: false };
    case "SET_RECEIPT_DATA":
      return { ...state, lastReceiptData: action.payload };
    case "SET_TODAY_SALES":
      return { ...state, todaySales: action.payload };
    case "SET_TERMINAL_HEALTH":
      return { ...state, terminalHealth: action.payload };

    // Ferreteria actions
    case "SET_CUSTOMER":
      return { ...state, customer: action.payload };
    case "SET_DOCUMENT_TYPE":
      return { ...state, documentType: action.payload };
    case "SET_SELECTED_FAMILY":
      return { ...state, selectedFamily: action.payload };
    case "SET_FAMILIES":
      return { ...state, families: action.payload };
    case "SET_PRODUCTS":
      return { ...state, products: action.payload, isLoadingProducts: false };
    case "SET_LOADING_PRODUCTS":
      return { ...state, isLoadingProducts: action.payload };

    case "SET_VIEW_MODE":
      return { ...state, viewMode: action.payload };

    case "SET_GRID_COLUMNS":
      return { ...state, gridColumns: action.payload };

    // Numeric keypad
    case "SET_ACTIVE_QTY_PRODUCT":
      return {
        ...state,
        activeQtyProductId: action.payload,
        keypadBuffer: action.payload === null ? "" : state.keypadBuffer,
      };
    case "SET_KEYPAD_BUFFER":
      return { ...state, keypadBuffer: action.payload };

    default:
      return state;
  }
}

/* ---------------------------------------------------------------------------
   Initial state
   ------------------------------------------------------------------------- */
function createInitialState(): POSState {
  return {
    searchQuery: "",
    searchResults: [],
    isSearching: false,
    cart: [],
    selectedPayment: null,
    cartTotals: computeCartTotals([]),
    paymentModalOpen: false,
    receiptModalOpen: false,
    closurePanelOpen: false,
    settingsPanelOpen: false,
    lastReceiptData: null,
    todaySales: [],
    terminalHealth: null,
    // Ferreteria
    customer: null,
    documentType: "ticket",
    selectedFamily: null,
    families: [],
    barcodeBuffer: "",
    products: [],
    isLoadingProducts: false,
    viewMode: "list",
    gridColumns: Number(localStorage.getItem("ferrowin:gridColumns")) || 4,
    activeQtyProductId: null,
    keypadBuffer: "",
  };
}

/* ---------------------------------------------------------------------------
   Provider
   ------------------------------------------------------------------------- */
function POSProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(
    cartReducer,
    undefined,
    createInitialState,
  );
  return (
    <POSContext.Provider value={{ state, dispatch }}>
      {children}
    </POSContext.Provider>
  );
}

/* ---------------------------------------------------------------------------
   Hook
   ------------------------------------------------------------------------- */
function usePOS(): POSContextValue {
  const ctx = useContext(POSContext);
  if (!ctx) {
    throw new Error("usePOS must be used within a <POSProvider>");
  }
  return ctx;
}

export { POSProvider, usePOS };
export type { POSState, POSAction, POSContextValue };
