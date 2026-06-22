# Cloudflare Worker proxy for new-api

This worker accepts POST requests from new-api and performs outbound fetches on its behalf.

It also exposes a simple health check endpoint:

- `GET /health` → returns JSON status

## Files

- `worker.js` - worker entrypoint
- `wrangler.toml` - Wrangler config
- `package.json` - local deploy scripts

## Required secret

Set this in Cloudflare before deployment:

- `WORKER_ACCESS_KEY` - shared secret used by new-api

## Optional vars

- `ALLOW_HTTP` - set this to `true` only if you must proxy plain HTTP URLs; if unset, HTTP stays disabled
- `ALLOWED_HOSTS=` - comma-separated base domains. Each domain allows itself and all subdomains. Example: `example.com,telegram.org`

## CLI deployment

1. Install dependencies:

   ```bash
   cd cloudflare-worker
   npm install
   ```

2. Authenticate Wrangler with Cloudflare:

   ```bash
   npx wrangler login
   ```

3. Set the shared secret used by new-api:

   ```bash
   npx wrangler secret put WORKER_ACCESS_KEY
   ```

4. Optional: edit `wrangler.toml` and set `ALLOWED_HOSTS` to the base domains you want to allow.

   Example:

   ```toml
   [vars]
   ALLOWED_HOSTS = "example.com,telegram.org"
   ```

5. Optional: if you must allow plain HTTP upstream URLs, deploy with `ALLOW_HTTP=true`.

   ```bash
   npx wrangler deploy --var ALLOW_HTTP:true
   ```

   If you do not set `ALLOW_HTTP=true`, HTTP stays disabled.

6. Otherwise deploy normally:

   ```bash
   npx wrangler deploy
   ```

Then set the following in new-api system settings:

- Worker URL: your deployed worker URL
- Worker Access Key: same value as `WORKER_ACCESS_KEY`

You can test the worker in a browser with:

```text
https://api-worker-proxy.your-subdomain.workers.dev/health
```

## Cloudflare dashboard deployment

You can also deploy from the Cloudflare website by creating a Worker and pasting `worker.js` into the editor.
Create:

- Secret: `WORKER_ACCESS_KEY`
- Variable: `ALLOWED_HOSTS` = optional comma-separated base domains

Only add `ALLOW_HTTP=true` if you explicitly need HTTP support.

Examples:

- `example.com` allows `example.com`, `files.example.com`, and `a.b.example.com`
- `telegram.org` allows `telegram.org` and `api.telegram.org`

If you use the dashboard route, `wrangler.toml` and `package.json` are not required by Cloudflare, but they are kept here for CLI deployment and version control.
