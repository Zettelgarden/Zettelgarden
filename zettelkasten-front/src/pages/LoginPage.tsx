import React, { FormEvent, useState, useEffect } from "react";
import { useAuth } from "../contexts/AuthContext";
import { login } from "../api/auth";
import { FaGithub, FaCode } from "react-icons/fa";

import { Link, useNavigate, useLocation } from "react-router-dom";

function LoginForm() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const { loginUser, loginUserFromToken } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const message = location.state?.message;

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

  useEffect(() => {
    const handleOAuthCallback = async () => {
      const params = new URLSearchParams(location.search);
      const token = params.get("token");

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
        <button
          onClick={handleGitHubLogin}
          className="w-full bg-gray-200 text-gray-700 py-2.5 rounded-lg hover:bg-gray-300 transition duration-200 flex items-center justify-center"
          type="button"
        >
          <FaGithub className="mr-2" />
          Continue with GitHub
        </button>

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
