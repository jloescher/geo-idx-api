# Growth Engineering

## When to use

Building features or flows specifically designed to increase acquisition, activation, or retention—often at the intersection of product and marketing.

## Project-relevant patterns

**Progressive domain expansion** — Start users on Pro (3 domains). When they attempt to add a 4th, trigger an upgrade modal: "Upgrade to Smart for 5 domains, or remove an existing domain." Make the constraint visible only at the point of friction.

**GHL webhook-driven re-engagement** — Use GHL's webhook events (contacts created, opportunities moved) to trigger targeted messaging. A location that hasn't created a contact in 7 days gets a "Sync your first lead" email with the widget embed code.

**GIS parcel layer as retention hook** — The parcel overlay feature (`/api/v1/gis`) increases time-on-site without increasing MLS data costs. Track map interaction metrics. Locations with high GIS usage but low listing usage are candidates for "Unlock full listings" campaigns.

## Pitfalls

Do not build viral loops that violate Stellar MLS data restrictions. Sharing listings via public URL, allowing unauthenticated previews beyond the teaser limit, or caching images outside the approved CDN path all create compliance risk. All growth tactics must respect `domain.token` auth and teaser gating.