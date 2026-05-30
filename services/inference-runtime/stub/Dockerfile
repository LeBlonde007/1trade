# CPU stub runtime (local dev only). Zero deps — Python stdlib. The real GPU worker is
# deploy/docker/Dockerfile.vllm. Build context: this directory.
FROM python:3.12-slim
WORKDIR /app
COPY server.py .
RUN useradd -u 65532 app
USER 65532
EXPOSE 8000
ENTRYPOINT ["python", "/app/server.py"]
