import { apiClient, getData } from "./client";
import { buildURL } from "./client";

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
  return getData(apiClient.post<ChatConversation>("/chat/conversations", params));
}

export function getConversations(primaryCardId?: number): Promise<ChatConversation[]> {
  const params: Record<string, number | undefined> = {};
  if (primaryCardId !== undefined) {
    params.primary_card_id = primaryCardId;
  }
  return getData(apiClient.get<ChatConversation[]>("/chat/conversations", { params }))
    .then((data) => data || []);
}

export function getConversation(conversationId: string): Promise<ConversationWithMessages> {
  return getData(apiClient.get<ConversationWithMessages>(`/chat/conversations/${conversationId}`));
}

export function sendMessage(conversationId: string, content: string, referencedCards?: string[], model?: string): Promise<ChatMessage[]> {
  const payload: SendMessageRequest = { content };
  if (referencedCards && referencedCards.length > 0) {
    payload.referenced_cards = referencedCards;
  }
  if (model) {
    payload.model = model;
  }

  return getData(apiClient.post<ChatMessage[]>(`/chat/conversations/${conversationId}/messages`, payload));
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
  const BASE_URL = import.meta.env.VITE_URL;
  const url = buildURL(BASE_URL, `/chat/conversations/${conversationId}/messages/stream`);

  const payload: SendMessageRequest = { content };
  if (referencedCards && referencedCards.length > 0) {
    payload.referenced_cards = referencedCards;
  }
  if (model) {
    payload.model = model;
  }

  return new Promise((resolve, reject) => {
    apiClient.fetchResponse(url, {
      method: "POST",
      body: JSON.stringify(payload),
    })
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
  return getData(apiClient.delete<void>(`/chat/conversations/${conversationId}`));
}

export function starConversation(conversationId: string): Promise<ChatConversation> {
  return getData(apiClient.post<ChatConversation>(`/chat/conversations/${conversationId}/star`));
}

export function updateConversationTitle(conversationId: string, title: string): Promise<ChatConversation> {
  return getData(apiClient.put<ChatConversation>(`/chat/conversations/${conversationId}/title`, { title }));
}

export function getUsageQuotas(): Promise<UsageQuota[]> {
  return getData(apiClient.get<UsageQuota[]>("/chat/usage"));
}

export function getConversationStatus(conversationId: string): Promise<ConversationStatus> {
  return getData(apiClient.get<ConversationStatus>(`/chat/conversations/${conversationId}/status`));
}

export function getChatInstructions(): Promise<ChatInstructions> {
  return getData(apiClient.get<ChatInstructions>("/chat/instructions"));
}

export function updateChatInstructions(instructions: string): Promise<ChatInstructions> {
  return getData(apiClient.put<ChatInstructions>("/chat/instructions", { instructions }));
}

export function regenerateMessage(conversationId: string, messageId: string): Promise<ChatMessage> {
  return getData(apiClient.post<ChatMessage>(`/chat/conversations/${conversationId}/messages/${messageId}/regenerate`));
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
  return getData(apiClient.post<RetryToolCallResponse>(`/chat/conversations/${conversationId}/tools/retry`, request));
}

export interface EditUserMessageRequest {
  content: string;
}

export function editUserMessage(conversationId: string, messageId: string, request: EditUserMessageRequest): Promise<ConversationWithMessages> {
  return getData(apiClient.put<ConversationWithMessages>(`/chat/conversations/${conversationId}/messages/${messageId}/edit`, request));
}
