import assert from "node:assert/strict";
import { afterEach, test } from "node:test";
import { authorizeUrl, enabledProviders, redirectUri, sameState } from "./oauth.ts";

const NAMES = ["SITE_URL", "GOOGLE_CLIENT_ID", "GITHUB_CLIENT_ID"];
afterEach(() => NAMES.forEach((n) => delete process.env[n]));

test("state must match exactly and be present on both sides", () => {
  assert.equal(sameState("abc123", "abc123"), true);
  assert.equal(sameState("abc123", "abc124"), false);
  assert.equal(sameState("abc", "abc123"), false);
  assert.equal(sameState("abc123", undefined), false);
  assert.equal(sameState(null, "abc123"), false);
  assert.equal(sameState("", ""), false);
  assert.equal(sameState(undefined, undefined), false);
});

test("redirect URI comes from SITE_URL, defaulting to local dev", () => {
  assert.equal(redirectUri("github"), "http://localhost:3100/auth/github/callback");
  process.env.SITE_URL = "https://app.example/";
  assert.equal(redirectUri("google"), "https://app.example/auth/google/callback");
});

test("authorize URLs ask for the right scopes", () => {
  const gh = authorizeUrl("github", "id1", "st").searchParams;
  assert.equal(gh.get("scope"), "user:email");
  assert.equal(gh.get("client_id"), "id1");
  assert.equal(gh.get("state"), "st");
  const g = authorizeUrl("google", "id2", "st").searchParams;
  assert.equal(g.get("scope"), "openid email");
  assert.equal(g.get("response_type"), "code");
  assert.equal(g.get("redirect_uri"), "http://localhost:3100/auth/google/callback");
});

test("only providers with a client id are enabled, GitHub first", () => {
  assert.deepEqual(enabledProviders(), []);
  process.env.GOOGLE_CLIENT_ID = "x";
  assert.deepEqual(enabledProviders(), ["google"]);
  process.env.GITHUB_CLIENT_ID = "y";
  assert.deepEqual(enabledProviders(), ["github", "google"]);
});
