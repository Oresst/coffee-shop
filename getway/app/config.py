from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    APP_HOST: str = "127.0.0.1"
    APP_PORT: int = 8000

    ORDER_SERVICE_URL: str = "http://order-service:8080"
    USER_SERVICE_URL: str = "http://user-service:8080"
    PAYMENT_SERVICE_URL: str = "http://127.0.0.1:8003"

    SECRET_KEY: str = "super_secret_key"
    ALGORITHM: str = "HS256"

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore"
    )

settings = Settings()
