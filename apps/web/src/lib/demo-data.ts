import type {
  CreateProjectInput,
  Delivery,
  Incident,
  IncidentAction,
  Overview,
  Project,
} from "./types";

export const DEMO_PROJECTS: Project[] = [
  {
    id: "prj_paystack_01",
    name: "Paystack Checkout",
    slug: "paystack-checkout",
    endpointKey: "ep_9f3c7a1b2e4d8c0f6a5b1e2d",
    sdkKey: "pk_live_paystack_71a9c4e2f8b0d3a6",
    targetUrl: "https://api.kora-shop.app/webhooks/paystack",
    githubRepo: "Mascottz/kora-checkout",
    whatsappPhone: "+2348094412201",
    status: "active",
    createdAt: "2026-11-12T09:15:00Z",
  },
  {
    id: "prj_stripe_02",
    name: "Stripe Billing",
    slug: "stripe-billing",
    endpointKey: "ep_4b8d1e7c9a0f3e2b5d6c8a1f",
    sdkKey: "pk_live_stripe_38d2b6a9c1e5f704",
    targetUrl: "https://billing.kora-shop.app/api/hooks/stripe",
    githubRepo: "Mascottz/kora-billing",
    whatsappPhone: "+2348094412201",
    status: "active",
    createdAt: "2026-11-11T14:30:00Z",
  },
  {
    id: "prj_flw_03",
    name: "Flutterwave Payouts",
    slug: "flutterwave-payouts",
    endpointKey: "ep_7d2e5f8a1c4b9e0d3f6a2c8b",
    sdkKey: "pk_live_flw_92e7a1c4b8d0f539",
    targetUrl: "https://payouts.kora-shop.app/webhooks/flw",
    githubRepo: "Mascottz/kora-payouts",
    whatsappPhone: "+2348094412201",
    status: "active",
    createdAt: "2026-11-10T11:00:00Z",
  },
  {
    id: "prj_dispatch_04",
    name: "Rimbra Dispatch",
    slug: "rimbra-dispatch",
    endpointKey: "ep_2c6a9d0e4f7b1a8c5e3d9f2a",
    sdkKey: "pk_live_rimbra_64c8f0a2d7b1e395",
    targetUrl: "https://dispatch.kora-shop.app/hooks/delivery",
    githubRepo: "Mascottz/kora-dispatch",
    whatsappPhone: "+2348094412201",
    status: "active",
    createdAt: "2026-11-09T16:45:00Z",
  },
];

export const DEMO_INCIDENTS: Incident[] = [
  {
    id: "inc_paystack_drift_01",
    projectId: "prj_paystack_01",
    title: "TypeError: Cannot read properties of undefined (reading 'amount')",
    plainSummary:
      "Paystack Checkout crashed on line 47 of src/webhooks/paystack.ts. Paystack started sending data.charged_amount instead of data.amount, so your order handler threw when converting kobo to naira. Incoming payment notices are safely held until the fix lands.",
    filePath: "src/webhooks/paystack.ts:47",
    commitSha: "8a4f2c1",
    status: "open",
    pausedCount: 6,
    prUrl: "",
    lastSeenAt: "2026-11-14T19:52:00Z",
  },
  {
    id: "inc_stripe_null_02",
    projectId: "prj_stripe_02",
    title: "TypeError: Cannot read properties of null (reading 'email')",
    plainSummary:
      "Stripe Billing crashed on line 112 of src/billing/invoice-hook.ts when a guest invoice failed without a customer email attached. PockDev caught the stack trace and held 3 billing notices so no customer renewal state is lost.",
    filePath: "src/billing/invoice-hook.ts:112",
    commitSha: "3e91b0d",
    status: "open",
    pausedCount: 3,
    prUrl: "",
    lastSeenAt: "2026-11-14T19:41:00Z",
  },
];

export const DEMO_DELIVERIES: Delivery[] = [
  {
    id: "dlv_98a1c4f0",
    projectId: "prj_paystack_01",
    status: "paused",
    httpStatus: 500,
    attemptCount: 4,
    schemaDriftNote: "Missing key data.amount; new key data.charged_amount",
    payloadPreview:
      '{"event":"charge.success","data":{"reference":"PSK_88412","charged_amount":450000,"currency":"NGN","customer":{"email":"tunde@lagos.dev"}}}',
    linkedIncidentId: "inc_paystack_drift_01",
    createdAt: "2026-11-14T19:51:42Z",
  },
  {
    id: "dlv_98a1c3d9",
    projectId: "prj_paystack_01",
    status: "paused",
    httpStatus: 500,
    attemptCount: 4,
    schemaDriftNote: "Missing key data.amount; new key data.charged_amount",
    payloadPreview:
      '{"event":"charge.success","data":{"reference":"PSK_88409","charged_amount":125000,"currency":"NGN","customer":{"email":"ada@studio.ng"}}}',
    linkedIncidentId: "inc_paystack_drift_01",
    createdAt: "2026-11-14T19:49:18Z",
  },
  {
    id: "dlv_98a1c2b7",
    projectId: "prj_paystack_01",
    status: "paused",
    httpStatus: 500,
    attemptCount: 3,
    schemaDriftNote: "Missing key data.amount; new key data.charged_amount",
    payloadPreview:
      '{"event":"charge.success","data":{"reference":"PSK_88394","charged_amount":890000,"currency":"NGN","customer":{"email":"chidi@ops.io"}}}',
    linkedIncidentId: "inc_paystack_drift_01",
    createdAt: "2026-11-14T19:46:05Z",
  },
  {
    id: "dlv_77b4e1a2",
    projectId: "prj_stripe_02",
    status: "paused",
    httpStatus: 500,
    attemptCount: 4,
    schemaDriftNote: "Field data.object.customer_email changed from string to null",
    payloadPreview:
      '{"type":"invoice.payment_failed","data":{"object":{"id":"in_1Q9x2","amount_due":4900,"customer_email":null,"attempt_count":2}}}',
    linkedIncidentId: "inc_stripe_null_02",
    createdAt: "2026-11-14T19:40:55Z",
  },
  {
    id: "dlv_65c9d2e8",
    projectId: "prj_stripe_02",
    status: "retrying",
    httpStatus: 502,
    attemptCount: 2,
    schemaDriftNote: "",
    payloadPreview:
      '{"type":"customer.subscription.updated","data":{"object":{"id":"sub_44K9","status":"active","plan":"team_monthly"}}}',
    linkedIncidentId: "",
    createdAt: "2026-11-14T19:38:20Z",
  },
  {
    id: "dlv_51f8a0c3",
    projectId: "prj_flw_03",
    status: "delivered",
    httpStatus: 200,
    attemptCount: 1,
    schemaDriftNote: "",
    payloadPreview:
      '{"event":"transfer.completed","data":{"id":904412,"amount":240000,"bank_name":"GTBank","status":"SUCCESSFUL"}}',
    linkedIncidentId: "",
    createdAt: "2026-11-14T19:35:10Z",
  },
  {
    id: "dlv_40e7b1d4",
    projectId: "prj_dispatch_04",
    status: "delivered",
    httpStatus: 200,
    attemptCount: 1,
    schemaDriftNote: "",
    payloadPreview:
      '{"event":"parcel.delivered","order_id":"ORD_5509","hub":"Yaba North","signed_by":"Bola A."}',
    linkedIncidentId: "",
    createdAt: "2026-11-14T19:31:04Z",
  },
  {
    id: "dlv_33d2c9a1",
    projectId: "prj_flw_03",
    status: "delivered",
    httpStatus: 200,
    attemptCount: 1,
    schemaDriftNote: "",
    payloadPreview:
      '{"event":"transfer.completed","data":{"id":904388,"amount":75000,"bank_name":"Kuda","status":"SUCCESSFUL"}}',
    linkedIncidentId: "",
    createdAt: "2026-11-14T19:22:47Z",
  },
];

export function computeOverview(
  projects: Project[],
  incidents: Incident[],
  deliveries: Delivery[]
): Overview {
  const openIncidents = incidents.filter(
    (inc) => inc.status === "open" || inc.status === "fixing"
  ).length;
  const pausedFromIncidents = incidents.reduce((sum, inc) => {
    if (inc.status === "resolved") return sum;
    return sum + inc.pausedCount;
  }, 0);
  const pausedFromDeliveries = deliveries.filter((d) => d.status === "paused").length;
  const pausedDeliveries = Math.max(pausedFromIncidents, pausedFromDeliveries);
  const deliveredCount = deliveries.filter((d) => d.status === "delivered").length;
  const totalBase = Math.max(1, deliveries.length + 34);
  const healthyTotal = deliveredCount + 34;
  const healthyRate = Math.min(
    100,
    Math.round(((healthyTotal - pausedDeliveries * 0.4) / totalBase) * 1000) / 10
  );

  return {
    projectsCount: projects.length,
    openIncidents,
    pausedDeliveries,
    caughtToday: deliveries.length + 142,
    healthyRate: Math.max(62, Math.min(100, healthyRate)),
  };
}

function randomHex(len: number): string {
  const chars = "0123456789abcdef";
  let out = "";
  for (let i = 0; i < len; i += 1) {
    out += chars[Math.floor(Math.random() * chars.length)];
  }
  return out;
}

export function slugifyName(name: string): string {
  return (
    name
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "") || "app"
  );
}

export function createDemoProject(input: CreateProjectInput): Project {
  const slug = input.slug.trim() || slugifyName(input.name);
  return {
    id: `prj_${randomHex(10)}`,
    name: input.name.trim(),
    slug,
    endpointKey: `ep_${randomHex(24)}`,
    sdkKey: `pk_${randomHex(32)}`,
    targetUrl: input.targetUrl.trim(),
    githubRepo: input.githubRepo.trim(),
    whatsappPhone: input.whatsappPhone.trim(),
    status: "active",
    createdAt: new Date().toISOString(),
  };
}

export function applyDemoReplayDelivery(
  deliveries: Delivery[],
  incidents: Incident[],
  deliveryId: string
): { deliveries: Delivery[]; incidents: Incident[]; updated: Delivery | null } {
  let updated: Delivery | null = null;
  let linkedIncidentId = "";

  const nextDeliveries = deliveries.map((d) => {
    if (d.id !== deliveryId) return d;
    linkedIncidentId = d.linkedIncidentId;
    updated = {
      ...d,
      status: "delivered",
      httpStatus: 200,
      attemptCount: d.attemptCount + 1,
    };
    return updated;
  });

  const nextIncidents = incidents.map((inc) => {
    if (!linkedIncidentId || inc.id !== linkedIncidentId) return inc;
    return {
      ...inc,
      pausedCount: Math.max(0, inc.pausedCount - 1),
    };
  });

  return { deliveries: nextDeliveries, incidents: nextIncidents, updated };
}

export function applyDemoIncidentAction(
  incidents: Incident[],
  deliveries: Delivery[],
  projects: Project[],
  incidentId: string,
  action: IncidentAction
): { incidents: Incident[]; deliveries: Delivery[]; updated: Incident | null } {
  let updated: Incident | null = null;
  let nextDeliveries = deliveries;

  const nextIncidents = incidents.map((inc) => {
    if (inc.id !== incidentId) return inc;
    const project = projects.find((p) => p.id === inc.projectId);
    const repo = project?.githubRepo || "Mascottz/app";
    const prNumber = 48 + (incidentId.charCodeAt(incidentId.length - 1) % 19);

    if (action === "draft_pr") {
      updated = {
        ...inc,
        status: "fixing",
        prUrl: inc.prUrl || `https://github.com/${repo}/pull/${prNumber}`,
      };
      return updated;
    }

    if (action === "replay_all") {
      nextDeliveries = deliveries.map((d) => {
        if (d.linkedIncidentId !== incidentId || d.status === "delivered") return d;
        return {
          ...d,
          status: "delivered",
          httpStatus: 200,
          attemptCount: d.attemptCount + 1,
        };
      });
      updated = {
        ...inc,
        pausedCount: 0,
      };
      return updated;
    }

    if (action === "resolve") {
      nextDeliveries = deliveries.map((d) => {
        if (d.linkedIncidentId !== incidentId || d.status === "delivered") return d;
        return {
          ...d,
          status: "delivered",
          httpStatus: 200,
          attemptCount: d.attemptCount + 1,
        };
      });
      updated = {
        ...inc,
        status: "resolved",
        pausedCount: 0,
      };
      return updated;
    }

    // mute
    updated = {
      ...inc,
      status: "muted",
    };
    return updated;
  });

  return { incidents: nextIncidents, deliveries: nextDeliveries, updated };
}

export function createSimulatedDrill(projects: Project[]): {
  incident: Incident;
  delivery: Delivery;
} {
  const targetProject = projects[2] ?? projects[0];
  const suffix = randomHex(4);
  const incidentId = `inc_drill_${suffix}`;
  const now = new Date().toISOString();

  const incident: Incident = {
    id: incidentId,
    projectId: targetProject.id,
    title: "RangeError: Invalid currency precision in settleTransfer()",
    plainSummary: `${targetProject.name} crashed in src/payouts/settle.ts on line 83. The partner sent amount as a formatted string ("240000.00") instead of an integer, causing your payout normalizer to throw. 4 events are safely held.`,
    filePath: "src/payouts/settle.ts:83",
    commitSha: "c41d9e2",
    status: "open",
    pausedCount: 4,
    prUrl: "",
    lastSeenAt: now,
  };

  const delivery: Delivery = {
    id: `dlv_drill_${suffix}`,
    projectId: targetProject.id,
    status: "paused",
    httpStatus: 500,
    attemptCount: 3,
    schemaDriftNote: "Type changed on data.amount: number -> string",
    payloadPreview:
      '{"event":"transfer.completed","data":{"id":904881,"amount":"240000.00","bank_name":"Access Bank","status":"SUCCESSFUL"}}',
    linkedIncidentId: incidentId,
    createdAt: now,
  };

  return { incident, delivery };
}
