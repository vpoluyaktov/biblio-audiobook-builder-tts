# ABB-TTS: Critical Development Reference

## Core Architecture
- **Pipeline Flow**: Web UI/API → Job Queue → TTS Providers → Audio Build → Outputs
- **Key Packages**: `tts/`, `ssml/`, `audio/`, `sanitize/`, `normalize/`
- **Deployment**: Docker-ready with Go module structure

## Non-Negotiables
1. **Text Processing Order** (MUST follow):
   ```mermaid
   flowchart LR
       A[Pronunciation Dict] --> B[Latin-RU Transliteration] --> C[Number Normalization] --> D[Stress Marking]
   ```
2. **Pattern Matching Logic**:
   - All CSV patterns auto-wrapped with `(?:^|\s)(PATTERN)(?:[\s.,)]|$)`
   - Always adds space after replacement
   - **Critical**: Compound units BEFORE simple (e.g. `км/ч` → `км`)
3. **File Naming**:
   - Chapters: `%04d_<name>.extension`
   - Multi-parts: `Book_Title, Part %03d.m4b`

## TTS Provider Selection (CPU Production)
| Model | Quality | CPU Speed | SSML | Recommendation |
|-------|---------|-----------|------|----------------|
| **Piper** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | **PRIMARY** (best balance) |
| Coqui TTS | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐ | High quality only |
| Mimic 3 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | Docker deployment |
| **eSpeak-NG** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | **Fallback/Testing** |

## Critical Russian Rules
- Dates: `27 марта` → genitive (`двадцать седьмого марта`)
- Measure units: `100 м` → plural genitive (`100 метров`)
- Years: `1876 г.` → prepositional (`восемьсот семьдесят шестом году`)
- Weight units MUST come AFTER `г.р.` in CSV

## Must-Fix Technical Debt
1. **Error Handling**:
   - Implement `%w` wrapping
   - Structured API error responses
2. **Concurrent Processing**:
   - Job state race conditions
   - Missing context cancellation
3. **Text Pipeline**:
   - Fix double spaces after punctuation
   - Add position-aware normalization

## Top Priority Refactors
- Extract provider-specific TTS logic to base implementation
- Implement parallel chapter processing
- Add Prometheus metrics for:
  - TTS latency
  - Queue depth
  - Normalization errors
- Structured logging with correlation IDs

> *Derived from Specification.md @2026-02-18 - Contains ONLY implementation-critical details*