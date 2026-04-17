// routes/cart-5x.ts
// ─────────────────────────────────────────────────────────────────────────────
// Simulation routes that intentionally trigger 5xx responses.
// Mounted under /cart/sim/ — registered before /cart/:userId so Fastify's
// static-segment matching picks them up first.
//
// Endpoints
//   GET    /cart/sim/bad-query    → aggregation with invalid pipeline stage → 500
//   POST   /cart/sim/bad-insert   → bulkWrite with invalid array filter name → 500
//   DELETE /cart/sim/bad-delete   → bulkWrite with invalid array filter name → 500

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

  // ── POST /cart/sim/bad-insert ────────────────────────────────────────────
  // Attempts a bulkWrite with an array filter variable name containing '$$',
  // which MongoDB server rejects:
  //   MongoServerError: The array filter for identifier 'not$$existed' contains
  //   an invalid use of '$'
  fastify.post('/cart/sim/bad-insert', async (
    request: FastifyRequest,
    reply: FastifyReply
  ) => {
    const log = request.log;

    log.info({ category: 'SIM' }, 'sim: bad-insert triggered');

    try {
      await Cart.collection.bulkWrite([
        {
          updateOne: {
            filter: { userId: 'sim' },
            update: { $set: { not_existed: 'sim' } },
            arrayFilters: [{ 'not$$existed': true }],
          },
        },
      ]);

      // Should never reach here.
      log.warn({ category: 'SIM' }, 'sim: bad-insert unexpectedly succeeded');
      return reply.send({ sim: 'bad-insert', result: 'unexpected_success' });
    } catch (err) {
      const e = err as Error;
      log.error({ category: 'DB_ERROR', err: e.message }, 'sim: insert failed as expected');
      return reply.status(500).send({
        error: 'database error',
        sim: 'bad-insert',
        detail: e.message,
        category: 'DB_ERROR',
      });
    }
  });

  // ── DELETE /cart/sim/bad-delete ──────────────────────────────────────────
  // Attempts a bulkWrite delete with an array filter variable name containing
  // '$$', which MongoDB server rejects:
  //   MongoServerError: The array filter for identifier 'not$$existed' contains
  //   an invalid use of '$'
  fastify.delete('/cart/sim/bad-delete', async (
    request: FastifyRequest,
    reply: FastifyReply
  ) => {
    const log = request.log;

    log.info({ category: 'SIM' }, 'sim: bad-delete triggered');

    try {
      await Cart.collection.bulkWrite([
        {
          deleteOne: {
            filter: { userId: 'sim', 'not$$existed': 'sim' },
          },
        },
      ]);

      // Should never reach here.
      log.warn({ category: 'SIM' }, 'sim: bad-delete unexpectedly succeeded');
      return reply.send({ sim: 'bad-delete', result: 'unexpected_success' });
    } catch (err) {
      const e = err as Error;
      log.error({ category: 'DB_ERROR', err: e.message }, 'sim: delete failed as expected');
      return reply.status(500).send({
        error: 'database error',
        sim: 'bad-delete',
        detail: e.message,
        category: 'DB_ERROR',
      });
    }
  });
}
