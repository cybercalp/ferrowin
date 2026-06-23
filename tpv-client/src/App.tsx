import { BrowserRouter, Routes, Route, Navigate } from "react-router";
import { AuthProvider, useAuth } from "./context/AuthContext";
import { SyncWarningBanner } from "./components/SyncWarningBanner";
import { ThemeProvider } from "./theme/ThemeProvider";
import { POSProvider } from "./pos/PosContext";
import { POSMainLayout } from "./pos/POSMainLayout";
import { LoginScreen } from "./screens/LoginScreen";
import { EntityManager } from "./components/EntityManager";
import { EntityDetailPanel } from "./components/EntityDetailPanel";
import { ClientDossierView } from "./components/ClientDossierView";
import { ClientCollection } from "./components/ClientCollection";
import "./App.css";

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div style={loadingContainerStyle}>
        <div style={loadingSpinnerStyle} />
        <p style={loadingTextStyle}>Cargando...</p>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

function AppContent() {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div style={loadingContainerStyle}>
        <div style={loadingSpinnerStyle} />
        <p style={loadingTextStyle}>Cargando...</p>
      </div>
    );
  }

  return (
    <Routes>
      <Route
        path="/login"
        element={
          isAuthenticated ? <Navigate to="/" replace /> : <LoginScreen />
        }
      />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <SyncWarningBanner />
            <POSProvider>
              <POSMainLayout />
            </POSProvider>
          </ProtectedRoute>
        }
      />
      <Route
        path="/entities"
        element={
          <ProtectedRoute>
            <EntityManager />
          </ProtectedRoute>
        }
      />
      <Route
        path="/entities/:id"
        element={
          <ProtectedRoute>
            <EntityDetailPanel />
          </ProtectedRoute>
        }
      />
      <Route
        path="/clients/:id/dossier"
        element={
          <ProtectedRoute>
            <ClientDossierView />
          </ProtectedRoute>
        }
      />
      <Route
        path="/clients/:id/collection"
        element={
          <ProtectedRoute>
            <ClientCollection />
          </ProtectedRoute>
        }
      />
    </Routes>
  );
}

function App() {
  return (
    <ThemeProvider>
      <BrowserRouter>
        <AuthProvider>
          <AppContent />
        </AuthProvider>
      </BrowserRouter>
    </ThemeProvider>
  );
}

export default App;

// ---------------------------------------------------------------------------
// Loading styles
// ---------------------------------------------------------------------------

const loadingContainerStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  alignItems: "center",
  justifyContent: "center",
  minHeight: "100vh",
  gap: "16px",
  backgroundColor: "var(--bg-primary, #f8fafc)",
};

const loadingSpinnerStyle: React.CSSProperties = {
  width: "36px",
  height: "36px",
  border: "3px solid var(--border-color, #e2e8f0)",
  borderTopColor: "var(--primary, #2563eb)",
  borderRadius: "50%",
  animation: "app-spin 0.8s linear infinite",
};

const loadingTextStyle: React.CSSProperties = {
  fontSize: "14px",
  color: "var(--text-secondary, #64748b)",
  margin: 0,
  fontFamily: "var(--font-family, system-ui)",
};
