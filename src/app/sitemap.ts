import type { MetadataRoute } from "next";
import { siteUrl } from "@/lib/oauth";

export default function sitemap(): MetadataRoute.Sitemap {
  return ["", "/signup", "/terms", "/privacy", "/refunds"].map((p) => ({ url: `${siteUrl()}${p}` }));
}
