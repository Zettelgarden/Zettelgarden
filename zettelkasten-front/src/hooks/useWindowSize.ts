import { useState, useEffect } from 'react';

interface WindowSize {
  width: number | undefined;
  height: number | undefined;
}

export function useWindowSize(): WindowSize {
  const [windowSize, setWindowSize] = useState<WindowSize>({
    width: undefined,
    height: undefined,
  });

  useEffect(() => {
    // Handler to call on window resize
    function handleResize() {
      // Set window width/height to state
      setWindowSize({
        width: window.innerWidth,
        height: window.innerHeight,
      });
    }

    // Only run on client side (avoid SSR issues)
    if (typeof window !== 'undefined') {
      // Set initial size
      handleResize();

      // Add event listener
      window.addEventListener('resize', handleResize);

      // Remove event listener on cleanup
      return () => window.removeEventListener('resize', handleResize);
    }
  }, []); // Empty array ensures effect only runs on mount and unmount

  return windowSize;
}

export function useIsDesktop(breakpoint: number = 1024): boolean {
  const { width } = useWindowSize();
  const [isDesktop, setIsDesktop] = useState<boolean>(false);

  useEffect(() => {
    if (width !== undefined) {
      setIsDesktop(width >= breakpoint);
    }
  }, [width, breakpoint]);

  return isDesktop;
}