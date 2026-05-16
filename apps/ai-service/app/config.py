from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    environment: str = "development"
    database_url: str = ""
    go_api_url: str = "http://localhost:8080"
    anthropic_api_key: str = ""
    openai_api_key: str = ""
    default_provider: str = "anthropic"
    default_model: str = "claude-opus-4-5"
    embedding_model: str = "text-embedding-3-small"
    max_chunk_size: int = 500
    rag_top_k: int = 8

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"


settings = Settings()
