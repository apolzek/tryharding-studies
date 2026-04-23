"""Tests the repository's mapping layer using an in-memory mongo fake."""
from __future__ import annotations

import pytest

from app.repository import CatalogRepository, _to_api


class FakeCursor:
    def __init__(self, items):
        self._items = list(items)

    def limit(self, n):
        self._items = self._items[:n]
        return self

    def __aiter__(self):
        self._iter = iter(self._items)
        return self

    async def __anext__(self):
        try:
            return next(self._iter)
        except StopIteration:
            raise StopAsyncIteration


class FakeCollection:
    def __init__(self):
        self.docs: dict[str, dict] = {}

    async def insert_one(self, doc):
        self.docs[doc["_id"]] = doc

    async def find_one(self, q):
        return self.docs.get(q["_id"])

    def find(self):
        return FakeCursor(self.docs.values())

    async def delete_one(self, q):
        self.docs.pop(q["_id"], None)


class FakeDB:
    def __init__(self):
        self.col = FakeCollection()

    def __getitem__(self, _):
        return self.col


@pytest.mark.asyncio
async def test_roundtrip():
    repo = CatalogRepository(FakeDB())
    c = await repo.create({"name": "Books", "categories": ["fiction"]})
    assert c["id"] and c["name"] == "Books"

    found = await repo.get(c["id"])
    assert found["name"] == "Books"

    lst = await repo.list()
    assert len(lst) == 1


def test_to_api_rewrites_id():
    assert _to_api({"_id": "x", "name": "y"}) == {"id": "x", "name": "y"}
