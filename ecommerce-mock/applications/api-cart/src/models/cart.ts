import { Schema, model, Document } from 'mongoose';

export interface ICartItem {
  itemId: string;
  productId: string;
  name: string;
  price: number;
  quantity: number;
  addedAt: Date;
}

export interface ICart extends Document {
  userId: string;
  items: ICartItem[];
  createdAt: Date;
  updatedAt: Date;
}

const itemSchema = new Schema<ICartItem>(
  {
    itemId:    { type: String, required: true },
    productId: { type: String, required: true },
    name:      { type: String, required: true },
    price:     { type: Number, required: true },
    quantity:  { type: Number, required: true, min: 1 },
    addedAt:   { type: Date,   default: Date.now },
  },
  { _id: false }
);

const cartSchema = new Schema<ICart>({
  userId:    { type: String, required: true, unique: true },
  items:     [itemSchema],
  createdAt: { type: Date, default: Date.now, immutable: true },
  updatedAt: { type: Date, default: Date.now },
});

// Reset updatedAt on each save — keeps the TTL index alive
cartSchema.pre('save', function () {
  this.updatedAt = new Date();
});

export const Cart = model<ICart>('Cart', cartSchema, 'carts');
