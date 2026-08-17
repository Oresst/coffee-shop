import logging
import ecs_logging
from opentelemetry import trace

INFO = "INFO"
WARNING = "WARNING"
ERROR = "ERROR"

def get_logger(name: str = "api-gateway"):
    """Создает и возвращает логгер с ECS-форматированием."""
    logger = logging.getLogger(name)
    logger.setLevel(logging.INFO)

    # Проверяем, есть ли уже хендлеры, чтобы не дублировать
    if not logger.handlers:
        handler = logging.StreamHandler()
        handler.setFormatter(ecs_logging.StdlibFormatter())
        logger.addHandler(handler)

    return logger


def get_trace_id() -> str:
    span = trace.get_current_span()
    if span and span.is_recording():
        return f"{span.get_span_context().trace_id:032x}"
    return ""


def log_with_trace_id(logger, level: str, message: str, **kwargs):
    trace_id = get_trace_id()
    extra = kwargs.pop("extra", {})

    if trace_id:
        extra["trace.id"] = trace_id

    log_method = getattr(logger, level.lower())
    log_method(message, extra=extra, **kwargs)
