import React, { createContext, useContext, useState } from "react";

interface EditorMessagesContextValue {
  message: string;
  setMessage: (message: string) => void;
  error: string;
  setError: (error: string) => void;
}

const EditorMessagesContext = createContext<EditorMessagesContextValue | undefined>(undefined);

export function useEditorMessagesContext() {
  const context = useContext(EditorMessagesContext);
  if (!context) {
    throw new Error("useEditorMessagesContext must be used within EditorMessagesProvider");
  }
  return context;
}

interface EditorMessagesProviderProps {
  children: React.ReactNode;
}

export function EditorMessagesProvider({ children }: EditorMessagesProviderProps) {
  const [message, setMessage] = useState<string>("");
  const [error, setError] = useState<string>("");

  return (
    <EditorMessagesContext.Provider value={{ message, setMessage, error, setError }}>
      {children}
    </EditorMessagesContext.Provider>
  );
}
