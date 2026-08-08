import React from 'react';
import { resendValidateEmail } from '../api/users';
import { User } from '../models/User';
import { useAuth } from '../contexts/AuthContext';

export function EmailValidationBanner() {
  const { currentUser } = useAuth();
  function resendEmail() {
    resendValidateEmail();
  }
  if (!currentUser) {
    return <div></div>;
  }
  if (currentUser['email_validated']) {
    return <div></div>;
  }
  return (
    <div className="bg-amber-50 border-l-4 border-amber-500 p-2 w-full">
      <div className="flex items-center justify-between max-w-7xl mx-auto">
        <div className="flex items-center space-x-2">
          <svg
            className="h-4 w-4 text-amber-400"
            viewBox="0 0 20 20"
            fill="currentColor"
          >
            <path
              fillRule="evenodd"
              d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
          <p className="text-sm text-amber-700">
            We've sent a verification email to your inbox.{' '}
            <button
              onClick={resendEmail}
              className="font-medium underline hover:text-amber-800 focus:outline-none"
            >
              Click here to resend the verification email
            </button>
          </p>
        </div>
      </div>
    </div>
  );
}
