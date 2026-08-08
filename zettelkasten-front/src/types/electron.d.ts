export {};

declare global {
  interface Window {
    electronAPI?: {
      platform: string;
      minimize: () => Promise<void>;
      maximize: () => Promise<void>;
      close: () => Promise<void>;
      isMaximized: () => Promise<boolean>;
      onMenuAction: (callback: (action: string) => void) => void;
    };
  }
}
