import sys
import json
from indic_transliteration import sanscript
from indic_transliteration.sanscript import transliterate

def clean_romanized(text):
    if not text:
        return text
    # Check if there are any Devanagari characters (Unicode range 0900-097F)
    if not any(ord(char) >= 0x0900 and ord(char) <= 0x097F for char in text):
        return text

    res = transliterate(text, sanscript.DEVANAGARI, sanscript.IAST)
    
    replacements = {
        'ā': 'aa', 'ī': 'ee', 'ū': 'oo', 
        'ṛ': 'ri', 'ṝ': 'ri', 'ḷ': 'l',
        'ṃ': 'n', 'ḥ': 'h',
        'c': 'ch', 'ch': 'chh',
        'ś': 'sh', 'ṣ': 'sh',
        'ñ': 'n', 'ṅ': 'n',
        'ṭ': 't', 'ḍ': 'd', 'ṇ': 'n',
        '|': '.', '।': '.'
    }
    
    # Sort keys by length descending to replace multi-character targets first
    for key in sorted(replacements.keys(), key=len, reverse=True):
        res = res.replace(key, replacements[key])
        
    # Standard rule for ending 'a' in Hindi (schwa deletion at word boundaries)
    words = res.split()
    for idx, w in enumerate(words):
        clean_w = w.strip(".,?!:;()")
        if len(clean_w) > 2 and clean_w.endswith('a'):
            if clean_w not in ["kya", "namaste"]:
                pos = w.rfind('a')
                if pos != -1 and pos == len(w) - 1 - (len(w) - len(clean_w)):
                    words[idx] = w[:pos] + w[pos+1:]
    
    res = " ".join(words)
    # Simplify double character replacements
    res = res.replace("chchh", "chch")
    return res

def romanize_json(json_path):
    with open(json_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
        
    if 'text' in data:
        data['text'] = clean_romanized(data['text'])
        
    if 'segments' in data:
        for seg in data['segments']:
            if 'text' in seg:
                seg['text'] = clean_romanized(seg['text'])
                
    with open(json_path, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False, indent=2)

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: python romanize.py <path_to_json>")
        sys.exit(1)
    romanize_json(sys.argv[1])
