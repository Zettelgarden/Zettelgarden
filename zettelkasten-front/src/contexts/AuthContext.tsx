import React, {
  useEffect,
  useState,
  createContext,
  useContext,
  ReactNode,
} from "react";
import {
  checkAdmin,
  updateUser as apiUpdateUser,
  getUserSubscription,
} from "../api/users";
import { getCurrentUser } from "../api/users";
import { LoginResponse } from "../models/Auth";
import { User, UserSubscription } from "../models/User";

interface AuthContextType {
  isAuthenticated: boolean;
  isLoading: boolean;
  isAdmin: boolean;
  hasSubscription: boolean;
  loginUser: (data: LoginResponse) => void;
  loginUserFromToken: (token: string) => void;
  logoutUser: () => void;
  currentUser: User | null;
  user: User | null;
  updateUser: (user: User) => void;
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
  };

  useEffect(() => {
    const initializeAuth = async () => {
      setIsLoading(true);
      const token = localStorage.getItem("token");
      if (token) {
        try {
          setIsAuthenticated(true);
          const adminStatus = await checkAdmin();
          setIsAdmin(adminStatus);
          const currentUser = await getCurrentUser();
          setCurrentUser(currentUser);
          setUser(currentUser);
          if (currentUser && currentUser.id) {
            const subscription = await getUserSubscription(currentUser.id);
            setHasSubscription(
              subscription &&
              subscription.stripe_subscription_status === "active" ||
              subscription.stripe_subscription_status === "trialing",
            );
          } else {
            setHasSubscription(false);
          }
        } catch (error) {
          console.error("Failed to initialize auth:", error);
          // Gently handle auth initialization failure
          logoutUser();
        }
      }
      setIsLoading(false);
    };
    initializeAuth();
  }, []);

  const loginUser = (data: LoginResponse) => {
    localStorage.setItem("token", data["access_token"]);
    localStorage.setItem("username", data["user"]["username"]);
    setHasSubscription(
      data["user"].stripe_subscription_status === "active",
    );
    setIsAuthenticated(true);
  };

  const loginUserFromToken = (token: string) => {
    localStorage.setItem("token", token);
    setIsAuthenticated(true);
  };

  const logoutUser = () => {
    localStorage.removeItem("token");
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
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
};
