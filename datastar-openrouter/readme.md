## Nexus Chat

Welcome to the Nexus Chat project! This application allows users to talk to many ai models using OpenRouter

### Features

- Real-time Messaging
- Responsive Design
- Integrated Web Search Tool
- Image & PDF Summarization
- Image Generation
- Ability to retry an answer from ai model with a different ai model

### Model/Library/Frameworks used

- OpenRouter (connect to many models/ let openrouter auto mode to auto select a model)
- Golang , templ , chi
- Datastar
- Goldmark Markdown
- Tailwind, Open Props
- Turso

### Installation

To get started with the Nexus Chat application, follow these steps:

1. Clone the repository:
   ```bash
   git clone https://github.com/gouthamrangarajan/go.git
   ```
2. Navigate to the project directory:
   ```bash
   cd datastar-openrouter
   ```
3. Create a .env file with following values

   - COOKIE_SECRET
   - DEFAULT_MODEL_ID
   - ENV
   - OPEN_ROUTER_API_KEY
   - OPEN_ROUTER_API_URL
   - OPEN_ROUTER_EMBEDDING_URL
   - OPEN_ROUTER_EMBEDDING_MODEL
   - TURSO_AUTH_TOKEN
   - TURSO_DATABASE_URL

4. Use Go & Templ (Terminal 1)
   ```bash
    templ generate --watch --proxy="http://localhost:3000" --cmd="go run ."
   ```
5. Use Tailwind cli (Terminal 2)
   ```bash
   npx @tailwindcss/cli -i ./input.css -o ./assets/css/styles.css --watch
   ```

### Usage

To run the application, use the following command:

Open your browser and navigate to `http://127.0.0.1:7331/` to start chatting!

### Deployed version

[rg-nexus-chat](https://rg-nexus-chat.up.railway.app/)

![screenshot](screenshot.png)
