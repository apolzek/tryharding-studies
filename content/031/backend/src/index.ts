import './telemetry.js';
import { buildServer } from './server.js';
import { ensureBucket } from './minio-client.js';

async function main(): Promise<void> {
  const app = await buildServer();

  try {
    await ensureBucket();
    app.log.info('minio bucket ready');
  } catch (err) {
    app.log.error({ err }, 'failed to ensure minio bucket (continuing)');
  }

  const port = parseInt(process.env.PORT ?? '8080', 10);
  const host = process.env.HOST ?? '0.0.0.0';

  await app.listen({ port, host });

  const shutdown = async (signal: string): Promise<void> => {
    app.log.info({ signal }, 'shutdown initiated');
    try {
      await app.close();
      process.exit(0);
    } catch (err) {
      app.log.error({ err }, 'shutdown error');
      process.exit(1);
    }
  };

  process.on('SIGTERM', () => void shutdown('SIGTERM'));
  process.on('SIGINT', () => void shutdown('SIGINT'));
}

main().catch((err: unknown) => {
  // eslint-disable-next-line no-console
  console.error('fatal startup error', err);
  process.exit(1);
});
