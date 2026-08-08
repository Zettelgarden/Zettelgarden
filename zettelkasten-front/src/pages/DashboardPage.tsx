import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { CardList } from '../components/cards/CardList';
import { setDocumentTitle } from '../utils/title';
import { useUIState } from '../contexts/UIStateContext';
import { semanticSearchCardsPaginated, getUnsortedCards } from '../api/cards';
import { PartialCard } from '../models/Card';
import { AddArticleDialog } from '../components/cards/AddArticleDialog';
import { useDialogState } from '../contexts/DialogStateContext';
import { MobileTopBar } from '../components/layout/MobileTopBar';

export function DashboardPage() {
  const navigate = useNavigate();
  const { toggleMobileSidebar } = useUIState();
  const [recentCards, setRecentCards] = useState<PartialCard[]>([]);
  const [unsortedCards, setUnsortedCards] = useState<PartialCard[]>([]);
  const [searchInput, setSearchInput] = useState('');
  const [isLoadingCards, setIsLoadingCards] = useState<boolean>(true);
  const [showAddArticleDialog, setShowAddArticleDialog] = useState(false);
  const { setShowCreateTaskWindow } = useDialogState();

  const handleSearchKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      const trimmed = searchInput.trim();
      if (trimmed) {
        navigate(`/app/search?term=${encodeURIComponent(trimmed)}`);
      }
    }
  };

  const fetchDashboardData = async () => {
    setIsLoadingCards(true);
    try {
      // Fetch recent cards using pagination (only get 10)
      const recentResponse = await semanticSearchCardsPaginated(
        '', // empty search term
        false, // fullText
        false, // showEntities
        false, // showFacts
        true, // showCards
        false, // showEmails
        'sortCreatedNewOld', // sortBy recent
        'classic', // searchType
        false, // rerank
        1, // page
        10, // perPage - only get 10 cards
      );
      console.log('recent', recentResponse);

      const recentCards: PartialCard[] = recentResponse.results.map(
        (result) => ({
          id: Number(result.metadata?.id) || 0,
          card_id: result.metadata?.card_id || '',
          title: result.title,
          body: result.preview || '',
          tags: result.tags || [],
          is_deleted: false,
          created_at: new Date(result.created_at),
          updated_at: new Date(result.updated_at),
          parent_id: result.metadata?.parent_id || 0,
          user_id: 0,
          link: '',
          parent: null,
        }),
      );
      console.log('recent', recentCards);
      setRecentCards(recentCards);

      // Fetch unsorted cards using the dedicated API endpoint
      const unsortedResponse = await getUnsortedCards(1, 10);
      setUnsortedCards(unsortedResponse.cards);
    } catch (error) {
      console.error('Error fetching dashboard data:', error);
      // Set empty arrays on error to prevent infinite loading
      setRecentCards([]);
      setUnsortedCards([]);
    } finally {
      setIsLoadingCards(false);
    }
  };

  useEffect(() => {
    setDocumentTitle('Index');
    fetchDashboardData();
  }, []);

  const handleNewStandardCard = () => {
    navigate('/app/card/new', { state: { cardType: 'standard' } });
  };

  const handleNewTask = () => {
    setShowCreateTaskWindow(true);
  };

  const handleAddArticle = () => {
    setShowAddArticleDialog(true);
  };

  return (
    <div>
      {/* Mobile Top Bar */}
      <MobileTopBar title="Dashboard" onMenuClick={toggleMobileSidebar} />

      {/* Main Content Section */}
      <div className="p-2">
        <div className="">
          <div className="mb-8">
            <div className="text-center">
              <h1 className="text-3xl font-bold text-gray-900 mb-3 p-10">
                Welcome to Zettelgarden 🌱
              </h1>
              <p className="text-lg text-gray-600 max-w-full mb-8">
                Your personal space for growing ideas. Create cards, connect
                thoughts, and watch your knowledge garden flourish.
              </p>
            </div>

            {/* Mobile Quick Actions - Only visible on small screens */}
            <div className="md:hidden border-t border-gray-200 bg-gray-50 px-4 py-6">
              <div className="grid grid-cols-4 gap-3">
                <button
                  onClick={handleNewStandardCard}
                  className="flex flex-col items-center p-3 bg-white rounded-lg shadow-sm hover:shadow-md hover:bg-gray-50 border border-gray-200 transition-all duration-200"
                >
                  <svg
                    className="w-6 h-6 text-blue-600 mb-1"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M12 4v16m8-8H4"
                    />
                  </svg>
                  <span className="text-xs font-medium text-gray-700">
                    Card
                  </span>
                </button>

                <button
                  onClick={handleAddArticle}
                  className="flex flex-col items-center p-3 bg-white rounded-lg shadow-sm hover:shadow-md hover:bg-gray-50 border border-gray-200 transition-all duration-200"
                >
                  <svg
                    className="w-6 h-6 text-green-600 mb-1"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                    />
                  </svg>
                  <span className="text-xs font-medium text-gray-700">
                    Article
                  </span>
                </button>

                <button
                  onClick={handleNewTask}
                  className="flex flex-col items-center p-3 bg-white rounded-lg shadow-sm hover:shadow-md hover:bg-gray-50 border border-gray-200 transition-all duration-200"
                >
                  <svg
                    className="w-6 h-6 text-orange-600 mb-1"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"
                    />
                  </svg>
                  <span className="text-xs font-medium text-gray-700">
                    Task
                  </span>
                </button>

                <button
                  onClick={() => navigate('/app/search')}
                  className="flex flex-col items-center p-3 bg-white rounded-lg shadow-sm hover:shadow-md hover:bg-gray-50 border border-gray-200 transition-all duration-200"
                >
                  <svg
                    className="w-6 h-6 text-purple-600 mb-1"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                    />
                  </svg>
                  <span className="text-xs font-medium text-gray-700">
                    Search
                  </span>
                </button>
              </div>
            </div>
            {/* Search Box */}
            <div className="max-w-4xl mx-auto mb-8">
              <div className="relative">
                <svg
                  className="absolute left-4 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                  />
                </svg>
                <input
                  type="text"
                  value={searchInput}
                  onChange={(e) => setSearchInput(e.target.value)}
                  onKeyDown={handleSearchKeyDown}
                  placeholder="Search cards..."
                  className="w-full pl-12 pr-4 py-3 text-sm border border-gray-300 rounded-xl bg-white shadow-sm hover:shadow-md transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
      <div className="flex flex-col md:flex-row border-t">
        {/* Left Section */}

        <div className="flex-shrink-0 md:w-8/12 border-r p-4 overflow-hidden">
          <a href="/app/search?recent=true">
            <span className="font-bold">Recent Cards</span>
          </a>
          {isLoadingCards ? (
            <div className="flex justify-center py-8">
              <div className="text-gray-500">Loading recent cards...</div>
            </div>
          ) : (
            <CardList
              sort={false}
              cards={recentCards}
              onCardUpdate={fetchDashboardData}
            />
          )}
          <hr />
        </div>

        {/* Right Section */}
        <div className="flex-shrink-0 md:w-4/12 border-l p-4 overflow-hidden">
          <div>
            <span className="font-bold">Unsorted Cards</span>
            {isLoadingCards ? (
              <div className="flex justify-center py-8">
                <div className="text-gray-500">Loading unsorted cards...</div>
              </div>
            ) : (
              <CardList
                cards={unsortedCards}
                onCardUpdate={fetchDashboardData}
              />
            )}
          </div>
          <hr />
        </div>
      </div>

      {/* Modal dialogs */}
      <AddArticleDialog
        show={showAddArticleDialog}
        onClose={() => setShowAddArticleDialog(false)}
      />
    </div>
  );
}
