# API Documentation

Complete API documentation for the Russian-Serbian FB2 Translator service.

## Base URL

- **HTTP/3 (QUIC)**: `https://localhost:8443`
- **HTTP/2 (TLS)**: `https://localhost:8443`
- **HTTP/1.1**: `http://localhost:8080` (if HTTP/3 disabled)

## Authentication

The API supports optional authentication via JWT tokens or API keys.

### JWT Authentication

Include the JWT token in the Authorization header:

```http
Authorization: Bearer <your-jwt-token>
```

### API Key Authentication

Include the API key in the header:

```http
X-API-Key: <your-api-key>
```

## Endpoints

### Health & Status

#### `GET /health`

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "time": "2025-01-15T10:30:00Z"
}
```

#### `GET /`

API information and available endpoints.

#### `GET /api/v1/providers`

List all available translation providers.

**Response:**
```json
{
  "providers": [
    {
      "name": "dictionary",
      "description": "Simple dictionary-based translation",
      "requires_api_key": false
    },
    {
      "name": "openai",
      "description": "OpenAI GPT models",
      "requires_api_key": true,
      "models": ["gpt-4", "gpt-3.5-turbo"]
    }
  ]
}
```

### Translation

#### `POST /api/v1/translate`

Translate Russian text to Serbian.

**Request:**
```json
{
  "text": "Привет, мир!",
  "provider": "dictionary",
  "model": "gpt-4",
  "context": "Literary text",
  "script": "cyrillic"
}
```

**Response:**
```json
{
  "original": "Привет, мир!",
  "translated": "Здраво, свете!",
  "provider": "dictionary",
  "session_id": "uuid",
  "stats": {
    "total": 1,
    "translated": 1,
    "cached": 0,
    "errors": 0
  }
}
```

#### `POST /api/v1/translate/fb2`

Translate a complete FB2 e-book file.

**Request:**
```http
POST /api/v1/translate/fb2
Content-Type: multipart/form-data

file: <fb2 file>
provider: dictionary
model: gpt-4
script: cyrillic
```

**Response:**
Returns the translated FB2 file as `application/xml`.

#### `POST /api/v1/translate/batch`

Batch translate multiple texts.

**Request:**
```json
{
  "texts": [
    "герой",
    "мир",
    "человек"
  ],
  "provider": "dictionary",
  "context": "Single words"
}
```

**Response:**
```json
{
  "originals": ["герой", "мир", "человек"],
  "translated": ["јунак", "свет", "човек"],
  "provider": "dictionary",
  "session_id": "uuid",
  "stats": {...}
}
```

### Script Conversion

#### `POST /api/v1/convert/script`

Convert between Cyrillic and Latin scripts.

**Request:**
```json
{
  "text": "Ратибор је јунак",
  "target": "latin"
}
```

**Response:**
```json
{
  "original": "Ратибор је јунак",
  "converted": "Ratibor je junak",
  "target": "latin"
}
```

### WebSocket

#### `GET /ws?session_id={id}`

WebSocket endpoint for real-time translation progress.

**Connection:**
```javascript
const ws = new WebSocket('wss://localhost:8443/ws?session_id=uuid');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(data.type, data.message);
};
```

**Event Types:**
- `translation_started`
- `translation_progress`
- `translation_completed`
- `translation_error`

## Provider Configuration

### Dictionary Provider

No configuration required. Uses built-in Russian-Serbian dictionary.

```json
{
  "provider": "dictionary"
}
```

### OpenAI Provider

Requires API key via environment variable or config:

```bash
export OPENAI_API_KEY="your-key"
```

```json
{
  "provider": "openai",
  "model": "gpt-4"
}
```

### Anthropic Provider

```bash
export ANTHROPIC_API_KEY="your-key"
```

```json
{
  "provider": "anthropic",
  "model": "claude-3-sonnet-20240229"
}
```

### Zhipu AI Provider

```bash
export ZHIPU_API_KEY="your-key"
```

```json
{
  "provider": "zhipu",
  "model": "glm-4"
}
```

### DeepSeek Provider

```bash
export DEEPSEEK_API_KEY="your-key"
```

```json
{
  "provider": "deepseek",
  "model": "deepseek-chat"
}
```

### Ollama Provider (Local)

Requires Ollama running locally:

```bash
ollama pull llama3:8b
```

```json
{
  "provider": "ollama",
  "model": "llama3:8b"
}
```

## Error Handling

The API returns standard HTTP status codes:

- `200 OK` - Request successful
- `400 Bad Request` - Invalid request parameters
- `401 Unauthorized` - Authentication required
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

**Error Response:**
```json
{
  "error": "Error message description"
}
```

## Rate Limiting

Default rate limits:
- **10 requests per second** per IP
- **20 burst requests** allowed

Rate limit headers:
```http
X-RateLimit-Limit: 10
X-RateLimit-Remaining: 9
X-RateLimit-Reset: 1642251600
```

## Examples

See the `/api/examples` directory for:
- **curl** scripts
- **HTTP** files
- **Postman** collection
- **WebSocket** test page
