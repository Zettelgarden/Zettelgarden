import { checkStatus } from "./common";

const base_url = import.meta.env.VITE_URL;

// Types for chat API
export interface ChatConversation {
  id: string;
  user_id: number;
  title?: string;
  model: string;
  system_prompt?: string;
  primary_card_id?: number;
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
  _metadata?: ToolResultMetadata;
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

// Tool error types
export type ToolErrorType =
  | "network"
  | "validation"
  | "database"
  | "not_found"
  | "permission"
  | "rate_limit"
  | "timeout"
  | "unknown";

export interface ToolError {
  type: ToolErrorType;
  message: string;
  retryable: boolean;
  tool_name: string;
  arguments?: Record<string, any>;
  suggestion?: string;
}

export interface ToolResultMetadata {
  has_error?: boolean;
  arguments?: Record<string, any>;
  timestamp?: string;
  tool_name?: string;
}

export interface CreateConversationRequest {
  title?: string;
  model?: string;
  system_prompt?: string;
  primary_card_id?: number;
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

export function getConversations(primaryCardId?: number): Promise<ChatConversation[]> {
  let url = `${base_url}/chat/conversations`;
  if (primaryCardId !== undefined) {
    url += `?primary_card_id=${primaryCardId}`;
  }
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

// Streaming event types
export interface StreamEvent {
  type: 'messages' | 'title' | 'content' | 'tool_call' | 'tool_result' | 'error' | 'done';
  data: any;
}

// Enhanced tool result event data
export interface ToolResultEventData {
  tool_call_id: string;
  name: string;
  result: Record<string, any>;
  has_error?: boolean;
  arguments?: Record<string, any>;
  timestamp?: string;
}

// Callback for streaming events
export type StreamEventCallback = (event: StreamEvent) => void;

export function sendMessageStream(
  conversationId: string,
  content: string,
  onEvent: StreamEventCallback,
  referencedCards?: string[],
  model?: string
): Promise<void> {
  const url = `${base_url}/chat/conversations/${conversationId}/messages/stream`;
  const token = localStorage.getItem("token");

  const payload: SendMessageRequest = { content };
  if (referencedCards && referencedCards.length > 0) {
    payload.referenced_cards = referencedCards;
  }
  if (model) {
    payload.model = model;
  }

  return new Promise((resolve, reject) => {
    fetch(url, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    })
      .then(checkStatus)
      .then((response) => {
        if (!response || !response.body) {
          throw new Error("Response or body is undefined");
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        const readStream = async () => {
          try {
            while (true) {
              const { done, value } = await reader.read();

              if (done) {
                console.log('Stream complete');
                resolve();
                break;
              }

              // Decode the chunk and add to buffer
              buffer += decoder.decode(value, { stream: true });

              // Process complete SSE messages
              const lines = buffer.split('\n');
              buffer = lines.pop() || ''; // Keep incomplete line in buffer

              let currentEvent: string | null = null;
              let currentData: string = '';

              for (const line of lines) {
                if (line.startsWith('event: ')) {
                  currentEvent = line.slice(7).trim();
                } else if (line.startsWith('data: ')) {
                  currentData = line.slice(6).trim();
                } else if (line === '' && currentEvent && currentData) {
                  // Complete event received
                  try {
                    const parsedData = JSON.parse(currentData);
                    onEvent({
                      type: currentEvent as StreamEvent['type'],
                      data: parsedData
                    });
                  } catch (e) {
                    console.error('Failed to parse SSE data:', currentData, e);
                    // Send error event to UI
                    onEvent({
                      type: 'error',
                      data: { error: 'Failed to parse server response' }
                    });
                  }
                  currentEvent = null;
                  currentData = '';
                }
              }
            }
          } catch (error) {
            console.error('Stream read error:', error);
            // Send error event to UI before rejecting
            onEvent({
              type: 'error',
              data: { error: error instanceof Error ? error.message : 'Stream connection error' }
            });
            reject(error);
          }
        };

        readStream();
      })
      .catch(reject);
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

// Retry a failed tool call
export interface RetryToolCallRequest {
  tool_name: string;
  arguments: Record<string, any>;
}

export interface RetryToolCallResponse {
  tool_name: string;
  result: Record<string, any>;
  has_error: boolean;
}

export function retryToolCall(conversationId: string, request: RetryToolCallRequest): Promise<RetryToolCallResponse> {
  const url = `${base_url}/chat/conversations/${conversationId}/tools/retry`;
  const token = localStorage.getItem("token");

  return fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(request),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<RetryToolCallResponse>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export interface EditUserMessageRequest {
  content: string;
}

export function editUserMessage(conversationId: string, messageId: string, request: EditUserMessageRequest): Promise<ConversationWithMessages> {
  const url = `${base_url}/chat/conversations/${conversationId}/messages/${messageId}/edit`;
  const token = localStorage.getItem("token");

  return fetch(url, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(request),
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