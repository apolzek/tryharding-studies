import { Client } from 'minio';

export const bucket = process.env.MINIO_BUCKET ?? 'rrweb-sessions';

export const minioClient = new Client({
  endPoint: process.env.MINIO_ENDPOINT ?? 'minio',
  port: parseInt(process.env.MINIO_PORT ?? '9000', 10),
  useSSL: (process.env.MINIO_USE_SSL ?? 'false') === 'true',
  accessKey: process.env.MINIO_ACCESS_KEY ?? 'minioadmin',
  secretKey: process.env.MINIO_SECRET_KEY ?? 'minioadmin',
});

export async function ensureBucket(): Promise<void> {
  try {
    const exists = await minioClient.bucketExists(bucket);
    if (!exists) {
      await minioClient.makeBucket(bucket, 'us-east-1');
    }
  } catch (err) {
    const code = (err as { code?: string }).code;
    if (code === 'BucketAlreadyOwnedByYou' || code === 'BucketAlreadyExists') {
      return;
    }
    throw err;
  }
}

export async function putObjectWithRetry(
  key: string,
  body: Buffer,
  contentType: string,
  maxAttempts = 3,
): Promise<void> {
  let lastErr: unknown;
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      await minioClient.putObject(bucket, key, body, body.length, {
        'Content-Type': contentType,
      });
      return;
    } catch (err) {
      lastErr = err;
      if (attempt === maxAttempts) break;
      const backoff = 100 * 2 ** (attempt - 1);
      await new Promise((resolve) => setTimeout(resolve, backoff));
    }
  }
  throw lastErr;
}
