FROM python:3.12-slim
WORKDIR /app
COPY loadgen/*.py ./
ENV PYTHONUNBUFFERED=1
EXPOSE 8000
CMD ["python3", "controld.py"]
