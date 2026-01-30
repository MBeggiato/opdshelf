export function getBaseUrl(c: any, cfg: any) {
  if (cfg.REVERSE_PROXY) return cfg.REVERSE_PROXY_HOST;
  return `${c.req.header('x-forwarded-proto') || 'http'}://${c.req.header('host')}`;
}
