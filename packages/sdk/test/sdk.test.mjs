import assert from "node:assert/strict";
import http from "node:http";
import test from "node:test";
import { capture, init, parseTopFrame, readDeliveryId, wrap } from "../dist/index.js";

test("parseTopFrame extracts file path and line across stack formats", () => {
  const v8Named = [
    "TypeError: Cannot read properties of undefined",
    "    at processCharge (/app/src/webhooks/paystack.ts:47:15)",
    "    at router (/app/src/index.ts:12:4)",
  ].join("\n");
  assert.deepEqual(parseTopFrame(v8Named), {
    filePath: "/app/src/webhooks/paystack.ts",
    line: 47,
  });

  const fileUrl = [
    "Error: boom",
    "    at async POST (file:///srv/app/route.ts:22:9)",
  ].join("\n");
  assert.deepEqual(parseTopFrame(fileUrl), {
    filePath: "/srv/app/route.ts",
    line: 22,
  });

  const scoped = [
    "Error: scoped",
    "    at /app/packages/@acme/hooks/handler.ts:91:3",
  ].join("\n");
  assert.deepEqual(parseTopFrame(scoped), {
    filePath: "/app/packages/@acme/hooks/handler.ts",
    line: 91,
  });
});

test("readDeliveryId reads header across Fetch, Node, Fastify, and Hono", () => {
  const webReq = new Request("https://example.com/hook", {
    headers: { "X-PockDev-Delivery-Id": "dlv_web_01" },
  });
  assert.equal(readDeliveryId(webReq), "dlv_web_01");

  const nodeReq = { headers: { "x-pockdev-delivery-id": "dlv_node_02" } };
  assert.equal(readDeliveryId(nodeReq), "dlv_node_02");

  const honoCtx = {
    req: {
      header: (name) => (name === "x-pockdev-delivery-id" ? "dlv_hono_03" : undefined),
    },
  };
  assert.equal(readDeliveryId(honoCtx), "dlv_hono_03");

  const fastifyReq = { headers: { "X-PockDev-Delivery-Id": ["dlv_fast_04"] }, raw: {} };
  assert.equal(readDeliveryId(fastifyReq), "dlv_fast_04");
});

test("capture and wrap post exact errwatch.Payload to /v1/errors", async () => {
  const received = [];
  const server = http.createServer((req, res) => {
    const chunks = [];
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", () => {
      received.push({
        url: req.url,
        method: req.method,
        key: req.headers["x-pockdev-key"],
        body: JSON.parse(Buffer.concat(chunks).toString("utf8")),
      });
      res.writeHead(202, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ received: true, groupId: "grp_1" }));
    });
  });

  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  const endpoint = `http://127.0.0.1:${address.port}`;

  try {
    init({ key: "pk_live_test_123", endpoint });

    const nextHandler = wrap(async (_req) => {
      throw new Error("Missing data.amount");
    });

    const req = new Request("https://app.local/api/paystack", {
      headers: { "x-pockdev-delivery-id": "dlv_abc99" },
    });

    await assert.rejects(() => nextHandler(req), /Missing data\.amount/);
    assert.equal(received.length, 1);
    assert.equal(received[0].url, "/v1/errors");
    assert.equal(received[0].method, "POST");
    assert.equal(received[0].key, "pk_live_test_123");
    assert.equal(received[0].body.message, "Missing data.amount");
    assert.equal(received[0].body.deliveryId, "dlv_abc99");
    assert.equal(received[0].body.level, "error");
    assert.ok(received[0].body.filePath.includes("sdk.test.mjs"));
    assert.ok(received[0].body.line > 0);

    // Express handler with next(err)
    let forwardedErr = null;
    const expressHandler = wrap(async (_req, _res, _next) => {
      throw new Error("Express boom");
    });
    await expressHandler(
      { headers: { "x-pockdev-delivery-id": "dlv_exp_55" } },
      {},
      (err) => {
        forwardedErr = err;
      }
    );
    assert.ok(forwardedErr instanceof Error);
    assert.equal(received.length, 2);
    assert.equal(received[1].body.deliveryId, "dlv_exp_55");

    // Explicit capture with fatal level and extra context
    await capture(new Error("Fatal crash"), null, { level: "fatal", route: "/checkout" });
    assert.equal(received.length, 3);
    assert.equal(received[2].body.level, "fatal");
    assert.deepEqual(received[2].body.context, { route: "/checkout" });
  } finally {
    server.close();
  }
});
