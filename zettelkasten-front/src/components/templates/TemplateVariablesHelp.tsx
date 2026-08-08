import React, { useState } from 'react';

/**
 * Component that displays help information about template variables
 */
export function TemplateVariablesHelp() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="relative inline-block">
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="text-blue-600 hover:text-blue-800 text-sm flex items-center"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          className="w-4 h-4 mr-1"
        >
          <path
            fillRule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM8.94 6.94a.75.75 0 11-1.5 0 .75.75 0 011.5 0zm-.25 3a.75.75 0 100 1.5.75.75 0 000-1.5z"
            clipRule="evenodd"
          />
        </svg>
        Template Variables
      </button>

      {isOpen && (
        <div className="absolute z-10 mt-2 w-72 bg-white border border-gray-200 rounded-md shadow-lg p-4">
          <div className="flex justify-between items-center mb-2">
            <h3 className="font-medium">Available Template Variables</h3>
            <button
              type="button"
              onClick={() => setIsOpen(false)}
              className="text-gray-500 hover:text-gray-700"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 20 20"
                fill="currentColor"
                className="w-5 h-5"
              >
                <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
              </svg>
            </button>
          </div>

          <p className="text-sm text-gray-600 mb-2">
            You can use these variables in your template title and body. They
            will be replaced with their actual values when you apply the
            template.
          </p>

          <ul className="text-sm space-y-1">
            <li>
              <code className="bg-gray-100 px-1 rounded">$date</code> - Current
              date (YYYY-MM-DD)
            </li>
            <li>
              <code className="bg-gray-100 px-1 rounded">$time</code> - Current
              time (HH:MM)
            </li>
            <li>
              <code className="bg-gray-100 px-1 rounded">$datetime</code> - Full
              date and time (YYYY-MM-DD HH:MM)
            </li>
            <li>
              <code className="bg-gray-100 px-1 rounded">$day</code> - Current
              day (numeric)
            </li>
            <li>
              <code className="bg-gray-100 px-1 rounded">$month</code> - Current
              month (numeric)
            </li>
            <li>
              <code className="bg-gray-100 px-1 rounded">$year</code> - Current
              year
            </li>
            <li>
              <code className="bg-gray-100 px-1 rounded">$weekday</code> -
              Current day of the week
            </li>
          </ul>

          <div className="mt-3 text-sm text-gray-600">
            <strong>Example:</strong> "Meeting Notes - $date" becomes "Meeting
            Notes - 2025-03-24"
          </div>
        </div>
      )}
    </div>
  );
}
