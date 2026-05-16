"""Pytest configuration and shared fixtures for OpsOwl AI service tests.

Rule: All LLM calls are mocked at the provider boundary (llm_router._call_anthropic / _call_openai).
Never make real API calls in tests.
"""
import pytest
from fastapi.testclient import TestClient
from unittest.mock import AsyncMock, patch

from app.main import app


@pytest.fixture
def client() -> TestClient:
    return TestClient(app)


@pytest.fixture
def mock_llm_router():
    """Mock the LLM router to return predictable responses without hitting any API."""
    with patch("app.services.llm_router.llm_router.complete", new_callable=AsyncMock) as mock:
        mock.return_value = (
            "Mocked AI response for testing.",
            "claude-opus-4-5",
            100,
            50,
        )
        yield mock


@pytest.fixture
def mock_embed():
    """Mock embedding generation — returns a deterministic 1536-dim vector."""
    with patch("app.services.llm_router.llm_router.embed", new_callable=AsyncMock) as mock:
        mock.return_value = [0.1] * 1536
        yield mock
