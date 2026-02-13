# Core Mobile Backend & Slack Integration

The central business logic hub of the ecosystem, written in Go, featuring deep integration with Slack for administrative management.

## 📋 Overview
This repository serves as the primary backend for the mobile application. It manages the recruitment lifecycle, handles secure communication between candidates and HR, and implements a comprehensive Slack-based management module.

## 🛠 Tech Stack
* **Language:** Go (Golang)
* **Database:** PostgreSQL (NeonDB) with `pg_cron` for automated token management.
* **Security:** AES-GCM encryption for sensitive data and secure hashing for tokens.
* **Integrations:** Slack API, Google Translate API, Firebase Cloud Messaging (FCM).

## ✨ Key Features

### Slack Management Interface
* **Interactive Control:** HR manages candidates and documents directly through interactive Slack modals and messages.
* **Slash Commands:** Custom commands like `/doloci_status` (set status) and `/obvesti` (notify) to trigger backend logic directly from the Slack interface.
* **Automated Translation:** Integrates Google Translate API to bridge the language gap between candidates and HR staff in real-time.

### Security & Token Logic
* **Encrypted Session Management:** Uses encrypted and hashed tokens to manage user sessions and refresh logic securely.
* **Cryptographic Standards:** Implements AES-GCM for the protection of sensitive candidate and organizational data.
* **Automated Maintenance:** Leverages `pg_cron` to maintain database health by automatically removing expired security tokens.
* **Clean Architecture:** Follows modular design principles to separate API handlers, business logic, and external service integrations.
