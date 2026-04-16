## GPU-accelerated face recognition from a webcam with InsightFace

### Objectives

This PoC explores how to do real-time face recognition from a local webcam on Ubuntu, using a small set of reference photos as the identity database. The goal is not to build a production access-control system but to evaluate the developer experience of plugging an off-the-shelf face recognition stack — [InsightFace](https://github.com/deepinsight/insightface) with the `buffalo_l` model running on `onnxruntime-gpu` — into a containerized workflow that owns webcam capture, GPU inference, and result rendering. The PoC compares two consumption modes: a one-shot **snapshot** (capture one frame, identify, save annotated JPEG) and a **live** OpenCV window with bounding boxes and FPS overlay. Detection and recognition share the same model pipeline, so identity matching reuses the embedding produced during detection — no second forward pass.

### Prerequisites

- Ubuntu host with a USB/integrated webcam at `/dev/video0`
- NVIDIA GPU with driver ≥ 525 and [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html)
- Docker + Docker Compose v2
- `make`
- An X server running on the host (default on Ubuntu desktop) for the `live` window

### Architecture

```
host webcam (/dev/video0)
       │
       ▼
face-recognizer (CUDA + onnxruntime-gpu)
       │
       ├── InsightFace buffalo_l  (detect + embed in one pass)
       │       │
       │       ▼
       │   identity = argmax cosine(embedding, known_faces[*])
       │
       ├── known_faces/  ──► enroll  ──► output/embeddings.pkl
       └── output/snapshot_<ts>.jpg  |  live OpenCV window
```

The container reads frames straight from `/dev/video0`, runs detection + recognition on the GPU, and either saves an annotated JPEG (snapshot) or pushes frames to an OpenCV window via the host X11 socket (live).

### Reproducing

#### 1) Provide reference photos

Drop one image per person into `known_faces/`. The filename (without extension) becomes the displayed name. One face per photo, well-lit, frontal works best:

```
known_faces/
├── alice.jpg
├── bob.png
└── carol.jpg
```

#### 2) Build the image

```sh
make build
```

First build is ~2–3 GB (CUDA base image + onnxruntime-gpu wheels).

#### 3) Build the embedding database

```sh
make enroll
```

Runs InsightFace once per file, takes the largest detected face, and writes mean embeddings to `output/embeddings.pkl`. Output looks like:

```
[init] model=buffalo_l providers=['CUDAExecutionProvider', 'CPUExecutionProvider']
[enroll] alice <- alice.jpg
[enroll] bob <- bob.png
[enroll] wrote 2 identities to /data/output/embeddings.pkl
```

Re-run `make enroll` whenever you add or replace photos in `known_faces/`. The buffalo_l model (~280 MB) is downloaded into a named Docker volume on first run and reused after that.

#### 4) Test with a single snapshot

```sh
make snapshot
```

Captures one frame, runs recognition, prints matches in the terminal, and writes `output/snapshot_<unix_ts>.jpg` with bounding boxes and labels. Open it with any image viewer:

```sh
xdg-open output/snapshot_*.jpg
```

#### 5) Live recognition window

```sh
make live
```

The `xhost +local:docker` allowance is included in the target so the container can attach to your X server. Press `q` in the OpenCV window to quit. FPS prints to the terminal every 30 frames.

#### Tuning

| Variable | Default | Effect |
|---|---|---|
| `MATCH_THRESHOLD` | `0.45` | Cosine similarity required to label a face. Lower → more matches but more false positives. |
| `DET_SIZE` | `640` | Detection input size. Larger → catches smaller faces, slower. |
| `MODEL_NAME` | `buffalo_l` | Try `buffalo_s` (smaller, faster) if VRAM is tight. |

Override per run, e.g.:

```sh
MATCH_THRESHOLD=0.55 make snapshot
```

#### Troubleshooting

- `cannot open camera /dev/video0` — verify the device exists with `ls /dev/video*` and is not held by another app (`fuser /dev/video0`). Override with `--device /dev/video1` if needed.
- Live window does not appear — confirm `echo $DISPLAY` is set (e.g. `:0`) and re-run `xhost +local:docker`.
- All faces show `unknown` — re-check threshold, lighting, and that `make enroll` saw the right files.
- `CUDAExecutionProvider` missing in the `[init]` log — the container is falling back to CPU. Check `nvidia-smi` from host and verify the NVIDIA Container Toolkit is installed.

### Results

InsightFace + onnxruntime-gpu was the path of least resistance: detection and recognition come bundled in `FaceAnalysis`, embeddings are L2-normalized for free, and switching between CUDA and CPU is a provider-list swap rather than a rebuild. On the RTX 4070 the full pipeline at 1280×720 sits comfortably above 30 fps with `buffalo_l`, and the cosine threshold around `0.45` separated my reference photos from strangers without false positives in casual testing. The choice to enroll once and persist embeddings to disk keeps the live loop pure inference — no I/O on the hot path. The biggest practical gotcha was OpenCV: `opencv-python-headless` ships without `imshow`, so the live window silently produced nothing until the full `opencv-python` build replaced it. Using `network_mode: host` plus `/tmp/.X11-unix` was the simplest way to get the container window onto the desktop without a remote-display protocol; for a headless server, `make snapshot` writing JPEGs to a mounted volume is the equivalent flow.

### References

```
🔗 https://github.com/deepinsight/insightface
🔗 https://github.com/deepinsight/insightface/tree/master/python-package
🔗 https://onnxruntime.ai/docs/execution-providers/CUDA-ExecutionProvider.html
🔗 https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html
🔗 https://docs.opencv.org/4.x/dd/d43/tutorial_py_video_display.html
```
