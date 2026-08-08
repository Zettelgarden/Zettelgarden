import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

export default function Success() {
  const navigate = useNavigate();
  const { refreshSubscription } = useAuth();
  const [refreshing, setRefreshing] = useState(true);

  useEffect(() => {
    async function activateSubscription() {
      const isActive = await refreshSubscription();
      setRefreshing(false);
      if (isActive) {
        // Short delay so user sees the success message
        setTimeout(() => navigate('/app', { replace: true }), 1500);
      }
    }
    activateSubscription();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  if (refreshing) {
    return (
      <div className="flex flex-col items-center mt-10">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600 mb-4"></div>
        <p className="text-gray-600">Activating your subscription...</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center mt-10">
      <h1 className="text-2xl font-bold text-green-600">
        Payment Successful 🎉
      </h1>
      <p>Your subscription is now active. Redirecting to your dashboard...</p>
    </div>
  );
}
