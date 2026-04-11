FROM python:3.9-slim

WORKDIR /app

COPY . /app

ENV POSTGRES_USER=rinha
ENV POSTGRES_PASSWORD=rinhadebackend
ENV POSTGRES_HOST=postgres
ENV POSTGRES_PORT=5432
ENV POSTGRES_DB=app_db

RUN pip install -r requirements.txt
EXPOSE 8888
CMD ["python", "/app/main.py"]
