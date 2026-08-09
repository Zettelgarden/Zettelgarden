import React, {
  useEffect,
  useState,
  createContext,
  useContext,
  ReactNode,
} from 'react';
import {
  checkAdmin,
  updateUser as apiUpdateUser,
  getUserSubscription,
} from '../api/users';
import { getBillingStatus } from '../api/billing';
import { getCurrentUser } from '../api/users';
import { LoginResponse } from '../models/Auth';
import { User, UserSubscription, defaultUser } from '../models/User';
import { isAuthError } from '../api/errors';
import { isDesktopApp } from '../data/tauriStorageAdapter';

// Last-known profile cache so the desktop app can open instantly offline
// (Phase 2b: the token lives in the OS keychain; the profile is not secret).
const USER_PROFILE_KEY = 'zg_user_profile';

function cacheUserProfile(user: User): void {
  try {
    localStorage.setItem(USER_PROFILE_KEY, JSON.stringify(user));
  } catch {
    // Non-fatal: profile caching is a startup nicety.
  }
}

function restoreUserProfile(): User | null {
  try {
    const raw = localStorage.getItem(USER_PROFILE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as User;
    return parsed && typeof parsed.id === 'number' ? parsed : null;
  } catch {
    return null;
  }
}

interface AuthContextType {
  isAuthenticated: boolean;
  isLoading: boolean;
  isAdmin: boolean;
  hasSubscription: boolean;
  loginUser: (data: LoginResponse) => void;
  loginUserFromToken: (token: string) => Promise<void>;
  logoutUser: () => void;
  currentUser: User | null;
  user: User | null;
  updateUser: (user: User) => void;
  refreshSubscription: () => Promise<boolean>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isAdmin, setIsAdmin] = useState(false);
  const [isLoading, setIsLoading] = useState(true); // Added loading state
  const [hasSubscription, setHasSubscription] = useState<boolean>(false);
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [user, setUser] = useState<User | null>(null);

  const updateUser = async (updatedUser: User) => {
    const response = await apiUpdateUser(updatedUser);
    setUser(response);
    setCurrentUser(response);
    cacheUserProfile(response);
  };

  useEffect(() => {
    const initializeAuth = async () => {
      setIsLoading(true);
      // Desktop: the keychain-backed localStorage shim primes asynchronously
      // (keychain IPC). Await it so the token read below can't miss a valid
      // keychain token on a slow/locked keychain and boot logged-out.
      if (isDesktopApp()) {
        const zg = (window as any).zgDesktop as
          | { ready?: Promise<unknown> }
          | undefined;
        await zg?.ready?.catch(() => undefined);
      }
      const token = localStorage.getItem('token');
      if (token) {
        try {
          setIsAuthenticated(true);
          const adminStatus = await checkAdmin();
          setIsAdmin(adminStatus);
          const currentUser = await getCurrentUser();
          setCurrentUser(currentUser);
          setUser(currentUser);
          cacheUserProfile(currentUser);
          if (currentUser && currentUser.id) {
            const subscription = await getUserSubscription(currentUser.id);
            const billing = await getBillingStatus();
            setHasSubscription(
              !billing.enabled ||
                (!!subscription &&
                  (subscription.stripe_subscription_status === 'active' ||
                    subscription.stripe_subscription_status === 'trialing')),
            );
          } else {
            setHasSubscription(false);
          }
        } catch (error) {
          console.error('Failed to initialize auth:', error);
          if (isDesktopApp() && !isAuthError(error)) {
            // Offline startup (NOT a bad token): keep the keychain session
            // and restore the last-known profile so the app opens into its
            // data (the local mirror serves cards/tasks/tags). Subscription
            // status is unknown until online — default to subscribed so the
            // self-hosted offline app never dead-ends on a paywall.
            const cached = restoreUserProfile();
            if (cached) {
              setCurrentUser(cached);
              setUser(cached);
              setIsAdmin(cached.is_admin);
              setHasSubscription(true);
            } else {
              setCurrentUser(defaultUser);
              setUser(defaultUser);
              setHasSubscription(true);
            }
          } else {
            // Expired/revoked token (or web app): force re-login rather than
            // pretending to be offline — a 401 would otherwise show a green
            // "Synced" indicator while every push fails and the outbox grows.
            logoutUser();
          }
        }
      }
      setIsLoading(false);
    };
    initializeAuth();
  }, []);

  const loginUser = async (data: LoginResponse) => {
    localStorage.setItem('token', data['access_token']);
    localStorage.setItem('username', data['user']['username']);
    cacheUserProfile(data['user']);
    const billing = await getBillingStatus();
    // When billing is disabled on this instance, everyone is treated as
    // subscribed (paywalls off) — there is nothing to upgrade to.
    setHasSubscription(
      !billing.enabled ||
        data['user'].stripe_subscription_status === 'active' ||
        data['user'].stripe_subscription_status === 'trialing',
    );
    setIsAuthenticated(true);
  };

  const loginUserFromToken = async (token: string) => {
    localStorage.setItem('token', token);
    setIsAuthenticated(true);

    try {
      // Fetch user data to get username and subscription info
      const adminStatus = await checkAdmin();
      setIsAdmin(adminStatus);
      const currentUser = await getCurrentUser();
      setCurrentUser(currentUser);
      setUser(currentUser);
      cacheUserProfile(currentUser);

      // Set username in localStorage like regular login
      if (currentUser && currentUser.username) {
        localStorage.setItem('username', currentUser.username);
      }

      // Update subscription status
      if (currentUser && currentUser.id) {
        const subscription = await getUserSubscription(currentUser.id);
        const billing = await getBillingStatus();
        setHasSubscription(
          !billing.enabled ||
            (!!subscription &&
              (subscription.stripe_subscription_status === 'active' ||
                subscription.stripe_subscription_status === 'trialing')),
        );
      } else {
        setHasSubscription(false);
      }
    } catch (error) {
      console.error('Failed to fetch user data after OAuth login:', error);
      // Don't logout on error, just continue with basic auth
    }
  };

  const refreshSubscription = async (): Promise<boolean> => {
    try {
      const currentUser = await getCurrentUser();
      if (currentUser && currentUser.id) {
        const subscription = await getUserSubscription(currentUser.id);
        const billing = await getBillingStatus();
        const isActive =
          !billing.enabled ||
          (!!subscription &&
            (subscription.stripe_subscription_status === 'active' ||
              subscription.stripe_subscription_status === 'trialing'));
        setHasSubscription(isActive);
        return isActive;
      }
    } catch (error) {
      console.error('Failed to refresh subscription:', error);
    }
    setHasSubscription(false);
    return false;
  };

  const logoutUser = () => {
    localStorage.removeItem('token');
    // The profile cache is not a secret, but it is account-scoped — don't
    // let a stale cross-account profile survive logout.
    localStorage.removeItem(USER_PROFILE_KEY);
    setIsAuthenticated(false);
    setIsAdmin(false); // Reset admin status on logout
  };

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        isLoading,
        isAdmin,
        hasSubscription,
        loginUser,
        loginUserFromToken,
        logoutUser,
        currentUser,
        user,
        updateUser,
        refreshSubscription,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
