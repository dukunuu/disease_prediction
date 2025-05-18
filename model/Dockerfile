# Dockerfile

# ---- Stage 1: Builder/Trainer ----
# Use a specific Python version for reproducibility
FROM python:3.11-slim as builder

# Set working directory
WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir --trusted-host pypi.python.org -r requirements.txt 

COPY model.py .
COPY model.csv .

RUN python model.py 

FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir --trusted-host pypi.python.org -r requirements.txt

COPY app.py .

COPY --from=builder /app/model_files ./model_files

EXPOSE 5000

CMD ["python", "app.py"]

