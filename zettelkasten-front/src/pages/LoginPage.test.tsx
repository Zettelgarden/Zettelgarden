import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import LoginForm from "./LoginPage";
import { BrowserRouter } from "react-router-dom";
import { AuthProvider } from "../contexts/AuthContext";

// Vite inlines import.meta.env at build time; in tests we can stub it per
// test by mutating the env object before render.
function renderLogin() {
  return render(
    <BrowserRouter>
      <AuthProvider>
        <LoginForm />
      </AuthProvider>
    </BrowserRouter>,
  );
}

describe("LoginPage OAuth buttons", () => {
  const originalEnv = { ...import.meta.env };

  beforeEach(() => {
    // Reset to defaults before each test: GitHub shown, OIDC hidden.
    Object.assign(import.meta.env, originalEnv);
    delete import.meta.env.VITE_OIDC_ENABLED;
    delete import.meta.env.VITE_GITHUB_AUTH_ENABLED;
  });

  it("shows the GitHub button by default", () => {
    renderLogin();
    expect(screen.getByText("Continue with GitHub")).toBeTruthy();
  });

  it("hides the GitHub button when VITE_GITHUB_AUTH_ENABLED=false", () => {
    import.meta.env.VITE_GITHUB_AUTH_ENABLED = "false";
    renderLogin();
    expect(screen.queryByText("Continue with GitHub")).toBeNull();
  });

  it("shows the OIDC button when VITE_OIDC_ENABLED=true alongside GitHub", () => {
    import.meta.env.VITE_OIDC_ENABLED = "true";
    renderLogin();
    expect(screen.getByText("Continue with GitHub")).toBeTruthy();
    expect(screen.getByText("Continue with SSO")).toBeTruthy();
  });
});
