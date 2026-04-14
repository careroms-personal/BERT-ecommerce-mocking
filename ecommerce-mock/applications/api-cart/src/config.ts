export interface Config {
  port: number;
  mongoUri: string;
  productServiceUrl: string;
  prettyLogs: boolean;
}

const config: Config = {
  port: parseInt(process.env.PORT ?? '8083', 10),
  mongoUri: process.env.MONGO_URI ?? 'mongodb://ecommerce:ecommerce_pass@mongodb:27017/cart_db?authSource=cart_db',
  productServiceUrl: process.env.PRODUCT_SERVICE_URL ?? 'http://api-product:8081',
  prettyLogs: process.env.PRETTY_LOGS === 'true',
};

export default config;
