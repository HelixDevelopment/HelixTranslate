#!/usr/bin/env python3
"""
Advanced LLM-powered FB2 translator for high-quality Russian to Serbian translations.
Supports OpenAI GPT, Anthropic Claude, and other LLM providers.
"""

import xml.etree.ElementTree as ET
import re
import sys
import time
import json
import os
from pathlib import Path
from typing import Dict, List, Tuple, Optional, Any
from dataclasses import dataclass

@dataclass
class LLMConfig:
    """Configuration for LLM providers"""
    provider: str  # 'openai', 'anthropic', 'ollama', 'custom'
    model: str
    api_key: Optional[str] = None
    base_url: Optional[str] = None
    temperature: float = 0.3
    max_tokens: Optional[int] = None

class LLMTranslator:
    """Base class for LLM translation providers"""
    
    def __init__(self, config: LLMConfig):
        self.config = config
        self.translation_cache = {}
        self.translation_stats = {
            'total': 0,
            'translated': 0,
            'errors': 0,
            'cached': 0
        }
        
        # Serbian Cyrillic to Latin mapping
        self.cyrl_to_latn = {
            'А': 'A', 'Б': 'B', 'В': 'V', 'Г': 'G', 'Д': 'D', 'Ђ': 'Đ', 'Е': 'E', 'Ж': 'Ž', 'З': 'Z',
            'И': 'I', 'Ј': 'J', 'К': 'K', 'Л': 'L', 'Љ': 'Lj', 'М': 'M', 'Н': 'N', 'Њ': 'Nj', 'О': 'O',
            'П': 'P', 'Р': 'R', 'С': 'S', 'Т': 'T', 'Ћ': 'Ć', 'У': 'U', 'Ф': 'F', 'Х': 'H', 'Ц': 'C',
            'Ч': 'Č', 'Џ': 'Dž', 'Ш': 'Š', 'а': 'a', 'б': 'b', 'в': 'v', 'г': 'g', 'д': 'd', 'ђ': 'đ',
            'е': 'e', 'ж': 'ž', 'з': 'z', 'и': 'i', 'ј': 'j', 'к': 'k', 'л': 'l', 'љ': 'lj', 'м': 'm',
            'н': 'n', 'њ': 'nj', 'о': 'o', 'п': 'p', 'р': 'r', 'с': 's', 'т': 't', 'ћ': 'ć', 'у': 'u',
            'ф': 'f', 'х': 'h', 'ц': 'c', 'ч': 'č', 'џ': 'dž', 'ш': 'ш'
        }
    
    def translate_text(self, text: str, context: str = "") -> Optional[str]:
        """Translate text using the configured LLM provider"""
        raise NotImplementedError("Subclasses must implement translate_text")
    
    def create_translation_prompt(self, text: str, context: str = "") -> str:
        """Create a high-quality translation prompt for the LLM"""
        prompt = f"""You are a professional literary translator specializing in Russian to Serbian translation. 
Your task is to translate the following Russian text into natural, idiomatic Serbian.

Guidelines:
1. Preserve the literary style and tone
2. Use appropriate Serbian vocabulary and grammar
3. Maintain cultural nuances and idioms
4. Keep names of people and places unchanged unless they have standard Serbian equivalents
5. Preserve formatting, punctuation, and paragraph structure
6. Use Serbian Cyrillic script (ћирилица)

Context: {context if context else "Literary text"}

Russian text:
{text}

Serbian translation:"""
        return prompt
    
    def enhance_translation(self, original: str, translated: str) -> str:
        """Post-process translation for quality"""
        # Fix common punctuation issues
        enhanced = translated
        
        # Ensure proper quotation marks
        enhanced = enhanced.replace('"', '"').replace('"', '"')
        
        # Fix apostrophes
        enhanced = enhanced.replace("'", "'")
        
        # Preserve paragraph structure
        if original.endswith('\n') and not enhanced.endswith('\n'):
            enhanced += '\n'
        
        # Fix sentence capitalization
        if enhanced and enhanced[0].islower() and original and original[0].isupper():
            enhanced = enhanced[0].upper() + enhanced[1:]
        
        return enhanced
    
    def convert_to_latin(self, text: str) -> str:
        """Convert Cyrillic Serbian to Latin script"""
        result = ""
        for char in text:
            result += self.cyrl_to_latn.get(char, char)
        return result

class OpenAITranslator(LLMTranslator):
    """OpenAI GPT-based translator"""
    
    def __init__(self, config: LLMConfig):
        super().__init__(config)
        try:
            import openai
            self.client = openai.OpenAI(api_key=config.api_key, base_url=config.base_url)
            print(f"✓ Connected to OpenAI (model: {config.model})")
        except ImportError:
            raise ImportError("OpenAI library not installed. Install with: pip install openai")
        except Exception as e:
            raise Exception(f"Failed to initialize OpenAI client: {e}")
    
    def translate_text(self, text: str, context: str = "") -> Optional[str]:
        """Translate using OpenAI GPT"""
        if not text or not text.strip():
            return text
        
        cache_key = (text, context)
        if cache_key in self.translation_cache:
            self.translation_stats['cached'] += 1
            return self.translation_cache[cache_key]
        
        try:
            prompt = self.create_translation_prompt(text, context)
            
            response = self.client.chat.completions.create(
                model=self.config.model,
                messages=[
                    {"role": "system", "content": "You are a professional Russian to Serbian literary translator."},
                    {"role": "user", "content": prompt}
                ],
                temperature=self.config.temperature,
                max_tokens=self.config.max_tokens
            )
            
            translation = response.choices[0].message.content.strip()
            enhanced = self.enhance_translation(text, translation)
            
            self.translation_cache[cache_key] = enhanced
            self.translation_stats['translated'] += 1
            
            return enhanced
            
        except Exception as e:
            self.translation_stats['errors'] += 1
            print(f"OpenAI translation error: {e}")
            return None

class AnthropicTranslator(LLMTranslator):
    """Anthropic Claude-based translator"""
    
    def __init__(self, config: LLMConfig):
        super().__init__(config)
        try:
            import anthropic
            self.client = anthropic.Anthropic(api_key=config.api_key)
            print(f"✓ Connected to Anthropic Claude (model: {config.model})")
        except ImportError:
            raise ImportError("Anthropic library not installed. Install with: pip install anthropic")
        except Exception as e:
            raise Exception(f"Failed to initialize Anthropic client: {e}")
    
    def translate_text(self, text: str, context: str = "") -> Optional[str]:
        """Translate using Anthropic Claude"""
        if not text or not text.strip():
            return text
        
        cache_key = (text, context)
        if cache_key in self.translation_cache:
            self.translation_stats['cached'] += 1
            return self.translation_cache[cache_key]
        
        try:
            prompt = self.create_translation_prompt(text, context)
            
            response = self.client.messages.create(
                model=self.config.model,
                max_tokens=self.config.max_tokens or 4000,
                temperature=self.config.temperature,
                messages=[
                    {"role": "user", "content": prompt}
                ]
            )
            
            translation = response.content[0].text.strip()
            enhanced = self.enhance_translation(text, translation)
            
            self.translation_cache[cache_key] = enhanced
            self.translation_stats['translated'] += 1
            
            return enhanced
            
        except Exception as e:
            self.translation_stats['errors'] += 1
            print(f"Anthropic translation error: {e}")
            return None

class DeepSeekTranslator(LLMTranslator):
    """DeepSeek-based translator"""
    
    def __init__(self, config: LLMConfig):
        super().__init__(config)
        try:
            import openai
            # DeepSeek uses OpenAI-compatible API
            self.client = openai.OpenAI(
                api_key=config.api_key,
                base_url=config.base_url or "https://api.deepseek.com/v1"
            )
            print(f"✓ Connected to DeepSeek (model: {config.model})")
        except ImportError:
            raise ImportError("OpenAI library not installed. Install with: pip install openai")
        except Exception as e:
            raise Exception(f"Failed to initialize DeepSeek client: {e}")
    
    def translate_text(self, text: str, context: str = "") -> Optional[str]:
        """Translate using DeepSeek"""
        if not text or not text.strip():
            return text
        
        cache_key = (text, context)
        if cache_key in self.translation_cache:
            self.translation_stats['cached'] += 1
            return self.translation_cache[cache_key]
        
        try:
            prompt = self.create_translation_prompt(text, context)
            
            response = self.client.chat.completions.create(
                model=self.config.model,
                messages=[
                    {"role": "system", "content": "You are a professional Russian to Serbian literary translator known for preserving cultural nuances and literary style."},
                    {"role": "user", "content": prompt}
                ],
                temperature=self.config.temperature,
                max_tokens=self.config.max_tokens
            )
            
            translation = response.choices[0].message.content.strip()
            enhanced = self.enhance_translation(text, translation)
            
            self.translation_cache[cache_key] = enhanced
            self.translation_stats['translated'] += 1
            
            return enhanced
            
        except Exception as e:
            self.translation_stats['errors'] += 1
            print(f"DeepSeek translation error: {e}")
            return None

class ZhipuTranslator(LLMTranslator):
    """Zhipu AI-based translator"""
    
    def __init__(self, config: LLMConfig):
        super().__init__(config)
        try:
            import openai
            # Zhipu AI uses OpenAI-compatible API with proper authentication
            self.client = openai.OpenAI(
                api_key=config.api_key,
                base_url=config.base_url or "https://open.bigmodel.cn/api/paas/v4",
                default_headers={
                    "Content-Type": "application/json",
                    "Authorization": f"Bearer {config.api_key}"
                }
            )
            
            # Test API connection and get available models
            try:
                response = self.client.models.list()
                available_models = [model.id for model in response.data]
                print(f"✓ Available Zhipu AI models: {', '.join(available_models[:3])}...")
                
                if config.model not in available_models:
                    # Try alternative model names
                    alternative_models = [
                        "glm-4",
                        "glm-4v", 
                        "glm-4-0520",
                        "glm-4-0206",
                        "glm-4-air",
                        "glm-4-airx",
                        "glm-4-flash",
                        "chatglm3",
                        "chatglm_pro"
                    ]
                    
                    for alt_model in alternative_models:
                        if alt_model in available_models:
                            config.model = alt_model
                            print(f"✓ Using alternative model: {alt_model}")
                            break
                    else:
                        print(f"⚠ Model {config.model} not found. Available: {', '.join(available_models)}")
                        # Use first available model
                        if available_models:
                            config.model = available_models[0]
                            print(f"✓ Auto-selecting model: {config.model}")
                        
            except Exception as e:
                print(f"⚠ Could not list models: {e}")
                
            print(f"✓ Connected to Zhipu AI (model: {config.model})")
        except ImportError:
            raise ImportError("OpenAI library not installed. Install with: pip install openai")
        except Exception as e:
            raise Exception(f"Failed to initialize Zhipu AI client: {e}")
    
    def translate_text(self, text: str, context: str = "") -> Optional[str]:
        """Translate using Zhipu AI"""
        if not text or not text.strip():
            return text
        
        cache_key = (text, context)
        if cache_key in self.translation_cache:
            self.translation_stats['cached'] += 1
            return self.translation_cache[cache_key]
        
        try:
            prompt = self.create_translation_prompt(text, context)
            
            response = self.client.chat.completions.create(
                model=self.config.model,
                messages=[
                    {"role": "system", "content": "You are a professional Russian to Serbian literary translator with deep understanding of cultural nuances and literary style."},
                    {"role": "user", "content": prompt}
                ],
                temperature=self.config.temperature,
                max_tokens=self.config.max_tokens,
                extra_headers={
                    "Content-Type": "application/json",
                    "Authorization": f"Bearer {self.config.api_key}"
                }
            )
            
            translation = response.choices[0].message.content.strip()
            enhanced = self.enhance_translation(text, translation)
            
            self.translation_cache[cache_key] = enhanced
            self.translation_stats['translated'] += 1
            
            return enhanced
            
        except Exception as e:
            self.translation_stats['errors'] += 1
            print(f"Zhipu AI translation error: {e}")
            return None

class OllamaTranslator(LLMTranslator):
    """Local Ollama-based translator"""
    
    def __init__(self, config: LLMConfig):
        super().__init__(config)
        try:
            import requests
            self.requests = requests
            self.base_url = config.base_url or "http://localhost:11434"
            
            # Check if Ollama is running and model is available
            response = self.requests.get(f"{self.base_url}/api/tags")
            if response.status_code != 200:
                raise Exception("Ollama is not running or not accessible")
            
            models = response.json().get('models', [])
            model_names = [m['name'] for m in models]
            if not any(config.model in name for name in model_names):
                print(f"⚠ Model {config.model} not found. Available models: {', '.join(model_names)}")
                print(f"Pull with: ollama pull {config.model}")
            
            print(f"✓ Connected to Ollama (model: {config.model})")
            
        except ImportError:
            raise ImportError("Requests library not installed. Install with: pip install requests")
        except Exception as e:
            raise Exception(f"Failed to initialize Ollama client: {e}")
    
    def translate_text(self, text: str, context: str = "") -> Optional[str]:
        """Translate using local Ollama model"""
        if not text or not text.strip():
            return text
        
        cache_key = (text, context)
        if cache_key in self.translation_cache:
            self.translation_stats['cached'] += 1
            return self.translation_cache[cache_key]
        
        try:
            prompt = self.create_translation_prompt(text, context)
            
            response = self.requests.post(
                f"{self.base_url}/api/generate",
                json={
                    "model": self.config.model,
                    "prompt": prompt,
                    "stream": False,
                    "options": {
                        "temperature": self.config.temperature
                    }
                }
            )
            
            if response.status_code != 200:
                raise Exception(f"Ollama API error: {response.status_code}")
            
            translation = response.json().get('response', '').strip()
            enhanced = self.enhance_translation(text, translation)
            
            self.translation_cache[cache_key] = enhanced
            self.translation_stats['translated'] += 1
            
            return enhanced
            
        except Exception as e:
            self.translation_stats['errors'] += 1
            print(f"Ollama translation error: {e}")
            return None

class AdvancedLLMFB2Translator:
    """Main FB2 translator using LLM providers"""
    
    def __init__(self, config: LLMConfig, use_latin: bool = False):
        self.config = config
        self.use_latin = use_latin
        
        # Initialize the appropriate translator
        if config.provider == 'openai':
            self.translator = OpenAITranslator(config)
        elif config.provider == 'anthropic':
            self.translator = AnthropicTranslator(config)
        elif config.provider == 'deepseek':
            self.translator = DeepSeekTranslator(config)
        elif config.provider == 'zhipu':
            self.translator = ZhipuTranslator(config)
        elif config.provider == 'ollama':
            self.translator = OllamaTranslator(config)
        else:
            raise ValueError(f"Unsupported provider: {config.provider}")
        
        print(f"Advanced LLM translator ready: {config.provider} → {config.model}")
        if use_latin:
            print("Output will be in Serbian Latin script")
    
    def process_fb2_structure(self, input_path: str, output_path: str) -> bool:
        """Process FB2 file with high-quality LLM translation"""
        try:
            # Register namespaces
            ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
            ET.register_namespace('l', "http://www.w3.org/1999/xlink")
            
            # Parse the file
            tree = ET.parse(input_path)
            root = tree.getroot()
            
            print(f"Processing FB2 with LLM translation...")
            
            # Update document metadata
            self.update_document_metadata(root)
            
            # Process all text elements
            self.process_element_translations(root)
            
            # Write the enhanced translation
            print(f"\nWriting LLM translation to: {output_path}")
            self.write_enhanced_xml(tree, output_path)
            
            # Show statistics
            self.translator.print_translation_stats()
            
            return True
            
        except Exception as e:
            print(f"Error processing FB2: {e}")
            import traceback
            traceback.print_exc()
            return False
    
    def update_document_metadata(self, root: ET.Element):
        """Update document metadata for Serbian translation"""
        # Update language
        description = root.find('.//{http://www.gribuser.ru/xml/fictionbook/2.0}description')
        if description is not None:
            title_info = description.find('{http://www.gribuser.ru/xml/fictionbook/2.0}title-info')
            if title_info is not None:
                lang = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}lang')
                if lang is not None:
                    lang.text = 'sr'
                
                # Translate title
                book_title = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}book-title')
                if book_title is not None and book_title.text:
                    translated_title = self.translator.translate_text(book_title.text, "book-title")
                    if translated_title:
                        book_title.text = translated_title
    
    def process_element_translations(self, element: ET.Element, context: str = ""):
        """Recursively process and translate all elements"""
        # Process element text
        if element.text and element.text.strip():
            text = element.text.strip()
            if len(text) > 2:
                self.translator.translation_stats['total'] += 1
                
                translation = self.translator.translate_text(text, context)
                if translation:
                    element.text = translation
                    print(".", end="", flush=True)
                else:
                    print("o", end="", flush=True)
        
        # Process child elements
        for child in element:
            child_context = f"{context}/{child.tag}" if context else child.tag
            self.process_element_translations(child, child_context)
        
        # Process tail text
        if element.tail and element.tail.strip():
            text = element.tail.strip()
            if len(text) > 2:
                self.translator.translation_stats['total'] += 1
                
                translation = self.translator.translate_text(text, f"{context}/tail")
                if translation:
                    element.tail = translation
                    print(".", end="", flush=True)
                else:
                    print("o", end="", flush=True)
    
    def write_enhanced_xml(self, tree: ET.ElementTree, output_path: str):
        """Write enhanced XML with proper formatting"""
        try:
            # Convert to Latin if requested
            if self.use_latin:
                self.convert_tree_to_latin(tree.getroot())
            
            # Get XML string
            root = tree.getroot()
            if root is None:
                raise ValueError("Root element is None")
            xml_string = ET.tostring(root, encoding='unicode')
            
            # Parse and format with minidom
            from xml.dom import minidom
            dom = minidom.parseString(xml_string)
            
            # Write with proper formatting
            with open(output_path, 'w', encoding='utf-8') as f:
                f.write(dom.toprettyxml(indent=" ", encoding=None))
            
            print(f"\n✓ LLM translation saved: {output_path}")
            
        except Exception as e:
            print(f"Enhanced writing failed: {e}")
            # Fallback to standard XML writing
            try:
                tree.write(output_path, encoding='utf-8', xml_declaration=True)
                print(f"\n✓ Translation saved: {output_path}")
            except Exception as e2:
                print(f"Standard writing also failed: {e2}")
                raise e2
    
    def convert_tree_to_latin(self, element: ET.Element):
        """Convert all text in tree from Cyrillic to Latin"""
        if element.text:
            element.text = self.translator.convert_to_latin(element.text)
        if element.tail:
            element.tail = self.translator.convert_to_latin(element.tail)
        
        for child in element:
            self.convert_tree_to_latin(child)

def load_config_from_file(config_path: str) -> LLMConfig:
    """Load configuration from JSON file"""
    try:
        with open(config_path, 'r', encoding='utf-8') as f:
            data = json.load(f)
        return LLMConfig(**data)
    except Exception as e:
        print(f"Failed to load config from {config_path}: {e}")
        return None

def save_default_config(config_path: str):
    """Save default configuration template"""
    default_config = {
        "provider": "openai",
        "model": "gpt-4",
        "api_key": "your-api-key-here",
        "base_url": None,
        "temperature": 0.3,
        "max_tokens": 4000
    }
    
    with open(config_path, 'w', encoding='utf-8') as f:
        json.dump(default_config, f, indent=2)
    
    print(f"Default config saved to {config_path}")
    print("Edit this file with your API keys and preferences")

def create_config_from_env() -> Optional[LLMConfig]:
    """Create config from environment variables"""
    provider = os.getenv('LLM_PROVIDER', 'openai')
    api_key = (os.getenv('LLM_API_KEY') or 
                os.getenv('OPENAI_API_KEY') or 
                os.getenv('ANTHROPIC_API_KEY') or
                os.getenv('DEEPSEEK_API_KEY') or
                os.getenv('ZHIPU_API_KEY'))
    base_url = os.getenv('LLM_BASE_URL')
    model = os.getenv('LLM_MODEL')
    
    if not api_key and provider not in ['ollama']:
        return None
    
    # Default models
    if not model:
        if provider == 'openai':
            model = 'gpt-4'
        elif provider == 'anthropic':
            model = 'claude-3-sonnet-20240229'
        elif provider == 'deepseek':
            model = 'deepseek-chat'
        elif provider == 'zhipu':
            model = 'glm-4'
        elif provider == 'ollama':
            model = 'llama3:8b'
    
    return LLMConfig(
        provider=provider,
        model=model,
        api_key=api_key,
        base_url=base_url,
        temperature=float(os.getenv('LLM_TEMPERATURE', '0.3')),
        max_tokens=int(os.getenv('LLM_MAX_TOKENS', '4000')) if os.getenv('LLM_MAX_TOKENS') else None
    )

def main():
    """Main translation function"""
    if len(sys.argv) < 2:
        print("Usage: python3 llm_fb2_translator.py <input_file.fb2> [output_file.b2] [options]")
        print("\nOptions:")
        print("  --provider openai|anthropic|deepseek|zhipu|ollama    # LLM provider")
        print("  --model MODEL_NAME                      # Model name")
        print("  --config CONFIG_FILE                    # JSON config file")
        print("  --latin                                 # Output in Latin script")
        print("  --create-config CONFIG_FILE            # Create default config")
        print("\nExamples:")
        print("  # Using OpenAI with environment variables")
        print("  LLM_PROVIDER=openai LLM_API_KEY=your-key python3 llm_fb2_translator.py book.fb2")
        print("  # Using Anthropic Claude")
        print("  ANTHROPIC_API_KEY=your-key python3 llm_fb2_translator.py book.fb2 --provider anthropic")
        print("  # Using DeepSeek (cost-effective)")
        print("  DEEPSEEK_API_KEY=your-key python3 llm_fb2_translator.py book.fb2 --provider deepseek")
        print("  # Using Zhipu AI (cutting edge)")
        print("  ZHIPU_API_KEY=your-key python3 llm_fb2_translator.py book.fb2 --provider zhipu")
        print("  # Using config file")
        print("  python3 llm_fb2_translator.py book.fb2 --config config.json")
        print("  # Using local Ollama")
        print("  python3 llm_fb2_translator.py book.fb2 --provider ollama --model llama3:8b")
        print("  # Create config template")
        print("  python3 llm_fb2_translator.py --create-config config.json")
        return False
    
    # Handle config creation
    if '--create-config' in sys.argv:
        config_idx = sys.argv.index('--create-config')
        if config_idx + 1 < len(sys.argv):
            config_path = sys.argv[config_idx + 1]
            save_default_config(config_path)
            return True
        else:
            print("Error: --create-config requires a file path")
            return False
    
    # Parse command line arguments
    input_file = sys.argv[1]
    
    # If this is a help/config request and no input file exists, don't process
    if not Path(input_file).exists() and not any(arg.startswith('--') for arg in sys.argv[2:]):
        print(f"Error: Input file {input_file} not found")
        print("Use --help for usage information or --create-config to create config")
        return False
    
    # Generate output filename if not provided
    output_file = None
    use_latin = '--latin' in sys.argv
    
    # Find output file (first non-option argument after input)
    for i, arg in enumerate(sys.argv[2:], 2):
        if not arg.startswith('--'):
            output_file = arg
            break
    
    if not output_file:
        input_path = Path(input_file)
        stem = input_path.stem
        suffix = "_sr_latin.b2" if use_latin else "_sr_llm.b2"
        output_file = f"{stem}{suffix}"
    
    # Load configuration
    config = None
    
    # Try config file first
    if '--config' in sys.argv:
        config_idx = sys.argv.index('--config')
        if config_idx + 1 < len(sys.argv):
            config_path = sys.argv[config_idx + 1]
            config = load_config_from_file(config_path)
    
    # Try command line options
    if not config:
        provider = 'openai'
        model = 'gpt-4'
        
        if '--provider' in sys.argv:
            provider_idx = sys.argv.index('--provider')
            if provider_idx + 1 < len(sys.argv):
                provider = sys.argv[provider_idx + 1]
        
        if '--model' in sys.argv:
            model_idx = sys.argv.index('--model')
            if model_idx + 1 < len(sys.argv):
                model = sys.argv[model_idx + 1]
        
        # Get API key from environment or prompt (only if we have a valid input file)
        api_key = (os.getenv('LLM_API_KEY') or 
                    os.getenv('OPENAI_API_KEY') or 
                    os.getenv('ANTHROPIC_API_KEY') or
                    os.getenv('DEEPSEEK_API_KEY') or
                    os.getenv('ZHIPU_API_KEY'))
        
        if provider != 'ollama' and not api_key and Path(input_file).exists():
            api_key = input(f"Enter {provider} API key: ").strip()
        
        config = LLMConfig(
            provider=provider,
            model=model,
            api_key=api_key,
            temperature=0.3,
            max_tokens=4000
        )
    
    # Try environment variables as last resort
    if not config:
        config = create_config_from_env()
    
    if not config:
        print("Error: No LLM configuration provided")
        print("Set environment variables (LLM_PROVIDER, LLM_API_KEY) or use --config")
        return False
    
    if not Path(input_file).exists() and not any(arg.startswith('--create-config') for arg in sys.argv):
        print(f"Error: Input file {input_file} not found")
        return False
    
    # Create translator and run
    translator = AdvancedLLMFB2Translator(config, use_latin)
    
    script_type = "Latin" if use_latin else "Cyrillic"
    print(f"\nStarting ADVANCED LLM Russian to Serbian FB2 Translation")
    print("="*60)
    print(f"Provider: {config.provider} ({config.model})")
    print(f"Output script: {script_type}")
    print("="*60)
    
    success = translator.process_fb2_structure(input_file, output_file)
    
    if success:
        print("\n" + "="*60)
        print("✓ LLM TRANSLATION COMPLETED SUCCESSFULLY!")
        print("="*60)
        print(f"✓ Output file: {output_file}")
        print("\nQuality benefits of LLM translation:")
        print("✓ Literary style preservation")
        print("✓ Cultural nuance handling")
        print("✓ Context-aware translations")
        print("✓ Natural Serbian phrasing")
        print("✓ Consistent character voice")
    else:
        print("\n✗ LLM translation encountered issues")
        print("Please check the error messages above")
    
    return success

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)