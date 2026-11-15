# @pockdev/sdk

Zero-dependency TypeScript SDK for [PockDev](https://pockdev.vercel.app). Captures unhandled crashes, extracts the top stack frame (`filePath` and `line`), reads the incoming `x-pockdev-delivery-id` header, and links each crash directly to paused webhooks in your PockDev engine so you can fix issues right from WhatsApp.

## Install

```bash
npm install @pockdev/sdk
```

Requires Node.js 18 or newer.

## Quick start (3 lines)

Call `init` once at startup with your project SDK key (`pk_...`) and wrap any webhook or route handler with `wrap`:

```ts
import { init, wrap } from "@pockdev/sdk";

init({ key: process.env.POCKDEV_KEY!, endpoint: "https://pockdev-engine.rimbra.live" });

export const POST = wrap(async (req: Request) => {
  const event = await req.json();
  await handleWebhook(event);
  return Response.json({ ok: true });
});
```

When a handler throws, `@pockdev/sdk` reports the crash to `POST /v1/errors` without masking your error or breaking your host process.

## Manual `capture(err, req, context)` by framework

Use `capture` inside custom error handlers or `try / catch` blocks. Passing `req` lets PockDev read `x-pockdev-delivery-id` automatically. Set `{ level: "fatal" }` in `context` for critical crashes.

### Next.js App Router

```ts
import { capture, init } from "@pockdev/sdk";

init({ key: process.env.POCKDEV_KEY!, endpoint: "https://pockdev-engine.rimbra.live" });

export async function POST(req: Request) {
  try {
    const payload = await req.json();
    await processPaystackCharge(payload.data);
    return Response.json({ received: true });
  } catch (err) {
    await capture(err, req, { provider: "paystack", route: "/api/webhooks/paystack" });
    return Response.json({ error: "processing failed" }, { status: 500 });
  }
}
```

### Express

```ts
import express from "express";
import { capture, init, wrap } from "@pockdev/sdk";

init({ key: process.env.POCKDEV_KEY!, endpoint: "https://pockdev-engine.rimbra.live" });

const app = express();
app.use(express.json());

// Automatic capture with wrap()
app.post("/webhooks/stripe", wrap(async (req, res) => {
  await processStripeEvent(req.body);
  res.json({ received: true });
}));

// Or manual capture in Express error middleware
app.use(async (err: unknown, req: express.Request, res: express.Response, _next: express.NextFunction) => {
  await capture(err, req, { route: req.path });
  res.status(500).json({ error: "internal error" });
});
```

### Fastify

```ts
import Fastify from "fastify";
import { capture, init, wrap } from "@pockdev/sdk";

init({ key: process.env.POCKDEV_KEY!, endpoint: "https://pockdev-engine.rimbra.live" });

const fastify = Fastify();

fastify.post("/webhooks/paystack", wrap(async (request, reply) => {
  await processPaystackCharge(request.body);
  return reply.send({ received: true });
}));

fastify.setErrorHandler(async (err, request, reply) => {
  await capture(err, request, { route: request.url, level: "fatal" });
  return reply.code(500).send({ error: "internal error" });
});
```

### Hono

```ts
import { Hono } from "hono";
import { capture, init, wrap } from "@pockdev/sdk";

init({ key: process.env.POCKDEV_KEY!, endpoint: "https://pockdev-engine.rimbra.live" });

const app = new Hono();

app.post("/webhooks/stripe", wrap(async (c) => {
  const payload = await c.req.json();
  await processStripeEvent(payload);
  return c.json({ received: true });
}));

app.onError(async (err, c) => {
  await capture(err, c, { route: c.req.path });
  return c.json({ error: "internal error" }, 500);
});
```

## API

- `init({ key, endpoint })`: Configures the SDK key (`X-PockDev-Key`) and engine URL. Defaults `endpoint` to `http://localhost:8080`.
- `wrap(handler)`: Wraps sync or async handlers for Node HTTP, Express, Fastify, Hono, and Next.js App Router. Captures thrown errors and forwards them normally (`next(err)` for Express, rethrow elsewhere).
- `capture(err, req?, extraContext?)`: Sends `{ message, stack, filePath, line, level, deliveryId, context }` to `POST /v1/errors`. Never throws.

## License

MIT
