import httpx


class UserServiceClient:
    base_url: str

    def __init__(self, base_url: str):
        self.base_url = base_url

    async def login(self, username: str, password: str):
        async with httpx.AsyncClient(timeout=10) as client:
            response = await client.post(
                f"{self.base_url}/login",
                data={"username": username, "password": password}
            )
            response.raise_for_status()
            return response.json()

    async def logout(self):
        pass

    async def refresh_token(self):
        pass
