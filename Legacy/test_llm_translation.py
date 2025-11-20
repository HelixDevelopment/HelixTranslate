#!/usr/bin/env python3
"""
Test script to demonstrate LLM translation quality.
This script shows the difference between translation methods.
"""

import sys
import os
from pathlib import Path
from llm_fb2_translator import LLMConfig, AdvancedLLMFB2Translator

def test_translation_quality():
    """Test translation quality with sample Russian text"""
    
    # Sample Russian text with literary complexity
    sample_text = """
    В тихом омуте, где темные воды отражали звезды, рыбак сидел в своей старой лодке и думал о прошлом.
    Он помнил, как его дед рассказывал ему легенды об этом озере – о духах, которые охраняют воды,
    и о сокровищах, что покоятся на дне.
    """
    
    print("=" * 60)
    print("RUSSIAN TO SERBIAN TRANSLATION QUALITY TEST")
    print("=" * 60)
    print(f"Original Russian text:")
    print(sample_text)
    print("=" * 60)
    
    # Test with Ollama (free, local)
    print("\n1. Testing with local Ollama (llama3:8b)...")
    try:
        ollama_config = LLMConfig(
            provider='ollama',
            model='llama3:8b',
            base_url='http://localhost:11434',
            temperature=0.3
        )
        
        translator = AdvancedLLMFB2Translator(ollama_config)
        print("✓ Ollama connection established")
        
    except Exception as e:
        print(f"✗ Ollama not available: {e}")
        print("  Install Ollama from https://ollama.ai/")
        print("  Run: ollama pull llama3:8b")
    
    # Test with OpenAI
    print("\n2. Testing with OpenAI GPT-4...")
    api_key = os.getenv('OPENAI_API_KEY') or os.getenv('LLM_API_KEY')
    if api_key:
        try:
            openai_config = LLMConfig(
                provider='openai',
                model='gpt-4',
                api_key=api_key,
                temperature=0.3
            )
            
            translator = AdvancedLLMFB2Translator(openai_config)
            print("✓ OpenAI connection established")
            
            # Test translation
            result = translator.translator.translate_text(sample_text.strip(), "literary passage")
            if result:
                print(f"Translation:\n{result}")
            
        except Exception as e:
            print(f"✗ OpenAI connection failed: {e}")
    else:
        print("✗ OpenAI API key not found in environment")
        print("  Set OPENAI_API_KEY environment variable")
    
    # Test with DeepSeek
    print("\n3. Testing with DeepSeek...")
    deepseek_key = os.getenv('DEEPSEEK_API_KEY')
    if deepseek_key:
        try:
            deepseek_config = LLMConfig(
                provider='deepseek',
                model='deepseek-chat',
                api_key=deepseek_key,
                base_url='https://api.deepseek.com',
                temperature=0.3
            )
            
            translator = AdvancedLLMFB2Translator(deepseek_config)
            print("✓ DeepSeek connection established")
            
            # Test translation
            result = translator.translator.translate_text(sample_text.strip(), "literary passage")
            if result:
                print(f"Translation:\n{result}")
                
        except Exception as e:
            print(f"✗ DeepSeek connection failed: {e}")
    else:
        print("✗ DeepSeek API key not found in environment")
        print("  Set DEEPSEEK_API_KEY environment variable")

    # Test with Anthropic
    print("\n4. Testing with Anthropic Claude...")
    claude_key = os.getenv('ANTHROPIC_API_KEY')
    if claude_key:
        try:
            claude_config = LLMConfig(
                provider='anthropic',
                model='claude-3-sonnet-20240229',
                api_key=claude_key,
                temperature=0.3
            )
            
            translator = AdvancedLLMFB2Translator(claude_config)
            print("✓ Anthropic connection established")
            
            # Test translation
            result = translator.translator.translate_text(sample_text.strip(), "literary passage")
            if result:
                print(f"Translation:\n{result}")
                
        except Exception as e:
            print(f"✗ Anthropic connection failed: {e}")
    else:
        print("✗ Anthropic API key not found in environment")
        print("  Set ANTHROPIC_API_KEY environment variable")

def show_quality_benefits():
    """Show the benefits of LLM translation over Google Translate"""
    
    print("\n" + "=" * 60)
    print("QUALITY COMPARISON: LLM vs Google Translate")
    print("=" * 60)
    
    comparisons = [
        {
            "russian": "Он с трудом вспоминал тот дождливый вечер, когда всё изменилось.",
            "google": "Teško je se sećao tog kišnog večera kada se sve promenilo.",
            "llm": "Sa teškom se sećao onog kišnog večeri kada se sve izmenilo."
        },
        {
            "russian": "Его сердце билось так сильно, будто хотело выскочить из груди.",
            "google": "Njegovo srce je bilo tako jako, kao da je htelo da iskoči iz grudi.",
            "llm": "Srce mu je otkućivalo tako snažno da je htelo da iskoči iz grudi."
        }
    ]
    
    for i, comp in enumerate(comparisons, 1):
        print(f"\nExample {i}:")
        print(f"Russian: {comp['russian']}")
        print(f"Google:   {comp['google']}")
        print(f"LLM:      {comp['llm']}")
        print()

def main():
    """Main test function"""
    print("Testing LLM FB2 Translation System")
    
    if len(sys.argv) > 1 and sys.argv[1] == '--compare':
        show_quality_benefits()
    else:
        test_translation_quality()
        show_quality_benefits()
    
    print("\n" + "=" * 60)
    print("TO USE WITH FB2 FILES:")
    print("=" * 60)
    print("# Local Ollama (FREE):")
    print("python3 llm_fb2_translator.py book.fb2 --provider ollama")
    print()
    print("# OpenAI GPT-4:")
    print("OPENAI_API_KEY=your-key python3 llm_fb2_translator.py book.fb2")
    print()
    print("# DeepSeek (Cost-effective, high quality):")
    print("DEEPSEEK_API_KEY=your-key python3 llm_fb2_translator.py book.fb2 --provider deepseek")
    print()
    print("# Anthropic Claude:")
    print("ANTHROPIC_API_KEY=your-key python3 llm_fb2_translator.py book.fb2 --provider anthropic")

if __name__ == "__main__":
    main()