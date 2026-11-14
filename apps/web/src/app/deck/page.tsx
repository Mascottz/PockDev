import { ControlDeck } from "@/components/ControlDeck";

export const metadata = {
  title: "PockDev Control Deck: Orbital Constellation and Webhook River",
  description:
    "Watch live webhook deliveries, inspect schema drift, draft fix PRs, and replay paused events right from PockDev.",
};

export default function DeckPage() {
  return <ControlDeck />;
}
