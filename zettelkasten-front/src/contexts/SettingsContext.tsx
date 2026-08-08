import React, {
  createContext,
  useContext,
  useEffect,
  useState,
  ReactNode,
} from 'react';
import { getSettings, AppSettings } from '../api/settings';
import { setBaseTitle } from '../utils/title';

interface SettingsContextType {
  /** Runtime admin settings; null until the public fetch resolves. */
  settings: AppSettings | null;
}

const SettingsContext = createContext<SettingsContextType>({ settings: null });

/**
 * Provides the instance's public runtime settings (site name, signups/mail
 * toggles, support email) to the whole app. Fetched once at boot regardless
 * of auth state so unauthenticated pages (login) can react to signups_enabled.
 * Also applies site_name to the document title base.
 */
export const SettingsProvider = ({ children }: { children: ReactNode }) => {
  const [settings, setSettings] = useState<AppSettings | null>(null);

  useEffect(() => {
    let cancelled = false;
    getSettings().then((s) => {
      if (cancelled) return;
      setSettings(s);
      setBaseTitle(s.siteName);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <SettingsContext.Provider value={{ settings }}>
      {children}
    </SettingsContext.Provider>
  );
};

export function useSettings(): SettingsContextType {
  return useContext(SettingsContext);
}
