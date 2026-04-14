import { v4 as uuidv4 } from 'uuid';
import axios from 'axios';
import type { FastifyInstance, FastifyRequest, FastifyReply } from 'fastify';
import { Cart } from '../models/cart';
import config from '../config';

interface UserParams { userId: string }
interface ItemParams { userId: string; itemId: string }
interface AddItemBody { productId: string; name: string; price: number; quantity?: number }

interface StockResponse { available: boolean; stock: number; product_id: string }

export default async function cartRoutes(fastify: FastifyInstance): Promise<void> {

  // ── GET /cart/:userId ─────────────────────────────────────────────────────
  fastify.get<{ Params: UserParams }>('/cart/:userId', async (
    request: FastifyRequest<{ Params: UserParams }>,
    reply: FastifyReply
  ) => {
    const { userId } = request.params;
    const log = request.log;

    log.info({ category: 'CART', userId }, 'fetching cart');

    let cart;
    try {
      cart = await Cart.findOne({ userId }).lean();
    } catch (err) {
      const e = err as Error;
      log.error({ category: 'DB_ERROR', userId, err: e.message }, 'failed to fetch cart');
      return reply.status(500).send({ error: 'database error' });
    }

    if (!cart) {
      log.warn({ category: 'NOT_FOUND', userId }, 'cart not found');
      return reply.status(404).send({ error: 'cart not found', user_id: userId });
    }

    const ageMs = Date.now() - new Date(cart.updatedAt).getTime();
    if (ageMs > 86_400_000) {
      log.warn({ category: 'CART_EXPIRED', userId, age_hours: Math.floor(ageMs / 3_600_000) }, 'cart has expired');
      await Cart.deleteOne({ userId });
      return reply.status(410).send({ error: 'cart expired', user_id: userId });
    }

    log.info({ category: 'CART', userId, item_count: cart.items.length }, 'cart fetched');
    return reply.send(cart);
  });

  // ── POST /cart/:userId/items ──────────────────────────────────────────────
  fastify.post<{ Params: UserParams; Body: AddItemBody }>('/cart/:userId/items', async (
    request: FastifyRequest<{ Params: UserParams; Body: AddItemBody }>,
    reply: FastifyReply
  ) => {
    const { userId } = request.params;
    const { productId, name, price, quantity = 1 } = request.body ?? ({} as AddItemBody);
    const log = request.log;

    if (!productId || !name || price === undefined) {
      log.warn({ category: 'ERROR', userId }, 'add item — missing required fields');
      return reply.status(400).send({ error: 'productId, name, and price are required' });
    }

    log.info({ category: 'CART', userId, product_id: productId, quantity }, 'adding item to cart');

    try {
      const { data } = await axios.get<StockResponse>(
        `${config.productServiceUrl}/products/${productId}/stock`,
        { timeout: 3000 }
      );
      if (!data.available) {
        log.warn({ category: 'STOCK', userId, product_id: productId }, 'add item rejected — product out of stock');
        return reply.status(409).send({ error: 'product out of stock', product_id: productId });
      }
      if (data.stock < quantity) {
        log.warn({ category: 'STOCK', userId, product_id: productId, requested: quantity, available: data.stock }, 'add item — insufficient stock');
        return reply.status(409).send({ error: 'insufficient stock', available: data.stock });
      }
    } catch (err) {
      const e = err as { response?: { status: number }; message: string };
      if (e.response?.status === 404) {
        log.warn({ category: 'NOT_FOUND', userId, product_id: productId }, 'add item — product not found');
        return reply.status(404).send({ error: 'product not found', product_id: productId });
      }
      log.warn({ category: 'ERROR', userId, product_id: productId, err: e.message }, 'stock check failed, proceeding optimistically');
    }

    try {
      let cart = await Cart.findOne({ userId });
      if (!cart) {
        cart = new Cart({ userId, items: [] });
      }

      const existing = cart.items.find(i => i.productId === productId);
      if (existing) {
        existing.quantity += quantity;
        log.info({ category: 'CART', userId, product_id: productId, new_quantity: existing.quantity }, 'cart item quantity updated');
      } else {
        cart.items.push({ itemId: uuidv4(), productId, name, price, quantity, addedAt: new Date() });
        log.info({ category: 'CART', userId, product_id: productId, quantity }, 'cart item added');
      }

      await cart.save();
      return reply.status(201).send(cart);
    } catch (err) {
      const e = err as Error;
      log.error({ category: 'DB_ERROR', userId, err: e.message }, 'failed to save cart');
      return reply.status(500).send({ error: 'database error' });
    }
  });

  // ── DELETE /cart/:userId/items/:itemId ────────────────────────────────────
  fastify.delete<{ Params: ItemParams }>('/cart/:userId/items/:itemId', async (
    request: FastifyRequest<{ Params: ItemParams }>,
    reply: FastifyReply
  ) => {
    const { userId, itemId } = request.params;
    const log = request.log;

    log.info({ category: 'CART', userId, item_id: itemId }, 'removing cart item');

    let cart;
    try {
      cart = await Cart.findOne({ userId });
    } catch (err) {
      const e = err as Error;
      log.error({ category: 'DB_ERROR', userId, err: e.message }, 'failed to fetch cart for item removal');
      return reply.status(500).send({ error: 'database error' });
    }

    if (!cart) {
      log.warn({ category: 'NOT_FOUND', userId }, 'cart not found for item removal');
      return reply.status(404).send({ error: 'cart not found' });
    }

    const before = cart.items.length;
    cart.items = cart.items.filter(i => i.itemId !== itemId);

    if (cart.items.length === before) {
      log.warn({ category: 'NOT_FOUND', userId, item_id: itemId }, 'item not found in cart');
      return reply.status(404).send({ error: 'item not found', item_id: itemId });
    }

    try {
      await cart.save();
    } catch (err) {
      const e = err as Error;
      log.error({ category: 'DB_ERROR', userId, err: e.message }, 'failed to save cart after item removal');
      return reply.status(500).send({ error: 'database error' });
    }

    log.info({ category: 'CART', userId, item_id: itemId, remaining_items: cart.items.length }, 'cart item removed');
    return reply.send(cart);
  });

  // ── DELETE /cart/:userId ──────────────────────────────────────────────────
  fastify.delete<{ Params: UserParams }>('/cart/:userId', async (
    request: FastifyRequest<{ Params: UserParams }>,
    reply: FastifyReply
  ) => {
    const { userId } = request.params;
    const log = request.log;

    log.info({ category: 'CART', userId }, 'clearing cart');

    try {
      await Cart.deleteOne({ userId });
    } catch (err) {
      const e = err as Error;
      log.error({ category: 'DB_ERROR', userId, err: e.message }, 'failed to clear cart');
      return reply.status(500).send({ error: 'database error' });
    }

    log.info({ category: 'CART', userId }, 'cart cleared');
    return reply.send({ message: 'cart cleared', user_id: userId });
  });

  // ── POST /cart/:userId/sync ───────────────────────────────────────────────
  fastify.post<{ Params: UserParams }>('/cart/:userId/sync', async (
    request: FastifyRequest<{ Params: UserParams }>,
    reply: FastifyReply
  ) => {
    const { userId } = request.params;
    const log = request.log;

    log.info({ category: 'CART', userId }, 'cart sync started');

    let cart;
    try {
      cart = await Cart.findOne({ userId });
    } catch (err) {
      const e = err as Error;
      log.error({ category: 'DB_ERROR', userId, err: e.message }, 'failed to fetch cart for sync');
      return reply.status(500).send({ error: 'database error' });
    }

    if (!cart) {
      log.warn({ category: 'NOT_FOUND', userId }, 'cart not found for sync');
      return reply.status(404).send({ error: 'cart not found' });
    }

    const mismatches: object[] = [];
    const removed: string[] = [];

    for (const item of cart.items) {
      try {
        const { data } = await axios.get<StockResponse>(
          `${config.productServiceUrl}/products/${item.productId}/stock`,
          { timeout: 3000 }
        );

        if (!data.available) {
          log.warn({ category: 'SYNC_ERROR', userId, product_id: item.productId, item_id: item.itemId }, 'sync — product out of stock, removing from cart');
          mismatches.push({ itemId: item.itemId, productId: item.productId, reason: 'out_of_stock' });
          removed.push(item.itemId);
        } else if (data.stock < item.quantity) {
          log.warn({ category: 'SYNC_ERROR', userId, product_id: item.productId, requested: item.quantity, available: data.stock }, 'sync — stock mismatch, adjusting quantity');
          mismatches.push({ itemId: item.itemId, productId: item.productId, reason: 'quantity_adjusted', from: item.quantity, to: data.stock });
          item.quantity = data.stock;
        }
      } catch (err) {
        const e = err as { response?: { status: number }; message: string };
        if (e.response?.status === 404) {
          log.warn({ category: 'SYNC_ERROR', userId, product_id: item.productId }, 'sync — product no longer exists, removing');
          mismatches.push({ itemId: item.itemId, productId: item.productId, reason: 'product_not_found' });
          removed.push(item.itemId);
        } else {
          log.error({ category: 'SYNC_ERROR', userId, product_id: item.productId, err: e.message }, 'sync — stock check failed');
        }
      }
    }

    cart.items = cart.items.filter(i => !removed.includes(i.itemId));

    try {
      await cart.save();
    } catch (err) {
      const e = err as Error;
      log.error({ category: 'DB_ERROR', userId, err: e.message }, 'failed to save cart after sync');
      return reply.status(500).send({ error: 'database error' });
    }

    log.info({ category: 'CART', userId, mismatch_count: mismatches.length, remaining_items: cart.items.length }, 'cart sync complete');
    return reply.send({ synced: true, mismatches, cart });
  });
}
