import { PricingView } from "@/components/PricingView";

export const metadata = {
  title: "PockDev Pricing: Free Shield and Pro Orbit Plans",
  description:
    "Start free with 2 watched apps and 5,000 monthly webhook events, or upgrade to Pro Orbit with Paystack for 20 apps, 100,000 monthly events, and 30-day retention.",
};

export default function PricingPage() {
  return <PricingView />;
}
