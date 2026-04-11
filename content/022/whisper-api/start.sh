#!/bin/bash
set -e
DIR="$(cd "$(dirname "$0")" && pwd)"

NVIDIA_BASE="/home/apolzek/.cache/uv/archive-v0/LGT6Y3amK4Wrqkzeg-THx/lib/python3.13/site-packages/nvidia"
export LD_LIBRARY_PATH="\
${NVIDIA_BASE}/cublas/lib:\
${NVIDIA_BASE}/cudnn/lib:\
${NVIDIA_BASE}/cuda_runtime/lib:\
${NVIDIA_BASE}/cufft/lib:\
${NVIDIA_BASE}/curand/lib:\
${NVIDIA_BASE}/cusolver/lib:\
${NVIDIA_BASE}/cusparse/lib:\
${NVIDIA_BASE}/nvjitlink/lib:\
${LD_LIBRARY_PATH}"

source "$DIR/venv/bin/activate"
exec uvicorn server:app --host 0.0.0.0 --port 8090 --reload
