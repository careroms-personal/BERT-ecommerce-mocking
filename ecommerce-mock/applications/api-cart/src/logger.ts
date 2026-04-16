import type { LoggerOptions } from 'pino';
import config from './config';

const loggerOptions: LoggerOptions = {
  level: 'info',
  base: { service: 'api-cart' },
  // ISO timestamp keeps it consistent with Go/Rust services
  timestamp: () => `,"time":"${new Date().toISOString()}"`,
};

if (config.prettyLogs) {
  (loggerOptions as Record<string, unknown>).transport = {
    target: 'pino-pretty',
    options: { colorize: true, translateTime: 'HH:MM:ss' },
  };
}

export default loggerOptions;
