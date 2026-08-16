from typing import Annotated, List
from fastapi import APIRouter, Depends, HTTPException, status

from app.config import settings
from middleware.jwt import get_current_user
from schemas.order import OrderCreate
from services.order_client import OrderServiceClient


router = APIRouter(
    prefix="/orders",
    tags=["Orders"]
)


def get_order_service_client() -> OrderServiceClient:
    return OrderServiceClient(base_url=settings.ORDER_SERVICE_URL)


@router.post(
    "/create",
    status_code=status.HTTP_201_CREATED,
    summary="Создать новый заказ",
)
async def create_order(
    order_in: List[OrderCreate],
    order_client: Annotated[OrderServiceClient, Depends(get_order_service_client)],
    current_user: Annotated[dict, Depends(get_current_user)]
):
    user_id = current_user["sub"]
    items = [item.model_dump() for item in order_in]

    try:
        result = await order_client.create_order(user_id, items)
        return {"status": "success", "data": result}
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Не удалось создать заказ: {str(e)}",
        )
