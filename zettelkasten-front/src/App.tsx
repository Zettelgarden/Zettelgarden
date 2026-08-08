import React, { useEffect } from 'react';
import { Admin } from './pages/admin/AdminPage';
import LoginForm from './pages/LoginPage';
import MainApp from './pages/MainApp';
import RegisterPage from './pages/RegisterPage';
import { Routes, Route, Navigate } from 'react-router-dom';
import PasswordReset from './pages/PasswordReset';
import EmailValidation from './pages/EmailValidation';
import { useAuth } from './contexts/AuthContext';
import { RssManagePage } from './pages/RssManagePage';

import { useNavigate, useLocation } from 'react-router-dom';

function App() {
  const navigate = useNavigate();
  const location = useLocation();
  const { isAuthenticated, isLoading } = useAuth();

  // When the SPA is served on a real path like /admin (not a hash route),
  // HashRouter sees the hash as empty (route = "/"). Redirect to the
  // corresponding hash route so React Router can match it properly.
  useEffect(() => {
    const realPath = window.location.pathname;
    const hash = window.location.hash;
    if ((hash === '' || hash === '#' || hash === '#/') && realPath !== '/') {
      navigate(realPath + window.location.search, { replace: true });
    }
  }, [navigate]);

  return (
    <div>
      <Routes>
        {/* Root serves the app shell, never marketing (6er.14): authenticated
            users land in the app, everyone else is sent to login. */}
        <Route
          path="/"
          element={
            isLoading ? (
              <div />
            ) : (
              <Navigate to={isAuthenticated ? '/app' : '/login'} replace />
            )
          }
        />
        <Route path="/app/rss/manage" element={<RssManagePage />} />
        <Route path="/app/*" element={<MainApp />} />
        <Route path="/admin/*" element={<Admin />} />
        <Route path="/login" element={<LoginForm />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/reset" element={<PasswordReset />} />
        <Route path="/validate" element={<EmailValidation />} />
      </Routes>
    </div>
  );
}

export default App;
