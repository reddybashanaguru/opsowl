"""LLM Router — abstracts Anthropic, OpenAI, and Ollama behind a single interface.

All AI features call this module. Provider-specific code lives here only.

Provider selection:
  - Set DEFAULT_PROVIDER=ollama for fully air-gapped on-premise deployments.
    All LLM inference and embeddings run locally via Ollama — zero data leaves
    the client's network. Required for RBI-regulated on-premise installations.
  - Set DEFAULT_PROVIDER=anthropic or openai for cloud-assisted deployments
    (requires explicit data processing agreements and client approval).

Fallback chain for cloud providers: primary → secondary → raise.
Ollama has no fallback — if it's down the request fails with a clear error.
"""
from __future__ import annotations

import asyncio
import time
from enum import Enum
from typing import Any

import httpx

from app.config import settings


class Provider(str, Enum):
    ANTHROPIC = "anthropic"
    OPENAI = "openai"
    OLLAMA = "ollama"


class LLMRouter:
    """Routes LLM requests to the configured provider."""

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
        Ollama runs locally — no data leaves the network.
        Cloud providers fall back from primary to secondary on failure.
        """
        primary = Provider(provider or settings.default_provider)

        if primary == Provider.OLLAMA:
            return await self._call_ollama(messages, system, max_tokens)

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

    async def _call_ollama(
        self, messages: list[dict], system: str, max_tokens: int
    ) -> tuple[str, str, int, int]:
        """Call a locally running Ollama instance. Zero data leaves the network."""
        model = settings.ollama_model
        all_messages = []
        if system:
            all_messages.append({"role": "system", "content": system})
        all_messages.extend(messages)

        payload = {
            "model": model,
            "messages": all_messages,
            "stream": False,
            "options": {"num_predict": max_tokens},
        }

        async with httpx.AsyncClient(base_url=settings.ollama_base_url, timeout=120.0) as client:
            resp = await client.post("/api/chat", json=payload)
            resp.raise_for_status()
            data = resp.json()

        text = data["message"]["content"]
        tokens_in = data.get("prompt_eval_count", 0)
        tokens_out = data.get("eval_count", 0)
        return text, model, tokens_in, tokens_out

    async def embed(self, text: str) -> list[float]:
        """Generate embedding vector. Uses Ollama locally or OpenAI for cloud."""
        if settings.default_provider == Provider.OLLAMA:
            return await self._embed_ollama(text)
        return await self._embed_openai(text)

    async def _embed_openai(self, text: str) -> list[float]:
        client = self._get_openai()
        response = await asyncio.to_thread(
            client.embeddings.create,
            model=settings.embedding_model,
            input=text,
        )
        return response.data[0].embedding

    async def _embed_ollama(self, text: str) -> list[float]:
        """Embed text via Ollama nomic-embed-text (768-dim, pgvector compatible)."""
        payload = {"model": settings.ollama_embedding_model, "prompt": text}
        async with httpx.AsyncClient(base_url=settings.ollama_base_url, timeout=30.0) as client:
            resp = await client.post("/api/embeddings", json=payload)
            resp.raise_for_status()
            return resp.json()["embedding"]


llm_router = LLMRouter()
