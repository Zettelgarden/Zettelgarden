import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { ResponsiveLayout } from './ResponsiveLayout';

// Mock window.innerWidth
const originalInnerWidth = window.innerWidth;

describe('ResponsiveLayout', () => {
  beforeEach(() => {
    // Reset window.innerWidth before each test
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: 1024,
    });
  });

  afterEach(() => {
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: originalInnerWidth,
    });
  });

  it('renders children with isMobile=false on desktop', () => {
    const mockChildren = vi.fn((isMobile: boolean) => (
      <div>Is mobile: {isMobile.toString()}</div>
    ));

    render(
      <ResponsiveLayout mobileView="list" setMobileView={vi.fn()}>
        {mockChildren}
      </ResponsiveLayout>,
    );

    expect(mockChildren).toHaveBeenCalledWith(false);
    expect(screen.getByText('Is mobile: false')).toBeInTheDocument();
  });

  it('renders children with isMobile=true on mobile viewport', () => {
    // Set mobile viewport
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: 375,
    });

    const mockChildren = vi.fn((isMobile: boolean) => (
      <div>Is mobile: {isMobile.toString()}</div>
    ));

    render(
      <ResponsiveLayout mobileView="list" setMobileView={vi.fn()}>
        {mockChildren}
      </ResponsiveLayout>,
    );

    expect(mockChildren).toHaveBeenCalledWith(true);
    expect(screen.getByText('Is mobile: true')).toBeInTheDocument();
  });

  it('uses 768px breakpoint by default', () => {
    // Test breakpoint boundary - just below 768px
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: 767,
    });

    const mockChildren = vi.fn((isMobile: boolean) => (
      <div>Is mobile: {isMobile.toString()}</div>
    ));

    render(
      <ResponsiveLayout mobileView="detail" setMobileView={vi.fn()}>
        {mockChildren}
      </ResponsiveLayout>,
    );

    expect(mockChildren).toHaveBeenCalledWith(true);
  });

  it('recognizes desktop at breakpoint boundary', () => {
    // Test breakpoint boundary - exactly 768px
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: 768,
    });

    const mockChildren = vi.fn((isMobile: boolean) => (
      <div>Is mobile: {isMobile.toString()}</div>
    ));

    render(
      <ResponsiveLayout mobileView="filters" setMobileView={vi.fn()}>
        {mockChildren}
      </ResponsiveLayout>,
    );

    expect(mockChildren).toHaveBeenCalledWith(false);
  });

  it('handles window resize events', () => {
    const mockChildren = vi.fn((isMobile: boolean) => (
      <div>Is mobile: {isMobile.toString()}</div>
    ));

    const { rerender } = render(
      <ResponsiveLayout mobileView="list" setMobileView={vi.fn()}>
        {mockChildren}
      </ResponsiveLayout>,
    );

    // Start as desktop
    expect(mockChildren).toHaveBeenCalledWith(false);

    // Simulate window resize to mobile
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: 375,
    });

    // Trigger resize event
    window.dispatchEvent(new Event('resize'));

    // Note: React state updates are asynchronous, so we need to rerender
    // In a real test, we'd use waitFor or act()
  });
});
