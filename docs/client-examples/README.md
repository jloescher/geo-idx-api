# Next.js client request examples

These files are **generated** when you run production smoke tests with a valid PAT:

```bash
export YAAK_BEARER_TOKEN='your-production-pat'
export YAAK_DOMAIN_SLUG='your-verified-domain.com'
make test-api-smoke
```

Each JSON file contains:

- `method`, `url`, `headers` — the resolved request against production
- `body` — POST payload (when applicable)
- `nextjs_fetch_example.snippet` — copy into a Next.js Server Action or Route Handler

## Environment variables for Next.js

| Variable | Purpose |
|----------|---------|
| `IDX_API_TOKEN` | Bearer PAT (`idx:full` or `idx:access`) |
| `IDX_DOMAIN_SLUG` | Verified domain slug (`X-Domain-Slug`) |
| `IDX_API_BASE_URL` | e.g. `https://idx.quantyralabs.cc` |

Never expose the PAT to the browser. Call the IDX API from server-side code only.

## Example Server Action pattern

```typescript
'use server';

export async function searchListings() {
  const base = process.env.IDX_API_BASE_URL!;
  const res = await fetch(`${base}/api/v1/search?dataset=stellar`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${process.env.IDX_API_TOKEN}`,
      'X-Domain-Slug': process.env.IDX_DOMAIN_SLUG!,
    },
    body: JSON.stringify({
      city: 'Largo',
      statuses: ['Active'],
      page: { limit: 24, skip: 0 },
    }),
  });
  if (!res.ok) {
    throw new Error(`IDX search failed: ${res.status}`);
  }
  return res.json();
}
```

See also [`docs/yaak-endpoint-smoke-test.md`](../yaak-endpoint-smoke-test.md) and the OpenAPI spec at [`docs/yaak-api-collection.json`](../yaak-api-collection.json).
