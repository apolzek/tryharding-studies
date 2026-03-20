import httpx
import asyncio
import time

RETRIES = 3
TIMEOUT = 10.0
RETRY_DELAY = 2


async def call(model_input):
    async with httpx.AsyncClient(timeout=TIMEOUT) as client:
        for attempt in range(RETRIES):
            try:
                resp = await client.post("http://ml-api:8000/predict/", json=model_input)
                resp.raise_for_status()
                return resp.json().get("Label", "Non-alert")
            except httpx.RequestError as e:
                pass
            except httpx.TimeoutException as e:
                pass
            except httpx.HTTPStatusError as e:
                break
            time.sleep(RETRY_DELAY)
        return None


def predict_api(model_input):
    return asyncio.run(call(model_input))
