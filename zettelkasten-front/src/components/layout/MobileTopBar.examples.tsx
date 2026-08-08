import React from 'react';
import { MobileTopBar } from './MobileTopBar';

/**
 * This file contains usage examples for the MobileTopBar component.
 * These are not meant to be run as tests, but as documentation.
 */

export function MobileTopBarExamples() {
  return (
    <div>
      {/* Example 1: Simple title only */}
      <MobileTopBar title="Settings" />

      {/* Example 2: With back button */}
      <MobileTopBar
        title="Article Details"
        onBack={() => console.log('Back clicked')}
      />

      {/* Example 3: With back button and action */}
      <MobileTopBar
        title="Edit Profile"
        onBack={() => console.log('Back clicked')}
        actions={
          <button
            onClick={() => console.log('Save clicked')}
            className="p-2 -mr-2 text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded-lg font-medium text-sm"
          >
            Save
          </button>
        }
      />

      {/* Example 4: With menu button and badge */}
      <MobileTopBar
        title="RSS Feeds"
        badge={5}
        onMenuClick={() => console.log('Menu clicked')}
        actions={
          <button
            onClick={() => console.log('Settings clicked')}
            className="p-2 -mr-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
            aria-label="Settings"
          >
            <svg
              className="w-6 h-6"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
              />
            </svg>
          </button>
        }
      />

      {/* Example 5: With multiple actions */}
      <MobileTopBar
        title="Messages"
        badge="3 new"
        onBack={() => console.log('Back clicked')}
        actions={
          <div className="flex items-center gap-1">
            <button
              onClick={() => console.log('Search clicked')}
              className="p-2 -mr-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
              aria-label="Search"
            >
              <svg
                className="w-6 h-6"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                />
              </svg>
            </button>
            <button
              onClick={() => console.log('More clicked')}
              className="p-2 -mr-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
              aria-label="More options"
            >
              <svg
                className="w-6 h-6"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"
                />
              </svg>
            </button>
          </div>
        }
      />

      {/* Example 6: With custom z-index and className */}
      <MobileTopBar
        title="Modal Header"
        onBack={() => console.log('Close clicked')}
        zIndex={50}
        className="shadow-md"
        actions={
          <button
            onClick={() => console.log('Done clicked')}
            className="p-2 -mr-2 text-blue-600 font-medium text-sm"
          >
            Done
          </button>
        }
      />

      {/* Example 7: Badge with large number */}
      <MobileTopBar
        title="Notifications"
        badge={150}
        onMenuClick={() => console.log('Menu clicked')}
      />

      {/* Example 8: Visible on all screen sizes */}
      <MobileTopBar
        title="Always Visible"
        onBack={() => console.log('Back clicked')}
        mobileOnly={false}
        actions={
          <button className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors">
            Action
          </button>
        }
      />
    </div>
  );
}
