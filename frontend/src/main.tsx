import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { App } from "./App";
import { AuthProvider } from "./auth/AuthProvider";
import { LiveProvider } from "./websocket/connection";
import "./index.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <AuthProvider>
      <BrowserRouter>
        <LiveProvider>
          <App />
        </LiveProvider>
      </BrowserRouter>
    </AuthProvider>
  </StrictMode>,
);
