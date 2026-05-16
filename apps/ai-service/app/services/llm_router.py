"""LLM Router — abstracts Anthropic and OpenAI behind a single interface.

All AI features call this module. Provider-specific code lives here only.
Fallback chain: primary provider → secondary provider → raise.
"""
from __future__ import annotations

from enum import Enum
from typing import Any

from app.config import settings


class Provider(str, Enum):
    ANTHROPIC = "anthropic"
    OPENAI = "openai"


class LLMRouter:
    """Routes LLM requests to the configured provider with fallback."""

    def __init__(self) -> None:
        self._anthropic_client = None
        self._openai_client = None

    def _get_anthropic(self) -> Any:
        if self._anthropic_client is None:
            import anthropic
            self._anthropic_client = anthropic.Anthropic(api_key=settings.anthropic_api_key)
        return self._anthropic_client

    def _get_openai(self) -> Any:
        if self._openai_client is None:
            from openai import OpenAI
            self._openai_client = OpenAI(api_key=settings.openai_api_key)
        return self._openai_client

    async def complete(
        self,
        messages: list[dict],
        *,
        system: str = "",
        max_tokens: int = 4096,
        provider: Provider | None = None,
    ) -> tuple[str, str, int, int]:
        """
        Returns (response_text, model_used, tokens_in, tokens_out).
        Falls back from primary to secondary provider on failure.
        """
        primary = Provider(provider or settings.default_provider)
        providers = [primary]
        secondary = Provider.OPENAI if primary == Provider.ANTHROPIC else Provider.ANTHROPIC
        providers.append(secondary)

        last_error: Exception | None = None
        for p in providers:
            try:
                if p == Provider.ANTHROPIC:
                    return await self._call_anthropic(messages, system, max_tokens)
                else:
                    return await self._call_openai(messages, system, max_tokens)
            except Exception as exc:
                last_error = exc
                continue

        raise RuntimeError(f"All providers failed. Last error: {last_error}") from last_error

    async def _call_anthropic(
        self, messages: list[dict], system: str, max_tokens: int
    ) -> tuple[str, str, int, int]:
        import asyncio
        client = self._get_anthropic()
        model = settings.default_model

        response = await asyncio.to_thread(
            client.messages.create,
            model=model,
            max_tokens=max_tokens,
            system=system,
            messages=messages,
        )
        text = response.content[0].text
        return text, model, response.usage.input_tokens, response.usage.output_tokens

    async def _call_openai(
        self, messages: list[dict], system: str, max_tokens: int
    ) -> tuple[str, str, int, int]:
        import asyncio
        client = self._get_openai()
        model = "gpt-4o"

        all_messages = []
        if system:
            all_messages.append({"role": "system", "content": system})
        all_messages.extend(messages)

        response = await asyncio.to_thread(
            client.chat.completions.create,
            model=model,
            max_tokens=max_tokens,
            messages=all_messages,
        )
        text = response.choices[0].message.content or ""
        usage = response.usage
        return text, model, usage.prompt_tokens, usage.completion_tokens

    async def embed(self, text: str) -> list[float]:
        """Generate embedding vector using OpenAI text-embedding-3-small."""
        import asyncio
        client = self._get_openai()
        response = await asyncio.to_thread(
            client.embeddings.create,
            model=settings.embedding_model,
            input=text,
        )
        return response.data[0].embedding


llm_router = LLMRouter()
