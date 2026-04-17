import os

import cv2
from ultralytics import YOLO

MODEL_PATH = os.environ.get("YOLO_MODEL", "yolov8s-worldv2.pt")
CONF_THRESHOLD = float(os.environ.get("YOLO_CONF", "0.10"))
DEFAULT_CLASSES = "person,glasses,eyeglasses,sunglasses,cell phone,laptop,bottle,cup,mug,book,keyboard,mouse,headphones,chair,backpack,monitor,pen,watch,remote control"
_raw = os.environ.get("OBJECT_CLASSES") or DEFAULT_CLASSES
CLASSES = [c.strip() for c in _raw.split(",") if c.strip()]


class ObjectDetector:
    def __init__(self, model_path: str = MODEL_PATH, conf: float = CONF_THRESHOLD, classes: list = None):
        self.model = YOLO(model_path)
        self.conf = conf
        self.classes = classes if classes is not None else CLASSES
        if "world" in model_path.lower() and self.classes:
            self.model.set_classes(self.classes)
        self.names = self.model.names
        print(f"[yolo] model={model_path} conf={conf} classes={self.classes or list(self.names.values())}")

    def detect(self, frame):
        r = self.model.predict(frame, conf=self.conf, verbose=False)[0]
        out = []
        for box in r.boxes:
            x1, y1, x2, y2 = [int(v) for v in box.xyxy[0].tolist()]
            cls = int(box.cls[0])
            score = float(box.conf[0])
            out.append((x1, y1, x2, y2, self.names[cls], score))
        return out


def draw(frame, detections):
    for x1, y1, x2, y2, name, score in detections:
        cv2.rectangle(frame, (x1, y1), (x2, y2), (255, 165, 0), 2)
        label = f"{name} {score:.2f}"
        (tw, th), _ = cv2.getTextSize(label, cv2.FONT_HERSHEY_SIMPLEX, 0.6, 2)
        cv2.rectangle(frame, (x1, y1 - th - 6), (x1 + tw + 4, y1), (255, 165, 0), -1)
        cv2.putText(frame, label, (x1 + 2, y1 - 4), cv2.FONT_HERSHEY_SIMPLEX, 0.6, (0, 0, 0), 2)
    return frame
