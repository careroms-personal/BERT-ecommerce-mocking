// routes/cart-5x.ts
// ─────────────────────────────────────────────────────────────────────────────
// Simulation routes that intentionally trigger 5xx responses.
// Mounted under /cart/sim/ — registered before /cart/:userId so Fastify's
// static-segment matching picks them up first.
//
// Endpoints
//   GET /cart/sim/bad-query   → aggregation with invalid pipeline stage → 500

import type { FastifyInstance, FastifyReply, FastifyRequest } from 'fastify';
import { Cart } from '../models/cart';

export default async function cartSimRoutes(fastify: FastifyInstance): Promise<void> {

  // ── GET /cart/sim/bad-query ──────────────────────────────────────────────
  // Runs an aggregation pipeline with a stage name that does not exist.
  // MongoDB rejects it server-side:
  //   MongoServerError: Unrecognized pipeline stage name: '$notAStage'
  fastify.get('/cart/sim/bad-query', async (
    request: FastifyRequest,
    reply: FastifyReply
  ) => {
    const log = request.log;

    log.info({ category: 'SIM' }, 'sim: bad-query triggered');

    try {
      // $notAStage is not a valid MongoDB aggregation stage.
      // MongoDB will reject this and throw MongoServerError.
      await (Cart.aggregate as Function)([{ $notAStage: {} }]);

      // Should never reach here.
      log.warn({ category: 'SIM' }, 'sim: bad-query unexpectedly succeeded');
      return reply.send({ sim: 'bad-query', result: 'unexpected_success' });
    } catch (err) {
      const e = err as Error;
      log.error({ category: 'DB_ERROR', err: e.message }, 'sim: query failed as expected');
      return reply.status(500).send({
        error: 'database error',
        sim: 'bad-query',
        detail: e.message,
        category: 'DB_ERROR',
      });
    }
  });
}
