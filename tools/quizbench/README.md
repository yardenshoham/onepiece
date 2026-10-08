# quizbench

Runs the production quiz generator (`pkg/quiz`) with other OpenRouter models on episodes 42–44, printing latency and the generated questions.

```sh
ONEPIECE_OPENROUTER_API_KEY=... go run ./tools/quizbench -runs 3 ~openai/gpt-luna-latest google/gemini-3.5-flash
```

Latency includes the generator's retries (up to 3). Before trying a model, check it supports `structured_outputs` and `reasoning` with the quiz schema (`minItems: 3`) and `max_tokens: 1500`.

## History

- **2026-07**: switched from `google/gemini-2.5-flash` to `openai/gpt-5.6-luna` (3.6–5.1s, about $0.005 per quiz).
- **2026-10**: switched to the auto-updating alias `~openai/gpt-luna-latest` (then GPT-6 Luna) at low effort. At medium effort, GPT-6 Luna took 10–17s and used up to about 1250 of 1500 tokens. At low effort it took 3–10s at about $0.0003 per quiz. If latency or truncation regresses, suspect a new Luna release first.

Models that lost:

| Model | Why |
| --- | --- |
| `anthropic/claude-haiku-4.5` | Rejects the schema (Anthropic only allows `minItems` 0/1) |
| `google/gemini-3-flash-preview`, `gemini-3.5-flash` | Reasoning eats `max_tokens` and truncates; 14–25s even at 4000 |
| `x-ai/grok-4.3` | 35–52s |
| `z-ai/glm-5.2` | Erratic: markdown fences, empty fields, 9–56s |
| `openai/gpt-5.6-luna-pro` | 2x slower and about 5x the cost of GPT-5.6 Luna |
