from typing import List

from pydantic import BaseModel


class OrderItems(BaseModel):
    product_id: int
    quantity: int
    price: float


class OrderCreate(BaseModel):
    items: List[OrderItems]
