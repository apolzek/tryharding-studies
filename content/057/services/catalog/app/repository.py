"""Repository pattern over MongoDB."""
from __future__ import annotations

import uuid
from typing import Any


class CatalogRepository:
    def __init__(self, db):
        self.col = db["catalogs"]

    async def create(self, data: dict[str, Any]) -> dict[str, Any]:
        doc = {"_id": str(uuid.uuid4()), **data}
        await self.col.insert_one(doc)
        return _to_api(doc)

    async def get(self, catalog_id: str) -> dict[str, Any] | None:
        doc = await self.col.find_one({"_id": catalog_id})
        return _to_api(doc) if doc else None

    async def list(self) -> list[dict[str, Any]]:
        out = []
        async for d in self.col.find().limit(100):
            out.append(_to_api(d))
        return out

    async def delete(self, catalog_id: str) -> None:
        await self.col.delete_one({"_id": catalog_id})


def _to_api(doc: dict[str, Any]) -> dict[str, Any]:
    doc = dict(doc)
    doc["id"] = doc.pop("_id")
    return doc
