from flask import Flask, request, jsonify
from flask_mail import Mail, Message
import os
import logging


def mail_config():
    """Returns the mail configuration derived from the environment, or None
    when mail is not configured.

    Mail is OPTIONAL for self-hosted installs (bead 6er.6): this service may
    run with every ZETTEL_MAIL_* var missing. Instead of crashing at boot, it
    serves /api/send and /api/send/mailing-list with a 503 until the operator
    provides SMTP credentials (or removes the service from compose entirely).
    """
    server = os.getenv("ZETTEL_MAIL_SERVER")
    username = os.getenv("ZETTEL_MAIL_USERNAME")
    password = os.getenv("ZETTEL_MAIL_PASSWORD")
    default_sender = os.getenv("ZETTEL_MAIL_DEFAULT_SENDER")

    if not server or not username or not password or not default_sender:
        return None

    return {
        "server": server,
        "username": username,
        "password": password,
        "default_sender": default_sender,
        "port": int(os.getenv("ZETTEL_MAIL_PORT", 587)),
        "log_file": os.getenv("ZETTEL_MAIL_LOG", "mail.log"),
    }


# Initialize Flask app (boots even when mail is not configured).
app = Flask(__name__)

# Configure mail settings from the environment. Flask-Mail does not connect at
# import time — it connects at send time — so a missing config only surfaces
# per-request (guarded in the send routes below).
cfg = mail_config()
app.config["MAIL_SERVER"] = cfg["server"] if cfg else None
app.config["MAIL_PORT"] = cfg["port"] if cfg else 587
app.config["MAIL_USE_TLS"] = True
app.config["MAIL_USERNAME"] = cfg["username"] if cfg else None
app.config["MAIL_PASSWORD"] = cfg["password"] if cfg else None
app.config["DEFAULT_SENDER"] = cfg["default_sender"] if cfg else None

mail = Mail(app)

# Configure logging
logging.basicConfig(
    filename=(cfg["log_file"] if cfg else "mail.log"),
    level=logging.INFO,  # Log level
    format="%(asctime)s - %(levelname)s - %(message)s",  # Log format
    datefmt="%Y-%m-%d %H:%M:%S",  # Date format in logs
)


def require_mail_config():
    """Returns None when mail is configured, else a (message, status) pair.

    Central guard for the send routes: missing SMTP credentials are a 503
    (service unavailable) rather than a crash or a confusing SMTP error.
    """
    if mail_config() is None:
        return (
            "Mail is not configured: set ZETTEL_MAIL_SERVER/USERNAME/PASSWORD/"
            "DEFAULT_SENDER (see .env.example) or run without the mail-service "
            "container when you don't need email.",
            503,
        )
    return None


@app.route("/api/send", methods=["POST"])
def send_mail():
    guard = require_mail_config()
    if guard:
        return jsonify({"message": guard[0]}), guard[1]

    data = request.get_json()
    subject = data.get("subject")
    recipient = data.get("recipient")
    body = data.get("body")
    is_html = data.get("is_html", False)

    if not subject or not recipient:
        return jsonify({"message": "Email needs a subject and recipient"}), 400

    # Create the email message
    message = Message(
        subject,
        sender=("Zettelgarden", app.config["DEFAULT_SENDER"]),
        recipients=[recipient],
    )

    # Set body based on whether it's HTML or plain text
    if is_html:
        message.html = body
    else:
        message.body = body

    # Send the email
    with app.app_context():
        try:
            mail.send(message)
            logging.info("Email sent successfully to %s", recipient)
            return jsonify({"message": "Email sent successfully"}), 200
        except Exception as e:
            logging.error("Error sending email: %s", str(e))
            return jsonify({"error": str(e)}), 500


@app.route("/api/send/mailing-list", methods=["POST"])
def send_mailing_list():
    guard = require_mail_config()
    if guard:
        return jsonify({"message": guard[0]}), guard[1]

    data = request.get_json()
    subject = data.get("subject")
    to_recipients = data.get("to_recipients", [])  # Main visible recipients
    bcc_recipients = data.get("bcc_recipients", [])  # BCC recipients
    body = data.get("body")
    is_html = data.get("is_html", False)

    if not subject or (not to_recipients and not bcc_recipients):
        return jsonify({"message": "Email needs a subject and at least one recipient (TO or BCC)"}), 400

    # Create the email message with BCC support
    message = Message(
        subject,
        sender=("Zettelgarden", app.config["DEFAULT_SENDER"]),
        recipients=to_recipients,
        bcc=bcc_recipients,
    )

    # Set body based on whether it's HTML or plain text
    if is_html:
        message.html = body
    else:
        message.body = body

    # Send the email
    with app.app_context():
        try:
            mail.send(message)
            total_recipients = len(to_recipients) + len(bcc_recipients)
            logging.info("Mailing list email sent successfully to %d recipients", total_recipients)
            return jsonify(
                {
                    "message": "Mailing list email sent successfully",
                    "recipients_count": total_recipients,
                }
            ), 200
        except Exception as e:
            logging.error("Error sending mailing list email: %s", str(e))
            return jsonify({"error": str(e)}), 500


@app.route("/health", methods=["GET"])
def health_check():
    # The process is alive and serving even without SMTP config; report the
    # mail state so operators/compose healthchecks can distinguish "degraded,
    # no mail" from "broken".
    return (
        jsonify(
            {
                "status": "healthy",
                "service": "zettel-mail-service",
                "mail_configured": mail_config() is not None,
            }
        ),
        200,
    )


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8081, debug=True)
