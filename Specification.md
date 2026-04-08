# Specification

## fix/roman-numeral-transliteration-order

### Problem

Roman numerals (I, II, III, IV, etc.) in Russian books were not being converted to Russian words (первая, вторая, третья, четвёртый, etc.) even though the normalization logic for this existed in the `normalize` package.

### Root Cause

The worker pipeline in `internal/server/worker.go` applied Latin-to-Russian transliteration **before** number normalization. The transliterator matches all uppercase Latin letter sequences and converts them letter-by-letter (e.g., `"III"` → `"ай ай ай"`). By the time `ProcessRomanNumerals` ran in the normalization step, the Latin characters were already destroyed.

### Fix

Swapped the order of steps 2 and 3 in the worker pipeline:

- **Before**: Sanitize → Transliterate → Normalize → Stress
- **After**: Sanitize → Normalize → Transliterate → Stress

This ensures Roman numerals are processed while they are still Latin characters. After normalization converts them to Cyrillic words (e.g., `"Глава III"` → `"Глава третья"`), the transliterator handles only remaining genuine Latin text (abbreviations, names).

### Files Changed

- `internal/server/worker.go` — Reordered steps 2 and 3
- `internal/normalize/roman_test.go` — Added `TestProcessRomanNumeralsRussian` regression tests

### Status: ✅ Complete
