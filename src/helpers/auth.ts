import { Context } from "hono";

export function checkBasicAuth(c: Context) {
  const authHeader = c.req.header('Authorization');
  if (authHeader) {
    const match = authHeader.match(/^Basic\s+(.*)$/);
    if (match) {
      try {
        const credentials = Buffer.from(match[1], 'base64').toString();
        const splitIndex = credentials.indexOf(':');
        if (splitIndex !== -1) {
          const username = credentials.substring(0, splitIndex);
          const password = credentials.substring(splitIndex + 1);

          if (username === process.env.ADMIN_USERNAME && password === process.env.ADMIN_PASSWORD) {
            return true;
          }
        }
      } catch (e) {
      }
    }
  }
  return false;
}
