from contextlib import asynccontextmanager

import uvicorn
import os
import logging
from fastapi import FastAPI, Depends
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.instrumentation.httpx import HTTPXClientInstrumentor
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

from middleware.jwt import get_current_user
from routes.users import router as users_router
from routes.orders import router as orders_router


logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


# --- Инициализация трейсинга ---
def init_tracing(app: FastAPI):
    endpoint = os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT")
    if not endpoint:
        logger.warning("OTEL_EXPORTER_OTLP_ENDPOINT not set. Tracing disabled.")
        return

    try:
        exporter = OTLPSpanExporter(endpoint=endpoint, insecure=True)

        resource = Resource(attributes={
            SERVICE_NAME: os.environ.get("OTEL_SERVICE_NAME", "api-gateway"),
            "service.version": os.environ.get("SERVICE_VERSION", "1.0.0"),
            "deployment.environment": os.environ.get("ENVIRONMENT", "development"),
        })

        provider = TracerProvider(resource=resource)
        processor = BatchSpanProcessor(exporter)
        provider.add_span_processor(processor)
        trace.set_tracer_provider(provider)

        FastAPIInstrumentor.instrument_app(app, tracer_provider=provider)

        logger.info(f"Tracing enabled: sending traces to {endpoint}")

    except Exception as e:
        logger.error(f"Failed to initialize tracing: {e}")


app = FastAPI(
    title="Coffee Shop Gateway",
    description="Маршрутизатор запросов к микросервисам",
    version="1.0",
)

init_tracing(app)
HTTPXClientInstrumentor().instrument()

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
