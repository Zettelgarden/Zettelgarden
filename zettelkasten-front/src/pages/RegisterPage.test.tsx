import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { BrowserRouter } from "react-router-dom";
import RegisterPage from "./RegisterPage";
import { useSettings } from "../contexts/SettingsContext";

// Control the instance's public settings (signups_enabled) directly.
vi.mock("../contexts/SettingsContext", () => ({
  useSettings: vi.fn(),
}));

const mockUseSettings = vi.mocked(useSettings);

function renderRegister() {
  return render(
    <BrowserRouter>
      <RegisterPage />
    </BrowserRouter>,
  );
}

describe("RegisterPage signups gate (6er.10)", () => {
  beforeEach(() => {
    mockUseSettings.mockReset();
  });

  it("shows the registration form when signups are enabled", () => {
    mockUseSettings.mockReturnValue({
      settings: {
        siteName: "Zettelgarden",
        signupsEnabled: true,
        oidcAutoProvision: true,
        mailEnabled: true,
        emailAutoValidate: true,
        supportEmail: "",
      },
    });
    renderRegister();
    expect(screen.getByText(/Create your Zettelgarden account/i)).toBeTruthy();
    expect(screen.getByText("Register")).toBeTruthy();
    expect(screen.queryByText(/Registration is closed/i)).toBeNull();
  });

  it("shows the closed message instead of the form when signups are disabled", () => {
    mockUseSettings.mockReturnValue({
      settings: {
        siteName: "Zettelgarden",
        signupsEnabled: false,
        oidcAutoProvision: true,
        mailEnabled: true,
        emailAutoValidate: true,
        supportEmail: "",
      },
    });
    renderRegister();
    expect(screen.getByText(/Registration is closed/i)).toBeTruthy();
    expect(screen.queryByText("Register")).toBeNull();
  });
});
