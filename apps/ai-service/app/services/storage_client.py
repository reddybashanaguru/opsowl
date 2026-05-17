"""MinIO storage client for downloading documents to process."""
from __future__ import annotations

import io

from minio import Minio
from minio.error import S3Error

from app.config import settings

_client: Minio | None = None


def _get_client() -> Minio:
    global _client
    if _client is None:
        _client = Minio(
            settings.storage_endpoint,
            access_key=settings.storage_access_key,
            secret_key=settings.storage_secret_key,
            secure=settings.storage_use_ssl,
        )
    return _client


def download(key: str) -> bytes:
    """Download an object from the configured bucket and return its bytes."""
    client = _get_client()
    try:
        response = client.get_object(settings.storage_bucket, key)
        return response.read()
    except S3Error as exc:
        raise RuntimeError(f"storage download failed: {exc}") from exc
    finally:
        try:
            response.close()
            response.release_conn()
        except Exception:
            pass
