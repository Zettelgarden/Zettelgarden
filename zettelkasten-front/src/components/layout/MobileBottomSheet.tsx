import React, { useEffect, useRef } from 'react';

interface MobileBottomSheetProps {
  isOpen: boolean;
  onClose: () => void;
  children: React.ReactNode;
  title?: string;
  maxHeight?: string;
  showCloseButton?: boolean;
}

export function MobileBottomSheet({
  isOpen,
  onClose,
  children,
  title,
  maxHeight = '80vh',
  showCloseButton = true,
}: MobileBottomSheetProps) {
  const sheetRef = useRef<HTMLDivElement>(null);
  const backdropRef = useRef<HTMLDivElement>(null);
  const touchStartRef = useRef<number | null>(null);
  const touchCurrentRef = useRef<number>(0);

  // Handle escape key and backdrop click
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      // Prevent body scroll when sheet is open
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = '';
    };
  }, [isOpen, onClose]);

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === backdropRef.current) {
      onClose();
    }
  };

  // Handle touch start for swipe down gesture
  const handleTouchStart = (e: React.TouchEvent) => {
    const touch = e.touches[0];
    touchStartRef.current = touch.clientY;
    touchCurrentRef.current = touch.clientY;
  };

  // Handle touch move for swipe down gesture
  const handleTouchMove = (e: React.TouchEvent) => {
    if (touchStartRef.current === null) return;

    const touch = e.touches[0];
    touchCurrentRef.current = touch.clientY;

    // Only track swipes moving downward
    const deltaY = touchCurrentRef.current - touchStartRef.current;
    if (deltaY > 0 && sheetRef.current) {
      // Apply transform to follow finger
      const transform = `translateY(${deltaY}px)`;
      sheetRef.current.style.transform = transform;
    }
  };

  // Handle touch end for swipe down gesture
  const handleTouchEnd = () => {
    if (touchStartRef.current === null || !sheetRef.current) return;

    const deltaY = touchCurrentRef.current - touchStartRef.current;
    const threshold = 100; // Minimum swipe distance to close

    if (deltaY > threshold) {
      onClose();
    } else {
      // Reset transform if swipe wasn't enough
      sheetRef.current.style.transform = '';
    }

    touchStartRef.current = null;
    touchCurrentRef.current = 0;
  };

  if (!isOpen) return null;

  return (
    <div
      ref={backdropRef}
      onClick={handleBackdropClick}
      className="fixed inset-0 bg-black/50 z-50 md:hidden"
      style={{ animation: 'fade-in 0.2s ease-out' }}
    >
      <div
        ref={sheetRef}
        className="fixed bottom-0 left-0 right-0 bg-white rounded-t-2xl shadow-2xl flex flex-col"
        style={{
          maxHeight,
          animation: 'slide-up 0.3s ease-out',
          transition: 'transform 0.3s ease-out',
        }}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
      >
        {/* Drag Handle */}
        <div className="flex justify-center pt-3 pb-2 px-4 flex-shrink-0">
          <div className="w-12 h-1.5 bg-gray-300 rounded-full" />
        </div>

        {/* Header */}
        {(title || showCloseButton) && (
          <div className="px-4 pb-3 border-b border-gray-200 flex-shrink-0">
            <div className="flex items-center justify-between">
              {title && (
                <h2 className="text-lg font-semibold text-gray-900">{title}</h2>
              )}
              {showCloseButton && (
                <button
                  onClick={onClose}
                  className="p-2 -mr-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg"
                  aria-label="Close"
                >
                  <svg
                    className="w-5 h-5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M6 18L18 6M6 6l12 12"
                    />
                  </svg>
                </button>
              )}
            </div>
          </div>
        )}

        {/* Content */}
        <div className="flex-1 overflow-y-auto">{children}</div>

        <style>{`
          @keyframes fade-in {
            from {
              opacity: 0;
            }
            to {
              opacity: 1;
            }
          }
          @keyframes slide-up {
            from {
              transform: translateY(100%);
            }
            to {
              transform: translateY(0);
            }
          }
        `}</style>
      </div>
    </div>
  );
}
