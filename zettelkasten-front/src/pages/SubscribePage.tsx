import React, { useState, useEffect } from "react";
import { loadStripe } from "@stripe/stripe-js";
import { useNavigate } from "react-router-dom";
import { getStripePublicKey } from "../api/billing";
import { getCurrentUser, getUserSubscription } from "../api/users";
import { User, UserSubscription } from "../models/User";

const base_url = import.meta.env.VITE_URL;

export default function SubscribePage() {
  const [stripePromise, setStripePromise] = useState<Promise<any> | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();
  const [user, setUser] = useState<User | null>(null);
  const [subscription, setSubscription] = useState<UserSubscription | null>(null);

  useEffect(() => {
    async function fetchStripeKey() {
      try {
        const response = await getStripePublicKey();
        setStripePromise(loadStripe(response.key));
      } catch (error) {
        console.error("Failed to fetch Stripe public key:", error);
      }
    }
    async function fetchUserAndSubscription() {
      try {
        const userResponse = await getCurrentUser();
        setUser(userResponse);
        if (userResponse && userResponse.id) {
          const subscriptionResponse = await getUserSubscription(userResponse.id);
          setSubscription(subscriptionResponse);
          if (subscriptionResponse && subscriptionResponse.stripe_subscription_status === 'active') {
            navigate('/app/settings');
          }
        }
      } catch (error) {
        console.error("Failed to fetch user and subscription:", error);
      }
    }
    fetchStripeKey();
    fetchUserAndSubscription();
  }, [navigate]);

  const startSubscription = async (plan: "monthly" | "annual") => {
    setLoading(true);
    setError(null);

    try {
      const res = await fetch(`${base_url}/billing/subscribe`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${localStorage.getItem("token")}`,
        },
        body: JSON.stringify({ plan }),
      });

      if (!res.ok) {
        throw new Error("Failed to start checkout");
      }

      const data = await res.json();
      if (data.checkout_url) {
        window.location.href = data.checkout_url;
      } else {
        throw new Error("No checkout URL returned");
      }
    } catch (err: any) {
      setError(err.message || "Something went wrong");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-5xl mx-auto px-6 py-12">
      <h1 className="text-3xl font-bold text-center mb-6 text-indigo-700">Simple, Transparent Pricing</h1>
      <p className="text-gray-600 text-center mb-12 max-w-2xl mx-auto">
        Upgrade to Zettelgarden Pro to unlock intelligent AI agents that can work with your knowledge base, advanced summarization, fact extraction, and early access to new features.
      </p>

      <div className="flex flex-col md:flex-row gap-8 justify-center items-stretch flex-wrap">
        {/* Free Plan */}
        <div className="bg-white shadow-md rounded-lg p-6 w-full max-w-sm flex flex-col">
          <h3 className="text-xl font-semibold text-indigo-700 mb-2">Free</h3>
          <p className="text-gray-700 mb-4">$0 / forever</p>
          <ul className="text-left mb-6 space-y-2">
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Create and Organize Cards</li>
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Knowledge Linking and Organization</li>
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Manage your Todos With Your Cards</li>
          </ul>
          <button
            disabled
            className="mt-auto w-full bg-gray-300 text-gray-600 px-4 py-3 rounded-lg font-medium cursor-not-allowed"
          >
            Current Plan
          </button>
        </div>

        {/* Monthly Plan */}
        <div className="bg-white shadow-md rounded-lg p-6 w-full max-w-sm flex flex-col">
          <h3 className="text-xl font-semibold text-indigo-700 mb-2">Monthly</h3>
          <p className="text-gray-700 mb-1">$10 / month</p>
          <p className="text-sm text-green-600 mb-3">30-day free trial included</p>
          <ul className="text-left mb-6 space-y-2">
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Create and Organize Cards</li>
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Knowledge Linking and Organization</li>
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Manage your Todos With Your Cards</li>
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> AI Agents with Tool-Using Capabilities</li>
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> AI Entity and Fact Extraction</li>
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Card Summarization and Analysis</li>
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Early Access To New Features</li>
          </ul>
          <button
            onClick={() => startSubscription("monthly")}
            disabled={loading}
            className="mt-auto w-full bg-indigo-600 text-white px-4 py-3 rounded-lg font-medium hover:bg-indigo-700 transition-colors disabled:opacity-50"
          >
            {loading ? "Redirecting..." : "Choose Monthly"}
          </button>
        </div>

        {/* Annual Plan */}
        <div className="bg-white shadow-md rounded-lg p-6 w-full max-w-sm flex flex-col">
          <h3 className="text-xl font-semibold text-indigo-700 mb-2">Annual</h3>
          <p className="text-gray-700 mb-1">$100 / year <span className="text-green-600">(Save 20%)</span></p>
          <p className="text-sm text-green-600 mb-3">30-day free trial included</p>
          <ul className="text-left mb-6 space-y-2">
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Create and Organize Cards</li>
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Knowledge Linking and Organization</li>
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Manage your Todos With Your Cards</li>
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> AI Agents with Tool-Using Capabilities</li>
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> AI Entity and Fact Extraction</li>
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Card Summarization and Analysis</li>
            <li className="flex items-center"><span className="text-green-600 mr-2">✓</span> Early Access To New Features</li>
          </ul>
          <button
            onClick={() => startSubscription("annual")}
            disabled={loading}
            className="mt-auto w-full bg-indigo-600 text-white px-4 py-3 rounded-lg font-medium hover:bg-indigo-700 transition-colors disabled:opacity-50"
          >
            {loading ? "Redirecting..." : "Choose Annual"}
          </button>
        </div>
      </div>

      {error && (
        <p className="mt-4 text-red-600 bg-red-50 p-2 rounded border border-red-200 text-center">
          {error}
        </p>
      )}
    </div>
  );
}
