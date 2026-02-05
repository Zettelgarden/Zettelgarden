import React, { useState, useEffect } from "react";
import { 
  sendMailingListMessage, 
  getMailingListMessages, 
  getMailingListSubscribers,
  MailingListMessage, 
  MailingListSubscriber 
} from "../../api/mailingList";

export function AdminMailingListSendPage() {
  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");
  const [testEmail, setTestEmail] = useState("");
  const [isSending, setIsSending] = useState(false);
  const [result, setResult] = useState<{ success: boolean; message: string } | null>(null);
  const [sentMessages, setSentMessages] = useState<MailingListMessage[]>([]);
  const [expandedMessageId, setExpandedMessageId] = useState<number | null>(null);
  const [subscribers, setSubscribers] = useState<MailingListSubscriber[]>([]);
  const [showConfirmation, setShowConfirmation] = useState(false);
  const [pendingAction, setPendingAction] = useState<{ isTest: boolean } | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [messages, subs] = await Promise.all([
          getMailingListMessages(),
          getMailingListSubscribers()
        ]);
        setSentMessages(messages || []);
        setSubscribers(subs || []);
      } catch (error) {
        console.error("Error fetching data:", error);
        setSentMessages([]);
        setSubscribers([]);
      }
    };
    fetchData();
  }, []);

  const handleSubmitClick = (isTest: boolean) => {
    setPendingAction({ isTest });
    setShowConfirmation(true);
  };

  const handleConfirmSend = async () => {
    if (!pendingAction) return;
    
    const { isTest } = pendingAction;
    setShowConfirmation(false);
    setPendingAction(null);
    
    setIsSending(true);
    setResult(null);
    
    try {
      // Get active subscribers' emails
      const subscriberEmails = subscribers
        .filter(sub => {
          // Log each subscriber's status to help debug
          console.log(`Subscriber ${sub.email}: subscribed=${sub.subscribed}`);
          return sub.subscribed === true;  // Explicit comparison
        })
        .map(sub => sub.email);

      console.log(`Found ${subscriberEmails.length} active subscribers:`, subscriberEmails);

      const response = await sendMailingListMessage({
        subject,
        body,
        to_recipients: isTest ? [testEmail] : [],
        bcc_recipients: isTest ? [] : subscriberEmails,
      });
      
      setResult({
        success: response.success,
        message: isTest 
          ? `Test email sent to ${testEmail}`
          : `Message sent to ${subscriberEmails.length} subscribers`,
      });

      if (!isTest) {
        const messages = await getMailingListMessages();
        setSentMessages(messages || []);
      }

      if (response.success) {
        setSubject("");
        setBody("");
        setTestEmail("");
      }
    } catch (error) {
      console.error("Error sending message:", error);
      setResult({
        success: false,
        message: error instanceof Error ? error.message : "An error occurred",
      });
    } finally {
      setIsSending(false);
    }
  };

  const handleCancelSend = () => {
    setShowConfirmation(false);
    setPendingAction(null);
  };

  const toggleMessageExpand = (messageId: number) => {
    setExpandedMessageId(expandedMessageId === messageId ? null : messageId);
  };

  const activeSubscriberCount = subscribers.filter(sub => sub.subscribed).length;

  return (
    <div className="container mx-auto px-4">
      <h1 className="text-2xl font-bold mb-6">Send Mailing List Message</h1>
      
      <div className="bg-white shadow-md rounded px-8 pt-6 pb-8 mb-4">
        <div className="mb-4">
          <div className="flex justify-between items-center">
            <label className="block text-gray-700 text-sm font-bold mb-2" htmlFor="subject">
              Subject
            </label>
            <span className="text-sm text-gray-600">
              {activeSubscriberCount} active subscribers
            </span>
          </div>
          <input
            id="subject"
            type="text"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
            placeholder="Enter message subject"
          />
        </div>

        <div className="mb-6">
          <label className="block text-gray-700 text-sm font-bold mb-2" htmlFor="body">
            Message Body
          </label>
          <textarea
            id="body"
            value={body}
            onChange={(e) => setBody(e.target.value)}
            className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline min-h-[120px] max-h-[40vh] sm:max-h-none resize-y"
            rows={10}
            placeholder="Enter message body"
          />
        </div>

        <div className="mb-6">
          <label className="block text-gray-700 text-sm font-bold mb-2" htmlFor="testEmail">
            Test Email (optional)
          </label>
          <input
            id="testEmail"
            type="email"
            value={testEmail}
            onChange={(e) => setTestEmail(e.target.value)}
            className="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
            placeholder="Enter email address for testing"
          />
        </div>

        {result && (
          <div className={`mb-4 p-4 rounded ${result.success ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>
            {result.message}
          </div>
        )}

        <div className="flex gap-4">
          <button
            onClick={() => handleSubmitClick(false)}
            disabled={isSending || !subject || !body || activeSubscriberCount === 0}
            className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded focus:outline-none focus:shadow-outline disabled:opacity-50"
          >
            {isSending ? "Sending..." : `Send to All Subscribers (${activeSubscriberCount})`}
          </button>
          
          <button
            onClick={() => handleSubmitClick(true)}
            disabled={isSending || !subject || !body || !testEmail}
            className="bg-gray-500 hover:bg-gray-700 text-white font-bold py-2 px-4 rounded focus:outline-none focus:shadow-outline disabled:opacity-50"
          >
            {isSending ? "Sending..." : "Send Test Email"}
          </button>
        </div>
      </div>

      {/* Confirmation Modal */}
      {showConfirmation && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-lg p-6 max-w-md w-full">
            <h2 className="text-xl font-bold mb-4">Confirm Send</h2>
            <p className="mb-6">
              {pendingAction?.isTest 
                ? `Are you sure you want to send a test email to ${testEmail}?`
                : `Are you sure you want to send this message to ${activeSubscriberCount} subscribers?`}
            </p>
            <div className="flex justify-end gap-4">
              <button
                onClick={handleCancelSend}
                className="px-4 py-2 text-gray-600 hover:text-gray-800"
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmSend}
                className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
              >
                Yes, Send
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Sent Messages History */}
      <div className="bg-white shadow-md rounded px-8 pt-6 pb-8">
        <h2 className="text-xl font-bold mb-4">Sent Messages History</h2>
        <div className="overflow-x-auto">
          <table className="min-w-full table-auto">
            <thead>
              <tr className="bg-gray-100">
                <th className="px-4 py-2 text-left">Subject</th>
                <th className="px-4 py-2 text-left">Sent At</th>
                <th className="px-4 py-2 text-left">Recipients</th>
                <th className="px-4 py-2 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {sentMessages.map((message) => (
                <React.Fragment key={message.id}>
                  <tr className="border-b hover:bg-gray-50">
                    <td className="px-4 py-2">{message.subject}</td>
                    <td className="px-4 py-2">{new Date(message.sent_at).toLocaleString()}</td>
                    <td className="px-4 py-2">{message.total_recipients}</td>
                    <td className="px-4 py-2">
                      <button
                        onClick={() => toggleMessageExpand(message.id)}
                        className="text-blue-500 hover:text-blue-700"
                      >
                        {expandedMessageId === message.id ? "Hide Content" : "Show Content"}
                      </button>
                    </td>
                  </tr>
                  {expandedMessageId === message.id && (
                    <tr className="bg-gray-50">
                      <td colSpan={4} className="px-4 py-4">
                        <div className="whitespace-pre-wrap">{message.body}</div>
                      </td>
                    </tr>
                  )}
                </React.Fragment>
              ))}
              {sentMessages.length === 0 && (
                <tr>
                  <td colSpan={4} className="px-4 py-2 text-center text-gray-500">
                    No messages sent yet
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
} 