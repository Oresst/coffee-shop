import uvicorn
from fastapi import FastAPI, Depends

from middleware.jwt import get_current_user
from routes.users import router as users_router
from routes.orders import router as orders_router


app = FastAPI(
    title="Coffee Shop Gateway",
    description="Маршрутизатор запросов к микросервисам",
    version="1.0",
)

app.include_router(users_router)
app.include_router(
    orders_router,
    dependencies=[Depends(get_current_user)]
)


@app.get("/health")
async def health():
    return {"status": "ok"}


if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=8000,
        reload=True,
    )
