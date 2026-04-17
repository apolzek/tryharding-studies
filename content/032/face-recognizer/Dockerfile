FROM nvidia/cuda:12.2.2-cudnn8-runtime-ubuntu22.04

ENV DEBIAN_FRONTEND=noninteractive \
    PYTHONUNBUFFERED=1 \
    INSIGHTFACE_HOME=/models

RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential \
        python3 python3-pip python3-dev \
        libgl1 libglib2.0-0 libgomp1 \
        libxext6 libsm6 libxrender1 \
        v4l-utils \
    && rm -rf /var/lib/apt/lists/*

RUN ln -sf /usr/bin/python3 /usr/bin/python

WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends git && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
RUN pip install --no-cache-dir git+https://github.com/openai/CLIP.git && python -c "import clip; print('clip OK:', clip.__file__)"

RUN python -c "from ultralytics import YOLO; YOLO('yolov8n.pt'); YOLO('yolov8s-worldv2.pt')"

COPY . .

ENTRYPOINT ["python", "-u", "main.py"]
