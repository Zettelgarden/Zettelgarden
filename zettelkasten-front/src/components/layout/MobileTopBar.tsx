import React, { ReactNode } from 'react';

interface MobileTopBarProps {
  /**
   * The title to display in the top bar
   */
  title: string;

  /**
   * Optional badge text or number to display next to the title
   * Examples: "5", "99+", or "New"
   */
  badge?: string | number;

  /**
   * Optional callback for the back button
   * If provided, shows a back button on the left side
   */
  onBack?: () => void;

  /**
   * Optional callback for a menu button
   * If provided and onBack is not provided, shows a hamburger menu button on the left side
   */
  onMenuClick?: () => void;

  /**
   * Optional action buttons or elements to display on the right side
   * Can be a single element or multiple elements wrapped in a fragment
   */
  actions?: ReactNode;

  /**
   * Optional custom class name for additional styling
   */
  className?: string;

  /**
   * Optional z-index value for the top bar
   * @default 40
   */
  zIndex?: number;

  /**
   * Whether to show the component only on mobile devices
   * @default true
   */
  mobileOnly?: boolean;
}

/**
 * A generic mobile top bar component for responsive pages.
 *
 * Features:
 * - Sticky positioning at the top of the viewport
 * - Left side: Back button, menu button, or custom left action
 * - Center: Title with optional badge
 * - Right side: Action buttons or custom content
 * - Responsive: Only visible on mobile devices by default (hidden on md+)
 *
 * @example
 * ```tsx
 * // With back button and actions
 * <MobileTopBar
 *   title="Article"
 *   onBack={() => navigate(-1)}
 *   actions={
 *     <button onClick={handleEdit}>Edit</button>
 *   }
 * />
 *
 * // With menu button and badge
 * <MobileTopBar
 *   title="RSS"
 *   badge={unreadCount}
 *   onMenuClick={() => setShowMenu(true)}
 *   actions={<SettingsButton />}
 * />
 *
 * // Simple title only
 * <MobileTopBar title="Settings" />
 * ```
 */
export function MobileTopBar({
  title,
  badge,
  onBack,
  onMenuClick,
  actions,
  className = '',
  zIndex = 40,
  mobileOnly = true,
}: MobileTopBarProps) {
  const visibilityClass = mobileOnly ? 'md:hidden' : '';

  // Format badge for display
  const displayBadge = badge !== undefined ? String(badge) : null;
  const shouldShowBadge =
    displayBadge !== null && displayBadge !== '' && displayBadge !== '0';

  return (
    <div
      className={`sticky top-0 bg-white border-b border-gray-200 px-4 py-3 flex items-center justify-between ${visibilityClass} ${className}`}
      style={{ zIndex }}
    >
      {/* Left side: Back button, Menu button, or spacer */}
      <div className="flex items-center">
        {onBack && (
          <button
            onClick={onBack}
            className="p-2 -ml-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors"
            aria-label="Go back"
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
                d="M15 19l-7-7 7-7"
              />
            </svg>
          </button>
        )}

        {!onBack && onMenuClick && (
          <button
            onClick={onMenuClick}
            className="p-2 -ml-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors"
            aria-label="Open menu"
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
                d="M4 6h16M4 12h16M4 18h16"
              />
            </svg>
          </button>
        )}
      </div>

      {/* Center: Title with optional badge */}
      <div className="flex items-center gap-2 flex-1 mx-4">
        <h1 className="text-lg font-semibold text-gray-900 truncate">
          {title}
        </h1>
        {shouldShowBadge && (
          <span className="bg-red-500 text-white text-xs font-bold px-2 py-0.5 rounded-full flex-shrink-0">
            {displayBadge}
          </span>
        )}
      </div>

      {/* Right side: Action buttons */}
      <div className="flex items-center gap-1">{actions}</div>
    </div>
  );
}

/**
 * Props for the left action button slot
 */
interface MobileTopBarLeftActionProps {
  children: ReactNode;
  onClick?: () => void;
  ariaLabel?: string;
}

/**
 * A helper component for custom left actions in MobileTopBar.
 * Use this when you need a custom left button instead of the default back or menu buttons.
 *
 * @example
 * ```tsx
 * <MobileTopBar
 *   title="Custom"
 *   leftAction={<MobileTopBarLeftAction onClick={handleCustomAction}>Custom</MobileTopBarLeftAction>}
 * />
 * ```
 */
export function MobileTopBarLeftAction({
  children,
  onClick,
  ariaLabel = 'Action',
}: MobileTopBarLeftActionProps) {
  return (
    <button
      onClick={onClick}
      className="p-2 -ml-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors"
      aria-label={ariaLabel}
    >
      {children}
    </button>
  );
}

export default MobileTopBar;
