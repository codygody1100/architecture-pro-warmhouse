FROM python:3.13-slim
WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
COPY app.py ./
ENV DATABASE_PATH=/data/telemetry.db
RUN mkdir /data
EXPOSE 8084
CMD ["gunicorn", "--bind", "0.0.0.0:8084", "--workers", "1", "--threads", "4", "app:create_app()"]
