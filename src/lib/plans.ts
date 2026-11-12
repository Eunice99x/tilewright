import app from "../../api/internal/service/app.json";

export const APP = { name: app.name, product: app.product, support_email: app.support_email };

export interface PlanInfo {
  id: string;
  name: string;
  price: number;
  billing?: "subscription" | "one_time";
  features: string[];
}

export const PLANS: PlanInfo[] = app.plans as PlanInfo[];

export const priceNote = (p: PlanInfo) => (p.billing === "one_time" ? "once" : "/ month");

export const planName = (id: string) => (id === "admin" ? "Admin" : (PLANS.find((p) => p.id === id)?.name ?? id));
