import React from 'react';
import { Combobox as HeadlessCombobox } from '@headlessui/react';

export interface ComboboxProps<T> {
  /** Currently selected item (null when the input is free text) */
  value: T | null;
  /** Called with the item the user picks from the list */
  onChange: (value: T) => void;
  /** Raw input text */
  inputValue: string;
  /** Called on every keystroke */
  onInputChange: (value: string) => void;
  /** Maps the selected item back to input text (defaults to String(item)) */
  displayValue?: (item: T) => string;
  placeholder?: string;
  autoFocus?: boolean;
  disabled?: boolean;
  /** Wrapper classes */
  className?: string;
  /** Input classes (appended) */
  inputClassName?: string;
  /** Inline styles for the input (e.g. 16px font to prevent mobile zoom) */
  inputStyle?: React.CSSProperties;
  /** Options panel classes (positioning/sizing overrides) */
  optionsClassName?: string;
  /** Show a loading row instead of options */
  isLoading?: boolean;
  /** Loading row content */
  loadingLabel?: React.ReactNode;
  /** Shown when the list is open with zero options (and not loading) */
  emptyState?: React.ReactNode;
  /** Items to list (filtering/ranking is the caller's job) */
  options: T[];
  getOptionKey: (item: T) => string;
  /** Render one option row; `active` is the keyboard/pointer highlight */
  renderOption: (item: T, active: boolean) => React.ReactElement;
}

/**
 * Combobox — searchable autocomplete on Headless UI Combobox.
 *
 * The caller owns the query (debounce/API/ranking) and passes `options` back;
 * the primitive owns input rendering, the options list, and keyboard
 * navigation (Arrow keys + Enter, Escape to close). Options render inline
 * (flowing) by default to match existing surfaces; pass `optionsClassName`
 * with `absolute` etc. to float them.
 */
export function Combobox<T>({
  value,
  onChange,
  inputValue,
  onInputChange,
  displayValue = (item: T) => String(item),
  placeholder,
  autoFocus = false,
  disabled = false,
  className = '',
  inputClassName = '',
  inputStyle,
  optionsClassName = '',
  isLoading = false,
  loadingLabel = 'Loading...',
  emptyState = 'No results found',
  options,
  getOptionKey,
  renderOption,
}: ComboboxProps<T>) {
  return (
    <HeadlessCombobox value={value} onChange={onChange}>
      <div className={`relative w-full ${className}`}>
        <HeadlessCombobox.Input
          className={`w-full px-3 py-2 min-h-[44px] text-gray-700 bg-white border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-200 transition-all duration-200 ${inputClassName}`}
          placeholder={placeholder}
          displayValue={displayValue}
          onChange={(e) => onInputChange(e.target.value)}
          autoFocus={autoFocus}
          disabled={disabled}
          style={inputStyle}
        />
        <HeadlessCombobox.Options
          className={`mt-1 w-full overflow-hidden bg-white rounded-lg shadow-lg border border-gray-200 max-h-60 overflow-y-auto text-sm z-10 ${optionsClassName}`}
        >
          {isLoading ? (
            <div className="p-2 text-gray-500">{loadingLabel}</div>
          ) : options.length > 0 ? (
            options.map((item) => (
              <HeadlessCombobox.Option key={getOptionKey(item)} value={item}>
                {({ active }) => renderOption(item, active)}
              </HeadlessCombobox.Option>
            ))
          ) : (
            <div className="p-2 text-gray-500">{emptyState}</div>
          )}
        </HeadlessCombobox.Options>
      </div>
    </HeadlessCombobox>
  );
}
