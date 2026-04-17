import Fastify from 'fastify';
import { connectDB } from './db';
import cartRoutes from './routes/cart';
import cartSimRoutes from './routes/cart-5x';
import loggerOptions from './logger';
import config from './config';

const fastify = Fastify({ logger: loggerOptions });

async function start(): Promise<void> {
  fastify.log.info({ category: 'SYSTEM', version: '1.0.0' }, 'service starting');

  try {
    await connectDB(fastify.log);
  } catch (err) {
    const e = err as Error;
    fastify.log.error({ category: 'DB_ERROR', err: e.message }, 'failed to connect to MongoDB, continuing — requests may fail');
  }

  fastify.get('/health', async () => {
    fastify.log.info({ category: 'SYSTEM' }, 'health check ok');
    return { status: 'ok', service: 'api-cart' };
  });

  fastify.register(cartSimRoutes);
  fastify.register(cartRoutes);

  fastify.addHook('onResponse', (request, reply, done) => {
    const status = reply.statusCode;
    const category = status === 404 ? 'NOT_FOUND' : status >= 500 ? 'ERROR' : 'CART';
    request.log.info(
      { category, method: request.method, url: request.url, status, response_time_ms: reply.elapsedTime.toFixed(2) },
      'request completed'
    );
    done();
  });

  try {
    await fastify.listen({ port: config.port, host: '0.0.0.0' });
    fastify.log.info({ category: 'SYSTEM', port: config.port }, 'server listening');
  } catch (err) {
    const e = err as Error;
    fastify.log.fatal({ category: 'ERROR', err: e.message }, 'server failed to start');
    process.exit(1);
  }
}

const shutdown = async (signal: string): Promise<void> => {
  fastify.log.info({ category: 'SYSTEM', signal }, 'shutdown signal received');
  await fastify.close();
  fastify.log.info({ category: 'SYSTEM' }, 'service stopped');
  process.exit(0);
};

process.on('SIGTERM', () => shutdown('SIGTERM'));
process.on('SIGINT',  () => shutdown('SIGINT'));

start();
