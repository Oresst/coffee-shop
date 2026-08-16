from pydantic import BaseModel, Field


class OrderCreate(BaseModel):
    item_id: int
    quantity: int = Field(..., gt=0)
    price: float = Field(..., ge=0.0)
