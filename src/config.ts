import { env } from 'hono/adapter';

export const getConfig = (c?: any) => {
  const merged = {
    ...(c ? env(c) : {}),
    ...(typeof process !== 'undefined' ? process.env : {})
  };

  return {
    PORT: merged.PORT || '8080',
    BOOKS_DIR: merged.BOOKS_DIR || './books',
    HOST: merged.HOST || 'localhost',
    REVERSE_PROXY: merged.REVERSE_PROXY === 'true',
    REVERSE_PROXY_HOST: merged.REVERSE_PROXY_HOST || '',
    REVERSE_PROXY_PORT: merged.REVERSE_PROXY_PORT || '',
  };
};
