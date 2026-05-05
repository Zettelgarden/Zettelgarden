import React from "react";
import { useNavigate } from "react-router-dom";

export default function Cancel() {
  const navigate = useNavigate();

  return (
    <div className="flex flex-col items-center mt-10">
      <h1 className="text-2xl font-bold text-red-600">Payment Canceled</h1>
      <p className="mb-4">You can try subscribing again anytime.</p>
      <button
        onClick={() => navigate("/app/subscription")}
        className="bg-indigo-600 text-white px-6 py-2 rounded-lg font-medium hover:bg-indigo-700 transition-colors"
      >
        Back to Plans
      </button>
    </div>
  );
}
