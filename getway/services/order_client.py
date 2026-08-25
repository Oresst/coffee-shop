import httpx


class OrderServiceClient:
    base_url: str

    def __init__(self, base_url: str):
        self.base_url = base_url

    async def create_order(self, user_id: int, items: list):
        async with httpx.AsyncClient(timeout=10) as client:
            response = await client.post(
                f"{self.base_url}/api/create_order",
                json={"user_id": user_id, "items": items},
                headers={"X-User-Id": str(user_id)},
            )
            response.raise_for_status()
            return response.json()
