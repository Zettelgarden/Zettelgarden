import React, { FormEvent, useState } from "react";
import { createUser } from "../api/users";
import { Link, useNavigate } from "react-router-dom";

function RegisterPage() {
  // State to store each input field's value
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");

  const navigate = useNavigate();

  // Handle form submission
  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();

    // Basic validation to check if passwords match
    if (password !== confirmPassword) {
      setError("Passwords do not match");
      return;
    }

    // Submit the form data
    console.log("Form submitted", {
      username,
      email,
      password,
      confirmPassword,
    });
    // Here you would typically send the data to your server via an API

    const userData = { username, email, password, confirmPassword };

    createUser(userData)
      .then((data) => {
        console.log("User created successfully", data);
        navigate("/login");
        navigate("/login", {
          state: {
            message:
              "Account successfully created. Check your email for a validation link.",
          },
        });
        // Handle successful user creation (e.g., redirecting the user or showing a success message)
        // Reset form or redirect user to login page, etc.
      })
      .catch((error) => {
        console.error("Error creating user:", error);
        setError("Failed to create user. Please try again.");
      });
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-100 px-4">
      <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-md">
        <h2 className="text-2xl font-bold text-center mb-6">
          Create your Zettelgarden account
        </h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Username:
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Email:
            </label>
            <input
              type="email"
              value={email}
              pattern="[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}"
              onChange={(e) => setEmail(e.target.value)}
              required
              className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Password:
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Confirm Password:
            </label>
            <input
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
              className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>
          {error && <p className="text-red-500 text-center">{error}</p>}
          <button
            type="submit"
            className="w-full bg-indigo-500 text-white py-2 rounded-md hover:bg-indigo-600 transition duration-200"
          >
            Register
          </button>
          <span className="block text-center mt-4">
            Already have an account?{" "}
            <Link to="/login" className="text-indigo-500 hover:underline">
              Login instead
            </Link>
            .
          </span>
        </form>
      </div>
    </div>
  );
}

export default RegisterPage;
