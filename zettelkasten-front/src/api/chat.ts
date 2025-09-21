import { checkStatus } from "./common";

const base_url = import.meta.env.VITE_URL;

// Types for chat API
export interface ChatConversation {
  id: string;
  user_id: number;
  title?: string;
  model: string;
  system_prompt?: string;
  starred: boolean;
  created_at: string;
  updated_at: string;
  message_count?: number;
}

export interface ChatMessage {
  id: string;
  conversation_id: string;
  role: "user" | "assistant" | "system" | "tool";
  content?: string;
  tool_calls?: ChatToolCall[];
  tool_call_id?: string;
  sequence_number: number;
  referenced_cards?: string[];
  status: "pending" | "processing" | "completed" | "failed";
  created_at: string;
}

export interface ChatToolCall {
  id: string;
  type: string;
  function: {
    name: string;
    arguments: Record<string, any>;
  };
}

export interface ConversationWithMessages {
  conversation: ChatConversation;
  messages: ChatMessage[];
}

export interface CreateConversationRequest {
  title?: string;
  model?: string;
  system_prompt?: string;
}

export interface SendMessageRequest {
  content: string;
  referenced_cards?: string[];
  model?: string;
}

export interface UsageQuota {
  id: number;
  user_id: number;
  quota_type: string;
  current_usage: number;
  max_limit: number;
  reset_date: string;
  created_at: string;
  updated_at: string;
}

export interface ConversationStatus {
  conversation_id: string;
  has_pending: boolean;
  has_processing: boolean;
  has_failed: boolean;
}

export interface ChatInstructions {
  id?: number;
  user_id: number;
  instructions: string;
  created_at?: string;
  updated_at?: string;
}

// API Functions

export function createConversation(params: CreateConversationRequest): Promise<ChatConversation> {
  const url = `${base_url}/chat/conversations`;
  const token = localStorage.getItem("token");

  return fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(params),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<ChatConversation>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function getConversations(): Promise<ChatConversation[]> {
  const url = `${base_url}/chat/conversations`;
  const token = localStorage.getItem("token");

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` }
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<ChatConversation[]>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function getConversation(conversationId: string): Promise<ConversationWithMessages> {
  const url = `${base_url}/chat/conversations/${conversationId}`;
  const token = localStorage.getItem("token");

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` }
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<ConversationWithMessages>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function sendMessage(conversationId: string, content: string, referencedCards?: string[], model?: string): Promise<ChatMessage[]> {
  const url = `${base_url}/chat/conversations/${conversationId}/messages`;
  const token = localStorage.getItem("token");

  const payload: SendMessageRequest = { content };
  if (referencedCards && referencedCards.length > 0) {
    payload.referenced_cards = referencedCards;
  }
  if (model) {
    payload.model = model;
  }

  return fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<ChatMessage[]>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function deleteConversation(conversationId: string): Promise<void> {
  const url = `${base_url}/chat/conversations/${conversationId}`;
  const token = localStorage.getItem("token");

  return fetch(url, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then(() => {
      // Delete request returns 204 No Content
      return;
    });
}

export function starConversation(conversationId: string): Promise<ChatConversation> {
  const url = `${base_url}/chat/conversations/${conversationId}/star`;
  const token = localStorage.getItem("token");

  return fetch(url, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<ChatConversation>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function updateConversationTitle(conversationId: string, title: string): Promise<ChatConversation> {
  const url = `${base_url}/chat/conversations/${conversationId}/title`;
  const token = localStorage.getItem("token");

  return fetch(url, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ title }),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<ChatConversation>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function getUsageQuotas(): Promise<UsageQuota[]> {
  const url = `${base_url}/chat/usage`;
  const token = localStorage.getItem("token");

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` }
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<UsageQuota[]>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function getConversationStatus(conversationId: string): Promise<ConversationStatus> {
  const url = `${base_url}/chat/conversations/${conversationId}/status`;
  const token = localStorage.getItem("token");

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` }
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<ConversationStatus>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function getChatInstructions(): Promise<ChatInstructions> {
  const url = `${base_url}/chat/instructions`;
  const token = localStorage.getItem("token");

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` }
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<ChatInstructions>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function updateChatInstructions(instructions: string): Promise<ChatInstructions> {
  const url = `${base_url}/chat/instructions`;
  const token = localStorage.getItem("token");

  return fetch(url, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ instructions }),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<ChatInstructions>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function regenerateMessage(conversationId: string, messageId: string): Promise<ChatMessage> {
  const url = `${base_url}/chat/conversations/${conversationId}/messages/${messageId}/regenerate`;
  const token = localStorage.getItem("token");

  return fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<ChatMessage>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}