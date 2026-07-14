import type { MetadataRoute } from "next";

export default function robots(): MetadataRoute.Robots {
  const baseUrl = "https://www.multica.ai";

  return {
    rules: {
      userAgent: "*",
      allow: "/docs/",
      disallow: "/",
    },
    sitemap: `${baseUrl}/docs/sitemap.xml`,
  };
}
