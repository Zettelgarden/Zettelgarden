import React, { useState, useEffect, FormEvent } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { requestPasswordReset, resetPassword } from "../api/auth"; // Make sure these are implemented in api.js

function PasswordReset() {
  const [email, setEmail] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [token, setToken] = useState("");
  const [message, setMessage] = useState("");
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
    try {
      const response = await requestPasswordReset(email);
      if (response.error) {
        setMessage(response.message);
      } else {
        setMessage(
          "If your email is in our system, you will receive a password reset link.",
        );
      }
    } catch (error) {
      setMessage("Failed to request password reset.");
    }
  };

  const handleResetPassword = async (e: FormEvent) => {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      setMessage("Passwords do not match.");
      return;
    }
    try {
      const response = await resetPassword(token, newPassword);
      if (response.error) {
        setMessage(response.message);
      } else {
        setMessage("Your password has been successfully updated.");
        navigate("/login"); // Redirect to login page or wherever appropriate
      }
    } catch (error) {
      setMessage("Failed to reset password.");
    }
  };

  return (
    <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-[20%] border border-gray-300 p-[30px]">
      {token ? (
        // Reset Password Form
        <div>
          <h2>Reset Password</h2>
          {message && <div>{message}</div>}
          <form onSubmit={handleResetPassword}>
            <input
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder="New Password"
              required
            />
            <input
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder="Confirm New Password"
              required
            />
            <button type="submit">Reset Password</button>
          </form>
        </div>
      ) : (
        // Request Password Reset Form
        <div>
          <h2>Request Password Reset</h2>
          {message && <div>{message}</div>}
          <form onSubmit={handleRequestReset}>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="Enter your email"
              required
            />
            <button type="submit">Request Reset</button>
          </form>
        </div>
      )}
    </div>
  );
}

export default PasswordReset;
