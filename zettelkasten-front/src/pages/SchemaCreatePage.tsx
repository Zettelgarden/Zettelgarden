import React from 'react';
import { useNavigate } from 'react-router-dom';
import { SchemaDialog } from '../components/schemas/SchemaDialog';
import { setDocumentTitle } from '../utils/title';

export function SchemaCreatePage() {
  const navigate = useNavigate();

  React.useEffect(() => {
    setDocumentTitle('Create Schema');
  }, []);

  const handleSuccess = () => {
    navigate('/app/schemas');
  };

  const handleClose = () => {
    navigate('/app/schemas');
  };

  return (
    <SchemaDialog
      schema={null}
      isOpen={true}
      onClose={handleClose}
      onSuccess={handleSuccess}
    />
  );
}
