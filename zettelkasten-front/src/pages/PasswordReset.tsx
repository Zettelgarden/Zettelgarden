import React, { useState, useEffect, FormEvent } from "react";
import { useNavigate, useLocation, Link } from "react-router-dom";
import { requestPasswordReset, resetPassword } from "../api/auth";

function PasswordReset() {
  const [email, setEmail] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [token, setToken] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    const query = new URLSearchParams(location.search);
    const token = query.get("token");
    if (token) {
      setToken(token);
    }
  }, [location]);

  const handleRequestReset = async (e: FormEvent) => {
    e.preventDefault();
    setMessage("");
    setError("");
    try {
      const response = await requestPasswordReset(email);
      if (response.error) {
        setError(response.message);
      } else {
        setMessage(
          "If your email is in our system, you will receive a password reset link.",
        );
      }
    } catch (error) {
      setError("Failed to request password reset.");
    }
  };

  const handleResetPassword = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    if (newPassword !== confirmPassword) {
      setError("Passwords do not match.");
      return;
    }
    try {
      const response = await resetPassword(token, newPassword);
      if (response.error) {
        setError(response.message);
      } else {
        setMessage("Your password has been successfully updated.");
        setTimeout(() => navigate("/login"), 2000);
      }
    } catch (error) {
      setError("Failed to reset password.");
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-50 px-4">
      <div className="bg-white p-8 rounded-lg shadow-lg w-full max-w-sm">
        {token ? (
          // Reset Password Form
          <div>
            <h2 className="text-2xl font-bold text-center mb-6">Reset Password</h2>
            {error && (
              <div className="text-center text-red-500 mb-4">
                {error}
              </div>
            )}
            {message && (
              <div className="text-center text-green-500 mb-4">
                {message}
              </div>
            )}
            <form onSubmit={handleResetPassword} className="space-y-4">
              <div>
                <label
                  htmlFor="new-password"
                  className="block text-sm font-medium text-gray-700"
                >
                  New Password <span className="text-red-500">*</span>
                </label>
                <input
                  id="new-password"
                  name="new-password"
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  placeholder="Enter new password"
                  className="mt-1 w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  style={{ fontSize: '16px' }}
                  required
                />
              </div>
              <div>
                <label
                  htmlFor="confirm-password"
                  className="block text-sm font-medium text-gray-700"
                >
                  Confirm New Password <span className="text-red-500">*</span>
                </label>
                <input
                  id="confirm-password"
                  name="confirm-password"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="Confirm new password"
                  className="mt-1 w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  style={{ fontSize: '16px' }}
                  required
                />
              </div>
              <button
                type="submit"
                className="w-full bg-blue-500 text-white py-2.5 rounded-lg hover:bg-blue-600 transition duration-200"
              >
                Reset Password
              </button>
            </form>
            <div className="text-center mt-6 text-sm">
              <Link to="/login" className="text-blue-500 hover:underline">
                Back to Login
              </Link>
            </div>
          </div>
        ) : (
          // Request Password Reset Form
          <div>
            <h2 className="text-2xl font-bold text-center mb-6">Request Password Reset</h2>
            {error && (
              <div className="text-center text-red-500 mb-4">
                {error}
              </div>
            )}
            {message && (
              <div className="text-center text-green-500 mb-4">
                {message}
              </div>
            )}
            <form onSubmit={handleRequestReset} className="space-y-4">
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
                  placeholder="Enter your email"
                  className="mt-1 w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  style={{ fontSize: '16px' }}
                  required
                />
              </div>
              <button
                type="submit"
                className="w-full bg-blue-500 text-white py-2.5 rounded-lg hover:bg-blue-600 transition duration-200"
              >
                Request Reset
              </button>
            </form>
            <div className="text-center mt-6 text-sm">
              <Link to="/login" className="text-blue-500 hover:underline">
                Back to Login
              </Link>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default PasswordReset;
