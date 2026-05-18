from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    environment: str = "development"
    database_url: str = ""
    go_api_url: str = "http://localhost:8080"

    # ── Cloud LLM credentials (used only when provider != ollama) ─────────────
    anthropic_api_key: str = ""
    openai_api_key: str = ""

    # ── LLM provider config ───────────────────────────────────────────────────
    # Set default_provider=ollama for fully air-gapped on-premise deployments.
    # When ollama is the provider, no data leaves the client's network.
    default_provider: str = "anthropic"
    default_model: str = "claude-opus-4-5"
    embedding_model: str = "text-embedding-3-small"

    # ── Ollama on-premise config ──────────────────────────────────────────────
    # Used when default_provider=ollama. Point to the local Ollama instance.
    ollama_base_url: str = "http://ollama:11434"
    ollama_model: str = "llama3.1:8b"
    ollama_embedding_model: str = "nomic-embed-text"

    # ── RAG settings ──────────────────────────────────────────────────────────
    max_chunk_size: int = 2000
    chunk_overlap: int = 200
    rag_top_k: int = 8

    # ── Storage (MinIO / S3) ──────────────────────────────────────────────────
    storage_endpoint: str = "localhost:9000"
    storage_access_key: str = "minioadmin"
    storage_secret_key: str = "minioadmin123"
    storage_bucket: str = "opsowl"
    storage_use_ssl: bool = False

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"


settings = Settings()
