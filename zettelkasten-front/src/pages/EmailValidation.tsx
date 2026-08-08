import React, { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { validateEmail } from '../api/users';

function EmailValidation() {
  const [message, setMessage] = useState('');
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    const query = new URLSearchParams(location.search);
    const token = query.get('token');
    if (token) {
      handleValidateEmail(token);
    }
  }, [location]);

  const handleValidateEmail = async (token: string) => {
    setMessage('');
    try {
      const response = await validateEmail(token);
      if (response.error) {
        setMessage(response.message);
      } else {
        setMessage('Your email has been successfully validated.');
        navigate('/login', {
          state: { message: 'Your email has been successfully validated.' },
        });
      }
    } catch (error) {
      setMessage('Failed to validate email.');
    }
  };

  return (
    <div className="validation-container">
      <h2>Email Validation</h2>
      {message && <div>{message}</div>}
    </div>
  );
}

export default EmailValidation;
