FROM cr.yandex/mirror/python:3.11-slim AS runner

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV PYTHONPATH=/app

COPY . /app

RUN pip install --no-cache-dir --user -r req.txt

EXPOSE 8000

CMD ["python", "app/main.py"]
