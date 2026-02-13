# Core Mobile Backend & Slack Integration

The central business logic hub of the ecosystem, written in Go, featuring deep integration with Slack for administrative management.

## 📋 Overview
This repository serves as the primary backend for the mobile application. Beyond handling mobile APIs and security, it implements a comprehensive **Slack-based management module**, allowing HR staff to manage the entire recruitment process without leaving their communication platform.

## 🛠 Tech Stack
* **Language:** Go (Golang)
* **Database:** PostgreSQL (NeonDB)
* **Integrations:** Slack API, Google Translate API, Firebase (FCM)

## ✨ Key Features

### Slack Integration 🤖
* **Administrative Control:** HR manages candidates and documents directly through interactive Slack modals and messages.
* **Slash Commands:** Custom commands like `/doloci status` (set status) and `/obvesti` (notify) to trigger backend logic from the Slack command line.
* **Automated Translation:** Uses Google Translate API to automatically translate messages from candidates into the HR staff's preferred language within the Slack thread.

### Core Backend Services ⚙️
* **Security & Encryption:** Uses AES-GCM for sensitive data encryption, SHA256 for general hashing and Bcrypt for password hashing.
* **Complex Data Modeling:** Implements Recursive CTEs to handle nested chat threads and candidate communication history.
* **Session Management:** Robust JWT access and refresh token logic, with automated cleanup of expired sessions.
