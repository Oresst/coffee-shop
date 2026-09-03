from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    APP_HOST: str = "127.0.0.1"
    APP_PORT: int = 8000

    ORDER_SERVICE_URL: str = "http://order-service:8080"
    USER_SERVICE_URL: str = "http://user-service:8080"
    PAYMENT_SERVICE_URL: str = "http://127.0.0.1:8003"
    INVENTORY_SERVICE_URL: str = "http://inventory-service:8080"

    SECRET_KEY: str = "super_secret_key"
    ALGORITHM: str = "HS256"

    OTEL_EXPORTER_OTLP_ENDPOINT: str = "http://jaeger:4317"

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore"
    )

settings = Settings()
