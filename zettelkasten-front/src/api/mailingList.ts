import { apiClient, getData } from "./client";

export interface SendMailingListMessageParams {
  subject: string;
  body: string;
  to_recipients: string[];
  bcc_recipients: string[];
}

export interface SendMailingListMessageResponse {
  success: boolean;
  message: string;
  recipientCount?: number;
}

export interface MailingListMessage {
  id: number;
  subject: string;
  body: string;
  sent_at: string;
  total_recipients: number;
}

export interface MailingListSubscriber {
  id: number;
  email: string;
  welcome_email_sent: boolean;
  subscribed: boolean;
  has_account: boolean;
  created_at: string;
  updated_at: string;
}

export async function getMailingListSubscribers(): Promise<MailingListSubscriber[]> {
  return getData(apiClient.get<MailingListSubscriber[]>("/mailing-list"));
}

export async function sendMailingListMessage(params: SendMailingListMessageParams): Promise<SendMailingListMessageResponse> {
  return getData(apiClient.post<SendMailingListMessageResponse>("/mailing-list/messages/send", params));
}

export async function getMailingListMessages(): Promise<MailingListMessage[]> {
  return getData(apiClient.get<MailingListMessage[]>("/mailing-list/messages"));
}
