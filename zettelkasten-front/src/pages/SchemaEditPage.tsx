import React, { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { SchemaDialog } from "../components/schemas/SchemaDialog";
import { fetchSchema } from "../api/schemas";
import { SchemaDefinition } from "../models/Schema";
import { setDocumentTitle } from "../utils/title";

export function SchemaEditPage() {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [schema, setSchema] = useState<SchemaDefinition | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setDocumentTitle("Edit Schema");
    if (!id) {
      setError("Schema ID is required");
      setLoading(false);
      return;
    }

    fetchSchema(parseInt(id))
      .then((data) => {
        setSchema(data);
        setLoading(false);
      })
      .catch((err) => {
        setError("Failed to load schema");
        setLoading(false);
        console.error("Error fetching schema:", err);
      });
  }, [id]);

  const handleSuccess = () => {
    navigate("/app/schemas");
  };

  const handleClose = () => {
    navigate("/app/schemas");
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-gray-500">Loading schema...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-red-600">{error}</div>
      </div>
    );
  }

  if (!schema) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-red-600">Schema not found</div>
      </div>
    );
  }

  return <SchemaDialog schema={schema} isOpen={true} onClose={handleClose} onSuccess={handleSuccess} />;
}
