import React, { FormEvent, useState, useEffect } from "react";
import { useAuth } from "../contexts/AuthContext";
import { login } from "../api/auth";
import { FaGithub, FaFingerprint } from "react-icons/fa";

import { Link, useNavigate, useLocation } from "react-router-dom";

// Friendly messages for the error codes emitted by the OAuth callback
// handlers (see go-backend/handlers/oidc.go and oauth.go `fail(...)`).
// Codes are shared across the OIDC and GitHub flows; no_email is GitHub-only.
const OAUTH_ERROR_MESSAGES: Record<string, string> = {
  missing_state: "Sign-in session expired. Please try again.",
  bad_state: "Sign-in session was invalid. Please try again.",
  state_mismatch: "Sign-in security check failed (state mismatch). Please try again.",
  missing_code: "The identity provider did not return an authorization code.",
  oidc_unavailable: "Single sign-on is not available right now. Please try again later.",
  exchange_failed: "Could not complete sign-in with the identity provider.",
  no_email: "GitHub did not return an email address.",
  no_id_token: "The identity provider did not return an identity token.",
  bad_id_token: "The identity token was invalid. Please try again.",
  nonce_mismatch: "Sign-in replay check failed. Please try again.",
  bad_claims: "Could not read your identity information from the provider.",
  user_resolve_failed: "Could not sign you in. If the problem persists, contact support.",
  jwt_failed: "Could not complete sign-in. Please try again.",
};

function LoginForm() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const { loginUser, loginUserFromToken } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const message = location.state?.message;

  const oidcEnabled = import.meta.env.VITE_OIDC_ENABLED === "true";
  const oidcLabel = import.meta.env.VITE_OIDC_LABEL || "Continue with SSO";

  // GitHub login is shown by default; set VITE_GITHUB_AUTH_ENABLED=false to
  // hide the button when a generic OIDC provider replaces it.
  const githubEnabled = import.meta.env.VITE_GITHUB_AUTH_ENABLED !== "false";

  const handleLogin = async (e: FormEvent) => {
    e.preventDefault();
    try {
      const response = await login(email, password);
      loginUser(response);
      navigate("/app/");
    } catch (message) {
      setError("Login Failed: " + message);
    }
  };

  const handleGitHubLogin = () => {
    const githubOAuthURL = `${import.meta.env.VITE_URL}/auth/github`;
    window.location.href = githubOAuthURL;
  };

  const handleOIDCLogin = () => {
    const oidcStartURL = `${import.meta.env.VITE_URL}/auth/oidc/start`;
    window.location.href = oidcStartURL;
  };

  useEffect(() => {
    const handleOAuthCallback = async () => {
      const params = new URLSearchParams(location.search);
      const token = params.get("token");
      const errorCode = params.get("error");

      if (errorCode) {
        setError(
          OAUTH_ERROR_MESSAGES[errorCode] ||
            "Sign-in failed. Please try again.",
        );
        return;
      }

      if (token) {
        try {
          await loginUserFromToken(token);
          navigate("/app/");
        } catch (error) {
          console.error("OAuth login failed:", error);
          setError("OAuth login failed. Please try again.");
        }
      }
    };

    handleOAuthCallback();
  }, [location, loginUserFromToken, navigate]);

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-50 px-4">
      <div className="bg-white p-8 rounded-lg shadow-lg w-full max-w-sm">
        <h2 className="text-2xl font-bold text-center mb-4">Login</h2>
        <div className="text-center text-red-500 mb-4">
          {error && <span>{error}</span>}
        </div>
        <div className="text-center text-green-500 mb-4">
          {message && <span>{message}</span>}
        </div>
        {githubEnabled && (
          <button
            onClick={handleGitHubLogin}
            className="w-full bg-gray-200 text-gray-700 py-2.5 rounded-lg hover:bg-gray-300 transition duration-200 flex items-center justify-center"
            type="button"
          >
            <FaGithub className="mr-2" />
            Continue with GitHub
          </button>
        )}
        {oidcEnabled && (
          <button
            onClick={handleOIDCLogin}
            className="w-full bg-blue-600 text-white py-2.5 rounded-lg hover:bg-blue-700 transition duration-200 flex items-center justify-center"
            type="button"
          >
            <FaFingerprint className="mr-2" />
            {oidcLabel}
          </button>
        )}

        <div className="my-4 flex items-center">
          <div className="flex-grow border-t border-gray-300"></div>
          <span className="flex-shrink mx-4 text-gray-500">
            or login with email
          </span>
          <div className="flex-grow border-t border-gray-300"></div>
        </div>

        <form onSubmit={handleLogin} className="space-y-4">
          <div>
            <label
              htmlFor="email"
              className="block text-sm font-medium text-gray-700"
            >
              Email <span className="text-red-500">*</span>
            </label>
            <input
              id="email"
              name="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="mt-1 w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
              style={{ fontSize: '16px' }}
              required
            />
          </div>
          <div>
            <label
              htmlFor="password"
              className="block text-sm font-medium text-gray-700"
            >
              Password <span className="text-red-500">*</span>
            </label>
            <input
              id="password"
              name="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="mt-1 w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
              style={{ fontSize: '16px' }}
              required
            />
          </div>
          <button
            type="submit"
            className="w-full bg-blue-500 text-white py-2.5 rounded-lg hover:bg-blue-600 transition duration-200"
          >
            Login
          </button>
        </form>

        <div className="text-center mt-6 text-sm">
          <p>
            Don't have an account?{" "}
            <Link to="/register" className="text-blue-500 hover:underline">
              Register here
            </Link>
          </p>
          <p className="mt-2">
            <Link to="/reset" className="text-blue-500 hover:underline">
              Forgot your password?
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}

export default LoginForm;
