from typing import Annotated
from fastapi import APIRouter, Depends, HTTPException, status

from app.config import settings
from services.user_client import UserServiceClient
from schemas.user import UserLogin


router = APIRouter(
    prefix="/users",
    tags=["Users"]
)


def get_user_service_client() -> UserServiceClient:
    return UserServiceClient(base_url=settings.USER_SERVICE_URL)


@router.post(
    "/login",
    status_code=status.HTTP_200_OK,
    summary="Вход",
)
async def login(
        user_data: UserLogin,
        user_service: Annotated[UserServiceClient, Depends(get_user_service_client)]
):
    try:
        result = await user_service.login(user_data.email, user_data.password)
        return {"status": "success", "data": result}
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Не удалось залогиниться: {str(e)}",
        )
