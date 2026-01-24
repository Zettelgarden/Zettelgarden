import React from "react";
import { useParams, useNavigate } from "react-router-dom";
import { SchemaTablePage } from "../components/schemas/SchemaTablePage";

export function SchemaTableWrapper() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  if (!id) {
    navigate("/app/schemas");
    return null;
  }

  return (
    <SchemaTablePage
      schemaId={parseInt(id)}
      onBack={() => navigate("/app/schemas")}
    />
  );
}
