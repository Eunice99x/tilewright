import { type NextRequest, NextResponse } from "next/server";

export function proxy(req: NextRequest) {
  if (!req.cookies.get("session")?.value) {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    url.search = "";
    return NextResponse.redirect(url);
  }
  return NextResponse.next();
}

export const config = { matcher: ["/dashboard", "/dashboard/:path*"] };
