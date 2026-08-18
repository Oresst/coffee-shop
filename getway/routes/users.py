from typing import Annotated
from fastapi import APIRouter, Depends, HTTPException, status

from app.config import settings
from app.logger import get_logger, log_with_trace_id, ERROR
from services.user_client import UserServiceClient
from schemas.user import UserLogin, UserRegister

logger = get_logger(__name__)


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
        log_with_trace_id(logger, ERROR, "Login error", extra={"user.email": user_data.email, "error": str(e)})
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Не удалось залогиниться: {str(e)}",
        )

@router.post(
    "/register",
    status_code=status.HTTP_201_CREATED,
    summary="Регистрация"
)
async def register(
        user_data: UserRegister,
        user_service: Annotated[UserServiceClient, Depends(get_user_service_client)]
):
    try:
        result = await user_service.register(user_data.email, user_data.password, user_data.name)
        return {"status": "success", "data": result}
    except Exception as e:
        log_with_trace_id(logger, ERROR, "Register error",
                          extra={
                              "user.email": user_data.email,
                              "error": str(e),
                              "user.name": user_data.name,
                          })
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Не удалось зарегистрироваться: {str(e)}",
        )
