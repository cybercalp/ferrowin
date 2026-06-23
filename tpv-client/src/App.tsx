import { SyncWarningBanner } from "./components/SyncWarningBanner";
import { ThemeProvider } from "./theme/ThemeProvider";
import { POSProvider } from "./pos/PosContext";
import { POSMainLayout } from "./pos/POSMainLayout";
import "./App.css";

function App() {
  return (
    <ThemeProvider>
      <SyncWarningBanner />
      <POSProvider>
        <POSMainLayout />
      </POSProvider>
    </ThemeProvider>
  );
}

export default App;
