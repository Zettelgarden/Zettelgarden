import os
import requests
import json

# --- Configuration ---
BASE_URL = "http://localhost:8079"  # Or whatever your backend URL is
EMAIL = os.environ.get("ZETTEL_USER")  # Replace with your test user's email
PASSWORD = os.environ.get("ZETTEL_PASSWORD")  # Replace with your test user's password


def login():
    """Logs in to the application and returns a JWT token."""
    print("Attempting to log in...")
    try:
        response = requests.post(
            f"{BASE_URL}/api/login", json={"email": EMAIL, "password": PASSWORD}
        )
        response.raise_for_status()
        token = response.json().get("token")
        if not token:
            print("Login failed: 'token' not found in response.")
            return None
        print("Login successful.")
        return token
    except requests.exceptions.RequestException as e:
        print(f"Login failed: {e}")
        return None


def get_conversations(token):
    """Fetches all conversations for the user."""
    print("\n--- Fetching conversations ---")
    headers = {"Authorization": f"Bearer {token}"}
    try:
        response = requests.get(f"{BASE_URL}/api/chat/conversations", headers=headers)
        response.raise_for_status()
        conversations = response.json()
        print("Found conversations:")
        print(json.dumps(conversations, indent=2))
        return conversations
    except requests.exceptions.RequestException as e:
        print(f"Failed to get conversations: {e}")
        return None


def get_conversation_messages(token, conversation_id):
    """Fetches all messages for a specific conversation."""
    print(f"\n--- Fetching messages for conversation {conversation_id} ---")
    headers = {"Authorization": f"Bearer {token}"}
    try:
        response = requests.get(
            f"{BASE_URL}/api/chat/conversations/{conversation_id}", headers=headers
        )
        response.raise_for_status()
        messages = response.json()
        print("Found messages:")
        print(json.dumps(messages, indent=2))
        return messages
    except requests.exceptions.RequestException as e:
        print(f"Failed to get messages: {e}")
        return None


def start_new_chat(token, message, card_pks=None):
    """Starts a new chat conversation."""
    print("\n--- Starting a new chat ---")
    headers = {"Authorization": f"Bearer {token}"}
    payload = {"message": message}
    if card_pks:
        payload["card_pks"] = card_pks
        print(f"Sending message with context from cards: {card_pks}")
    else:
        print("Sending message with no card context.")

    try:
        response = requests.post(
            f"{BASE_URL}/api/chat/completions", headers=headers, json=payload
        )
        response.raise_for_status()
        chat_response = response.json()
        print("Received response:")
        print(json.dumps(chat_response, indent=2))
        return chat_response
    except requests.exceptions.RequestException as e:
        print(f"Failed to start new chat: {e}")
        return None


def continue_chat(token, conversation_id, message, card_pks=None):
    """Continues an existing chat conversation."""
    print(f"\n--- Continuing chat in conversation {conversation_id} ---")
    headers = {"Authorization": f"Bearer {token}"}
    payload = {"conversation_id": conversation_id, "message": message}
    if card_pks:
        payload["card_pks"] = card_pks
        print(f"Sending message with context from cards: {card_pks}")
    else:
        print("Sending message with no card context.")

    try:
        response = requests.post(
            f"{BASE_URL}/api/chat/completions", headers=headers, json=payload
        )
        response.raise_for_status()
        chat_response = response.json()
        print("Received response:")
        print(json.dumps(chat_response, indent=2))
        return chat_response
    except requests.exceptions.RequestException as e:
        print(f"Failed to continue chat: {e}")
        return None


if __name__ == "__main__":
    jwt_token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOjEsImZyZXNoIjp0cnVlLCJ0eXBlIjoiYWNjZXNzIiwiZXhwIjoxNzU4NzExNzg3LCJpYXQiOjE3NTc0MTU3ODd9.kThVseTJe4rZImTsK60omsKjlxsDlkUjJs--XSez4XY"
    if jwt_token:
        # 1. Start a new conversation
        start_new_chat(jwt_token, "Hello, what is the capital of France?")

        # 2. Get all conversations and find the latest one
        all_conversations = get_conversations(jwt_token)
        if all_conversations:
            latest_conversation_id = all_conversations[0]["id"]

            # 3. Get messages for that conversation
            get_conversation_messages(jwt_token, latest_conversation_id)

            # 4. Continue the conversation
            continue_chat(
                jwt_token, latest_conversation_id, "Thanks! What about Germany?"
            )

            # 5. Example with card context (replace with actual card PKs from your DB)
            # start_new_chat(jwt_token, "Summarize the main points of the provided cards.", card_pks=[1, 2, 3])
