import React from 'react';
import { SocialLinks } from './SocialLinks';
import { useSettings } from '../contexts/SettingsContext';

export const Footer: React.FC<{ className?: string }> = ({
  className = '',
}) => {
  const { settings } = useSettings();
  const supportEmail = settings?.supportEmail;

  return (
    <footer className={`w-full border-t border-gray-200 mt-12 ${className}`}>
      <div className="max-w-screen-xl mx-auto py-12 px-4 flex flex-col sm:flex-row items-center justify-between">
        <div className="grid grid-cols-2 gap-x-12 gap-y-4 text-sm text-gray-600">
          <span className="flex items-center">
            <svg
              className="w-4 h-4 mr-2"
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path d="M10 2a6 6 0 00-6 6v3.586l-.707.707A1 1 0 004 14h12a1 1 0 00.707-1.707L16 11.586V8a6 6 0 00-6-6zM10 18a3 3 0 01-3-3h6a3 3 0 01-3 3z" />
            </svg>
            Made in Ottawa 🍁
          </span>
          {supportEmail ? (
            <a
              href={`mailto:${supportEmail}`}
              className="hover:text-gray-900 flex items-center"
            >
              <svg
                className="w-4 h-4 mr-2"
                fill="currentColor"
                viewBox="0 0 20 20"
              >
                <path d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z" />
                <path d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z" />
              </svg>
              Contact Us
            </a>
          ) : null}
          <a
            href="https://github.com/NickSavage/Zettelgarden/issues"
            className="hover:text-gray-900 flex items-center"
          >
            <svg
              className="w-4 h-4 mr-2"
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path
                fillRule="evenodd"
                d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                clipRule="evenodd"
              />
            </svg>
            Report an Issue
          </a>
          <a
            href="https://github.com/NickSavage/Zettelgarden/blob/master/CONTRIBUTING.md"
            className="hover:text-gray-900 flex items-center"
          >
            <svg
              className="w-4 h-4 mr-2"
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path
                fillRule="evenodd"
                d="M12.316 3.051a1 1 0 01.633 1.265l-4 12a1 1 0 11-1.898-.632l4-12a1 1 0 011.265-.633zM5.707 6.293a1 1 0 010 1.414L3.414 10l2.293 2.293a1 1 0 11-1.414 1.414l-3-3a1 1 0 010-1.414l3-3a1 1 0 011.414 0zm8.586 0a1 1 0 011.414 0l3 3a1 1 0 010 1.414l-3 3a1 1 0 11-1.414-1.414L16.586 10l-2.293-2.293a1 1 0 010-1.414z"
              />
            </svg>
            Contribute
          </a>
        </div>
        <SocialLinks className="mt-4 sm:mt-0" />
      </div>
    </footer>
  );
};
