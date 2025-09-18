import React from 'react';
import { useNavigate } from 'react-router-dom';
import { setDocumentTitle } from '../utils/title';
import { useAuth } from '../contexts/AuthContext';

export function GettingStartedPage({ setShowGettingStarted }) {
  const navigate = useNavigate();
  const { hasSubscription } = useAuth();

  React.useEffect(() => {
    setDocumentTitle("Welcome to Zettelgarden");
  }, []);

  return (
    <div className="min-h-screen bg-gradient-to-br from-green-50 to-blue-50 flex items-center justify-center p-6">
      <div className="max-w-2xl mx-auto">
        <div className="bg-white rounded-2xl shadow-xl p-8 text-center">
          {/* Welcome Header */}
          <div className="mb-8">
            <div className="text-6xl mb-4">🌱</div>
            <h1 className="text-4xl font-bold text-gray-900 mb-3">
              Welcome to Zettelgarden!
            </h1>
            <p className="text-lg text-gray-600">
              Your personal space for growing ideas and connecting thoughts
            </p>
          </div>

          {/* Quick Start Steps */}
          <div className="text-left mb-8 space-y-4">
            <h2 className="text-xl font-semibold text-gray-800 mb-4 text-center">
              Let's get you started:
            </h2>

            <div className="space-y-3">
              <div className="flex items-start gap-3 p-3 bg-green-50 rounded-lg">
                <span className="flex-shrink-0 w-6 h-6 bg-green-500 text-white rounded-full flex items-center justify-center text-sm font-medium">1</span>
                <div>
                  <p className="font-medium text-gray-800">Create your first card</p>
                  <p className="text-sm text-gray-600">Press 'c' anywhere or use the + button to create a note</p>
                </div>
              </div>

              <div className="flex items-start gap-3 p-3 bg-blue-50 rounded-lg">
                <span className="flex-shrink-0 w-6 h-6 bg-blue-500 text-white rounded-full flex items-center justify-center text-sm font-medium">2</span>
                <div>
                  <p className="font-medium text-gray-800">Connect your ideas</p>
                  <p className="text-sm text-gray-600">Link cards together by mentioning them with @cardname</p>
                </div>
              </div>

              <div className="flex items-start gap-3 p-3 bg-purple-50 rounded-lg">
                <span className="flex-shrink-0 w-6 h-6 bg-purple-500 text-white rounded-full flex items-center justify-center text-sm font-medium">3</span>
                <div>
                  <p className="font-medium text-gray-800">Search & discover</p>
                  <p className="text-sm text-gray-600">Press 's' to search through your knowledge base</p>
                </div>
              </div>

              {hasSubscription && (
                <div className="flex items-start gap-3 p-3 bg-yellow-50 rounded-lg">
                  <span className="flex-shrink-0 w-6 h-6 bg-yellow-500 text-white rounded-full flex items-center justify-center text-sm font-medium">4</span>
                  <div>
                    <p className="font-medium text-gray-800">Chat with your knowledge</p>
                    <p className="text-sm text-gray-600">Use AI to analyze and discuss your cards</p>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex flex-col sm:flex-row gap-4 justify-center mb-6">
            <button
              onClick={() => { setShowGettingStarted(false); navigate('/app') }}
              className="bg-green-600 hover:bg-green-700 text-white font-semibold py-3 px-6 rounded-lg transition-colors duration-200 flex items-center justify-center gap-2"
            >
              <span>🚀</span>
              Start Creating
            </button>

            <button
              onClick={() => { setShowGettingStarted(false); navigate('/app/help') }}
              className="bg-gray-100 hover:bg-gray-200 text-gray-700 font-semibold py-3 px-6 rounded-lg transition-colors duration-200 flex items-center justify-center gap-2"
            >
              <span>📚</span>
              View Help Center
            </button>
          </div>

          {/* Tips */}
          <div className="text-sm text-gray-500 bg-gray-50 rounded-lg p-4">
            <p className="mb-2">
              <strong>Pro tip:</strong> Use keyboard shortcuts to work faster:
            </p>
            <div className="flex flex-wrap justify-center gap-4 text-xs">
              <span><kbd className="bg-white px-2 py-1 rounded border">c</kbd> Create card</span>
              <span><kbd className="bg-white px-2 py-1 rounded border">t</kbd> Create task</span>
              <span><kbd className="bg-white px-2 py-1 rounded border">s</kbd> Search</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}