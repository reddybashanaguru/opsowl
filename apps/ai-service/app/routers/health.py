from fastapi import APIRouter
from app.config import settings

router = APIRouter(tags=["health"])


@router.get("/health")
async def health() -> dict:
    return {"status": "ok", "service": "opsowl-ai"}


@router.get("/ready")
async def ready() -> dict:
    return {"status": "ready"}


@router.get("/config")
async def ai_config() -> dict:
    provider = settings.default_provider.lower()
    if provider == "ollama":
        return {
            "provider": "ollama",
            "mode": "air-gapped",
            "model": settings.ollama_model,
        }
    return {
        "provider": provider,
        "mode": "cloud",
        "model": None,
    }
