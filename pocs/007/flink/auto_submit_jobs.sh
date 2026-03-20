#!/bin/bash

echo "Starting Flink JobManager..."
/docker-entrypoint.sh jobmanager &

# Wait for JobManager to start up
echo "Waiting for JobManager to become available..."
sleep 1

# Submit all .py jobs
echo "Submitting Python jobs:"
for file in /opt/flink/jobs/*_job.py; do
  echo "Submitting $file"
  /opt/flink/bin/flink run -py "$file" -m http://localhost:8081 || echo "Failed to run $file"
done
