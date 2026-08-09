import React, { Fragment } from 'react';
import { Dialog, Transition } from '@headlessui/react';

export interface ModalProps {
  open: boolean;
  onClose: () => void;
  children: React.ReactNode;
  /** Standardized max-width preset (override via className for intentional per-dialog sizing) */
  size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl';
  /** Extra classes for the Dialog.Panel (sizing/rounded overrides) */
  className?: string;
  /** Extra classes for the Dialog root (e.g. z-index overrides) */
  dialogClassName?: string;
  /** Element to receive focus on open (defaults to the first focusable child) */
  initialFocus?: React.RefObject<HTMLElement>;
}

const sizeClasses: Record<NonNullable<ModalProps['size']>, string> = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-xl',
  '2xl': 'max-w-2xl',
  '3xl': 'max-w-3xl',
  '4xl': 'max-w-4xl',
};

/**
 * Modal — single shared dialog shell built on Headless UI Dialog.
 *
 * Provides overlay, sizing, Escape-to-close, focus trap, scroll-lock and
 * aria-* attributes. Controlled via `open`/`onClose`, so keyboard-shortcut
 * dialogs driven by DialogStateContext keep working unchanged.
 */
export function Modal({
  open,
  onClose,
  children,
  size = 'md',
  className = '',
  dialogClassName = '',
  initialFocus,
}: ModalProps) {
  return (
    <Transition appear show={open} as={Fragment}>
      <Dialog
        as="div"
        className={`relative z-50 ${dialogClassName}`}
        onClose={onClose}
        initialFocus={initialFocus}
      >
        {/* Overlay */}
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div
            className="fixed inset-0 bg-black bg-opacity-50"
            aria-hidden="true"
          />
        </Transition.Child>

        {/* Dialog container */}
        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4 text-center">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <Dialog.Panel
                className={`w-full ${sizeClasses[size]} transform overflow-hidden rounded-lg bg-white p-6 text-left align-middle shadow-xl transition-all ${className}`}
              >
                {children}
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
}
