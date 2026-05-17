from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.routers import health


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    yield
    # Shutdown — close DB pool if open
    try:
        from app.db import close_pool
        await close_pool()
    except Exception:
        pass


app = FastAPI(
    title="OpsOwl AI Service",
    version="1.0.0",
    description="AI orchestration service for OpsOwl — LLM routing, RAG, document intelligence, PII masking.",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(health.router)

from app.routers import documents, copilot  # noqa: E402
app.include_router(documents.router)
app.include_router(copilot.router)
