import mongoose from 'mongoose';
import type { FastifyBaseLogger } from 'fastify';
import config from './config';

export async function connectDB(log: FastifyBaseLogger): Promise<void> {
  mongoose.connection.on('connected', () => {
    log.info({ category: 'SYSTEM' }, 'MongoDB connected');
  });

  mongoose.connection.on('error', (err: Error) => {
    log.error({ category: 'DB_ERROR', err: err.message }, 'MongoDB connection error');
  });

  mongoose.connection.on('disconnected', () => {
    log.warn({ category: 'DB_ERROR' }, 'MongoDB disconnected');
  });

  await mongoose.connect(config.mongoUri);
}
