---
name: translate_text
description: Translate text from one language to another.
parameters:
  type: object
  properties:
    text:
      type: string
      description: "The text to translate."
    source_lang:
      type: string
      description: "Source language code (e.g., 'en', 'zh'). Optional, defaults to auto."
    target_lang:
      type: string
      description: "Target language code (e.g., 'zh', 'en')."
  required:
    - text
    - target_lang
---

# Translation Skill
- Use this tool to translate text.
- Do NOT try to construct curl commands yourself. The system handles the API call.
- Just provide the 'text' and 'target_lang'.

