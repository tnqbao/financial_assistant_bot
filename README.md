# AI Assistant using Gemini-1.5-Flash

This is a simple chatbot that uses the Gemini-1.5-Flash model to generate responses to user queries. The chatbot is
built using the Rasa framework and the Gemini-1.5-Flash model. It can answer questions, provide information, and engage
in conversations with users. The chatbot is designed to be user-friendly and customizable to meet the needs of different
users.

## Platforms Intergrated

<ul>
    <li>Telegram</li>
    <li>Update soon ...</li>
</ul>

## Prerequisites

<ul>
  <li>Go 1.19 or newer</li>
  <li><a href="https://www.postman.com/" target="_blank">Postman</a> (optional, for API testing)</li>
</ul>

## Installation

* Clone the repository:
   ``` bash
    git clone https://github.com/tnqbao/financial_management_bot.git
    cd financial_management_bot
   ```
* Setup your module:
  ``` bash
   go mod edit -module=your-link-github-repo 
  ```
* Install dependencies:
  ``` bash
    go mod tidy 
  ``` 

  * Set up environment variables:
      * Create a `.env` file in the project root and configure the following variables:
    ```dotenv
      GEMINI_API_KEYS=your-gemini-api-key (1 or more split by comma ',') 
      GEMINI_API_URL=your-gemini-api-url
      POSTGRES_USER=your-postgres-user
      POSTGRES_PASSWORD=your-postgres-password
      POSTGRES_DB=your-postgres-db
      POSTGRES_HOST=your-postgres-host
      TELEGRAM_BOT_TOKEN=your-telegram-bot-token
    ```

* Start the server:
    ``` bash 
    go run main.go
    ```