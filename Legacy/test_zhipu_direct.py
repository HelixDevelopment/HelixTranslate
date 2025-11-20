#!/usr/bin/env python3
"""Test Zhipu AI with direct API call to bypass any issues"""

import os
import openai

def test_zhipu_direct():
    api_key = os.environ.get("ZHIPU_API_KEY")
    
    if not api_key:
        print("❌ ZHIPU_API_KEY environment variable not set")
        return False
    
    try:
        client = openai.OpenAI(
            api_key=api_key,
            base_url="https://open.bigmodel.cn/api/paas/v4"
        )
        
        print("🔍 Testing Zhipu AI direct...")
        
        # Try a simple translation
        response = client.chat.completions.create(
            model="glm-4.5",
            messages=[
                {"role": "system", "content": "You are a professional Russian to Serbian translator."},
                {"role": "user", "content": "Translate: Привет мир"}
            ],
            max_tokens=100,
            temperature=0.3
        )
        
        translation = response.choices[0].message.content.strip()
        print(f"✅ Translation: {translation}")
        return True
        
    except Exception as e:
        print(f"❌ Error: {e}")
        return False

if __name__ == "__main__":
    success = test_zhipu_direct()
    if success:
        print("🎉 Zhipu AI is working!")
    else:
        print("❌ Zhipu AI failed")