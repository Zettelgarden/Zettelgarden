import React, { useState, useRef } from "react";
import { BacklinkInputDropdownList } from "../cards/BacklinkInputDropdownList";
import { PartialCard } from "../../models/Card";

interface ChatInputProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: (referencedCards?: string[]) => void;
  onCardReference: (cardIds: string[]) => void;
  placeholder?: string;
  disabled?: boolean;
  isLoading?: boolean;
  submitButtonText?: string;
  className?: string;
  multiline?: boolean;
}

export function ChatInput({
  value,
  onChange,
  onSubmit,
  onCardReference,
  placeholder = "Type your message...",
  disabled = false,
  isLoading = false,
  submitButtonText = "Send",
  className = "",
  multiline = false,
}: ChatInputProps) {
  const [showCardDropdown, setShowCardDropdown] = useState(false);
  const [referencedCards, setReferencedCards] = useState<Set<string>>(new Set());
  const [atTriggerPosition, setAtTriggerPosition] = useState(0);
  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement>(null);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const newValue = e.target.value;
    const cursorPosition = e.target.selectionStart || 0;

    onChange(newValue);

    // Check for @ trigger
    const textBeforeCursor = newValue.substring(0, cursorPosition);
    const lastAtIndex = textBeforeCursor.lastIndexOf('@');

    if (lastAtIndex !== -1) {
      const textAfterAt = textBeforeCursor.substring(lastAtIndex + 1);
      // Only show dropdown if @ is at start or preceded by whitespace, and no whitespace after @
      const charBeforeAt = lastAtIndex > 0 ? textBeforeCursor[lastAtIndex - 1] : ' ';
      if ((charBeforeAt === ' ' || lastAtIndex === 0) && !textAfterAt.includes(' ')) {
        setAtTriggerPosition(lastAtIndex);
        setShowCardDropdown(true);
      } else {
        setShowCardDropdown(false);
      }
    } else {
      setShowCardDropdown(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && (!multiline || !e.shiftKey)) {
      e.preventDefault();
      onSubmit(referencedCards.size > 0 ? Array.from(referencedCards) : undefined);
    } else if (e.key === 'Escape') {
      setShowCardDropdown(false);
    }
  };

  const handleCardSelect = (card: PartialCard) => {
    if (!inputRef.current) return;

    const input = inputRef.current;
    const currentValue = value;

    // Replace from @ to current cursor position with the card reference
    const beforeAt = currentValue.substring(0, atTriggerPosition);
    const afterAt = currentValue.substring(atTriggerPosition + 1); // Skip the @ character
    const cardReference = `@[${card.title}]`;
    const newValue = beforeAt + cardReference + afterAt;

    onChange(newValue);

    // Add card to referenced cards set
    const newReferencedCards = new Set(referencedCards).add(String(card.id));
    setReferencedCards(newReferencedCards);
    onCardReference(Array.from(newReferencedCards));

    // Position cursor after the card reference
    const newCursorPosition = atTriggerPosition + cardReference.length;
    setTimeout(() => {
      input.setSelectionRange(newCursorPosition, newCursorPosition);
      input.focus();
    }, 0);

    setShowCardDropdown(false);
  };

  const handleCardDropdownSearch = (searchTerm: string) => {
    // BacklinkInputDropdownList handles its own search
  };

  const InputComponent = multiline ? 'textarea' : 'input';

  return (
    <div className={`relative ${className}`}>
      <div className="flex gap-3">
        <InputComponent
          ref={inputRef as any}
          type={multiline ? undefined : "text"}
          value={value}
          onChange={handleInputChange}
          onKeyPress={handleKeyPress}
          placeholder={placeholder}
          className="flex-1 px-4 py-3 text-sm border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all duration-200"
          disabled={disabled}
          rows={multiline ? 3 : undefined}
          style={multiline ? { resize: 'none' } : undefined}
        />
        <button
          onClick={() => onSubmit(referencedCards.size > 0 ? Array.from(referencedCards) : undefined)}
          disabled={!value.trim() || disabled || isLoading}
          className="px-6 py-3 bg-blue-500 hover:bg-blue-600 text-white rounded-xl font-medium transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
        >
          {isLoading ? (
            <>
              <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
              <span>Starting...</span>
            </>
          ) : (
            <>
              <span>{submitButtonText}</span>
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
              </svg>
            </>
          )}
        </button>
      </div>

      {/* Card Selection Modal */}
      {showCardDropdown && (
        <div
          className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
          onClick={() => setShowCardDropdown(false)}
        >
          <div
            className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4">
              <h3 className="text-lg font-semibold text-gray-900 mb-2">Select a Card</h3>
              <p className="text-sm text-gray-600">Choose a card to reference in your message</p>
            </div>
            <BacklinkInputDropdownList
              onSelect={handleCardSelect}
              onSearch={handleCardDropdownSearch}
              placeholder="Search cards..."
              className="w-full"
              autoFocus={true}
            />
            <div className="mt-4 flex justify-end">
              <button
                onClick={() => setShowCardDropdown(false)}
                className="px-4 py-2 text-sm text-gray-600 hover:text-gray-800 transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}