"""Inter-service authentication for the Python AI service.

The Go API sends `Authorization: Bearer <AI_SERVICE_SECRET>` on every request.
If AI_SERVICE_SECRET is not configured the check is skipped (dev/test convenience).
"""
from __future__ import annotations

import os

from fastapi import HTTPException, Security, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

_bearer = HTTPBearer(auto_error=False)


def require_service_token(
    credentials: HTTPAuthorizationCredentials | None = Security(_bearer),
) -> None:
    """FastAPI dependency — raises 401 when the service secret does not match."""
    secret = os.getenv("AI_SERVICE_SECRET", "")
    if not secret:
        return  # unconfigured → open (dev/test)
    if credentials is None or credentials.credentials != secret:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid or missing service token",
        )
