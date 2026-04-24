import React from "react";
import ReactDOM from "react-dom/client";
import "./index.css";
import App from "./App";
import { AuthProvider } from "./contexts/AuthContext";
import { HashRouter } from "react-router-dom";
import { Buffer } from 'buffer';
globalThis.Buffer = Buffer;

const rootElement = document.getElementById("root");

if (rootElement) {
  const root = ReactDOM.createRoot(rootElement);
  root.render(
    <React.StrictMode>
      <HashRouter>
        <AuthProvider>
          <App />
        </AuthProvider>
      </HashRouter>
    </React.StrictMode>,
  );
} else {
  console.error("Root element not found");
}
