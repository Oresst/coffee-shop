import httpx

from app.logger import get_logger, log_with_trace_id, INFO

logger = get_logger(__name__)


class UserServiceClient:
    base_url: str

    def __init__(self, base_url: str):
        self.base_url = base_url

    async def register(self, email: str, password: str, name: str):
        log_with_trace_id(logger, INFO, "Register attempt", extra={"user.email": email, "user.name": name})

        async with httpx.AsyncClient(timeout=10) as client:
            response = await client.post(
                f"{self.base_url}/api/register",
                json={"email": email, "password": password, "name": name},
            )
            response.raise_for_status()
            return response.json()

    async def login(self, email: str, password: str):
        log_with_trace_id(logger, INFO, "Login attempt", extra={"user.email": email})

        async with httpx.AsyncClient(timeout=10) as client:
            response = await client.post(
                f"{self.base_url}/api/login",
                json={"email": email, "password": password},
            )
            response.raise_for_status()
            return response.json()

    async def logout(self):
        pass

    async def refresh_token(self):
        pass
