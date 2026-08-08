import React from 'react';
import { useNavigate } from 'react-router-dom';

/**
 * Shown when a free-tier user navigates to a pro-only route.
 */
export function Paywall({ feature }: { feature: string }) {
  const navigate = useNavigate();

  return (
    <div className="flex flex-col items-center justify-center py-20 px-6">
      <div className="text-5xl mb-4">🔒</div>
      <h2 className="text-2xl font-bold text-gray-800 mb-2">
        {feature} is a Pro feature
      </h2>
      <p className="text-gray-600 mb-6 max-w-md text-center">
        Upgrade to Zettelgarden Pro to unlock {feature.toLowerCase()} and other
        advanced features. Start with a 30-day free trial.
      </p>
      <button
        onClick={() => navigate('/app/subscription')}
        className="bg-indigo-600 text-white px-6 py-3 rounded-lg font-medium hover:bg-indigo-700 transition-colors"
      >
        View Plans
      </button>
    </div>
  );
}
