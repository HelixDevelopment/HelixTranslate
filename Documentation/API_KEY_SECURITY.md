# API Key Security Guide

## IMPORTANT: NEVER Hardcode API Keys

For security reasons, **NEVER** hardcode API keys in source code. Always use environment variables or secure configuration files.

## Best Practices

### 1. Use Environment Variables

Set API keys as environment variables:

```bash
# In your shell
export OPENAI_API_KEY="your-openai-key"
export ANTHROPIC_API_KEY="your-anthropic-key"
export ZHIPU_API_KEY="your-zhipu-key"
export DEEPSEEK_API_KEY="your-deepseek-key"

# Or in a .env file (ensure it's in .gitignore)
cat > .env << EOF
OPENAI_API_KEY=your-openai-key
ANTHROPIC_API_KEY=your-anthropic-key
ZHIPU_API_KEY=your-zhipu-key
DEEPSEEK_API_KEY=your-deepseek-key
EOF
```

### 2. Add .env to .gitignore

Always exclude .env files from version control:

```bash
echo ".env" >> .gitignore
```

### 3. Load Keys in Python

Use the `os.environ.get()` pattern to securely load keys:

```python
import os

openai_key = os.environ.get("OPENAI_API_KEY")
anthropic_key = os.environ.get("ANTHROPIC_API_KEY")
zhipu_key = os.environ.get("ZHIPU_API_KEY")
deepseek_key = os.environ.get("DEEPSEEK_API_KEY")
```

## Why This Matters

1. **Security**: Hardcoded keys can be exposed in git history
2. **Flexibility**: Different keys for different environments
3. **Compliance**: Follow security best practices
4. **Collaboration**: Team members use their own keys

## For Development

When testing or debugging, you can temporarily set keys:

```bash
# For single command
OPENAI_API_KEY="your-key" python3 script.py

# For session
export OPENAI_API_KEY="your-key"
python3 script.py
```

## In Production

Use your cloud provider's secret management service:
- AWS Secrets Manager
- Google Cloud Secret Manager
- Azure Key Vault
- Docker secrets
- Kubernetes secrets

## Detection

The repository includes checks to prevent hardcoded API keys from being committed. Look for patterns like:
- `sk-` (OpenAI/DeepSeek format)
- Long alphanumeric strings (32+ characters)
- Direct assignment to API key variables

## Recovery

If you accidentally commit an API key:
1. Remove the key from the code
2. Revoke the API key from the provider
3. Generate a new key
4. Rewrite git history to remove the key completely

Remember: Treat API keys like passwords - they provide access to your accounts and billing information!