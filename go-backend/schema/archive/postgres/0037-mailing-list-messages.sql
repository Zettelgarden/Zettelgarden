CREATE TABLE IF NOT EXISTS mailing_list_messages (
    id SERIAL PRIMARY KEY,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_recipients INT
);

CREATE TABLE IF NOT EXISTS mailing_list_recipients (
    id SERIAL PRIMARY KEY,
    message_id INT NOT NULL,
    recipient_email TEXT NOT NULL,
    recipient_type TEXT NOT NULL, -- 'to' or 'bcc'
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id) REFERENCES mailing_list_messages(id)
);