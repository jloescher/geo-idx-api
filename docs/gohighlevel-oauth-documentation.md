# GoHighLevel OAuth 2.0 API Documentation

## Overview

GoHighLevel (HighLevel) is a CRM platform with over 70,000+ agencies and 600,000+ businesses actively using the product. The Developer Marketplace allows developers to create and distribute custom applications and integrations for use with the HighLevel CRM.

The OAuth 2.0 implementation follows the Authorization Code Grant flow, providing secure, delegated access to HighLevel resources without sharing user credentials.

**Base URL:** `https://services.leadconnectorhq.com`

---

## OAuth 2.0 Authorization Flow

### Step 1: Register an OAuth App

Before integrating, you must register an application in the HighLevel Marketplace.

#### App Types

| Type | Description |
|------|-------------|
| **Private Apps** | For personal/internal use, not listed in the Marketplace |
| **Public Apps** | Visible and installable by all users once approved |

#### Configuration Steps

1. Create the app and configure the profile (logo, category, company name, description, preview images)
2. Navigate to **Advanced Settings > Auth** section
3. Configure the following:

##### Scopes

Scopes define the level of access your app will have. Request only the minimum scopes necessary for your app's functionality.

See the [Scopes Reference](#scopes-reference) section for all available scopes.

##### Redirect URL

The callback URL where the authorization server sends the authorization code after the user installs the app.

Example: `https://myapp.com/oauth/callback/highlevel`

##### Client Keys

- Click "Add" to generate a Client ID and Client Secret
- **Important:** Copy and securely store the Client Secret immediately - it cannot be viewed again after creation

---

### Step 2: App Installation Flow

1. Provide the Installation URL to the location/agency admin
2. Admin selects their location to connect
3. User is redirected to your Redirect URL with an Authorization Code

**Callback Example:**

```url
https://myapp.com/oauth/callback/highlevel?code=7676cjcbdc6t76cdcbkjcd09821jknnkj
```

---

### Step 3: Exchange Authorization Code for Access Token

Use the authorization code to obtain an access token via the token endpoint.

#### POST /oauth/token

**Endpoint:** `https://services.leadconnectorhq.com/oauth/token`

**Request:**

```bash
curl -X POST "https://services.leadconnectorhq.com/oauth/token" \
  -H "Accept: application/json" \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "665c6bb13d4e5364bdec0e2f-mawqjyjd",
    "client_secret": "74032272-7f45-4e07-8717-5e1ddbfe3de0",
    "grant_type": "authorization_code",
    "code": "363fe3f086e2db02bb9c34722902d21f76c9b217",
    "user_type": "Company",
    "redirect_uri": "https://myapp.com/oauth/callback/highlevel"
  }'
```

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `client_id` | string | Yes | Your OAuth client ID |
| `client_secret` | string | Yes | Your OAuth client secret |
| `grant_type` | string | Yes | Must be `authorization_code` |
| `code` | string | Yes | Authorization code from callback |
| `user_type` | string | Yes | `Company` (Agency) or `Location` |
| `redirect_uri` | string | Yes | Must match the registered redirect URL |

**Response:**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdXRoQ2xhc3MiOiJDb20",
  "token_type": "Bearer",
  "expires_in": 86399,
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdXRoQ2xhc3MiOiJDb21w",
  "scope": "calendars.readonly calendars/events.readonly calendars.write",
  "refreshTokenId": "685a9cc9f7434ae2fc66c31d",
  "userType": "Company",
  "companyId": "GNb7aIv4rQFVb9iwNl5K",
  "isBulkInstallation": true,
  "userId": "Rg6BRRiHh7dS9gJy3W8a"
}
```

**Response Fields:**

| Field | Description |
|-------|-------------|
| `access_token` | JWT token for API authentication (expires in ~24 hours) |
| `token_type` | Always `Bearer` |
| `expires_in` | Token lifetime in seconds (86399 ≈ 24 hours) |
| `refresh_token` | Token for obtaining new access tokens (valid 1 year) |
| `scope` | Granted scopes for this token |
| `userType` | `Company` (Agency) or `Location` |
| `companyId` | Agency/company identifier |
| `locationId` | Location identifier (Location tokens only) |
| `userId` | User who installed the app |

---

### Step 4: Refresh Access Token

Access tokens expire after approximately 24 hours. Use the refresh token to obtain a new access token without re-authorization.

#### POST /oauth/token (Refresh Flow)

**Request:**

```bash
curl --request POST \
  --url https://services.leadconnectorhq.com/oauth/token \
  --header 'Accept: application/json' \
  --header 'Content-Type: application/x-www-form-urlencoded' \
  --data client_id=665c6bb13d4e5364bdece2f-mawqjyjd \
  --data client_secret=74032272-7f45-4e7-8717-5e1ddbfe3de0 \
  --data grant_type=refresh_token \
  --data refresh_token=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.dCiY5HiwOaxk8Cz6pQ0K \
  --data user_type=Company \
  --data redirect_uri=https://myapp.com/oauth/callback/highlevel
```

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `client_id` | string | Yes | Your OAuth client ID |
| `client_secret` | string | Yes | Your OAuth client secret |
| `grant_type` | string | Yes | Must be `refresh_token` |
| `refresh_token` | string | Yes | Valid refresh token |
| `user_type` | string | Yes | `Company` or `Location` |
| `redirect_uri` | string | Yes | Your registered redirect URL |

**Token Lifecycle:**

- **Access Token:** Expires after ~24 hours (86,399 seconds)
- **Refresh Token:** Valid for 1 year or until used
- Once a refresh token is used, it becomes invalid and a new refresh token is returned
- Unused refresh tokens remain valid for 1 year

---

### Step 5: Generate Location Token from Agency Token

Convert an Agency-level access token to a Location-level token for sub-account operations.

#### POST /oauth/locationToken

**Endpoint:** `https://services.leadconnectorhq.com/oauth/locationToken`

**Request:**

```bash
curl -L "https://services.leadconnectorhq.com/oauth/locationToken" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -H "Version: 2021-07-28" \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d '{
    "companyId": "GNb7aIv4rQFV9iwNl5K",
    "locationId": "HjiMUOsCCHCjtxzEf8PR"
  }'
```

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `companyId` | string | Yes | Agency/company ID |
| `locationId` | string | Yes | Target sub-account/location ID |

**Headers:**

| Header | Description |
|--------|-------------|
| `Authorization` | Bearer token (Agency-level access token) |
| `Version` | API version (e.g., `2021-07-28`) |

**Response:**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 86400,
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "scope": "calendars.readonly calendars/events.readonly...",
  "userType": "Location",
  "companyId": "GNb7aIv4rQFVb9iNl5K",
  "locationId": "HjiMUOsCCHCjtxzEf8PR",
  "userId": "Rg6BRRiHh7dS9gy3W8a",
  "traceId": "8f712294-b015-42a-8c69-7fcb960560aa"
}
```

---

## Access Token Types

### Agency Token (userType: Company)

Used for agency-level operations:
- Creating sub-accounts
- Managing agency settings
- Bulk operations across locations

**Response Example:**

```json
{
  "userType": "Company",
  "companyId": "GNb7aIv4rQFVb9iwNl5K",
  "isBulkInstallation": true
}
```

### Location Token (userType: Location)

Used for sub-account/location-level operations:
- Contact management
- Calendar operations
- Location-specific data

**Response Example:**

```json
{
  "userType": "Location",
  "companyId": "GNb7aIv4rQFVb9iNl5K",
  "locationId": "HjiMUOsCCHCjtxzEf8PR"
}
```

---

## Webhook Events

### Configuration

1. Navigate to your app in the Marketplace dashboard
2. Click **Advanced Settings > Webhooks**
3. Enter your webhook URL
4. Toggle switches to subscribe to desired events

### App Install Webhook (Default)

This event fires whenever your app is installed. It is subscribed by default when a webhook URL is configured.

**Payload Example:**

```json
{
  "type": "INSTALL",
  "appId": "665c6bb13d4e5364bdec0e2f",
  "versionId": "665c6bb13d4e5364bdec0e2f",
  "installType": "Location",
  "locationId": "HjiMUOsCCHCjtxzEf8PR",
  "companyId": "GNb7aIv4rQFVb9iwNl5K",
  "userId": "Rg6BRRiHh7dS9gJy3W8a",
  "companyName": "Marketplace and Integrations Prod Agency",
  "isWhitelabelCompany": true,
  "whitelabelDetails": {
    "logoUrl": "https://...",
    "domain": "rajender.dentistsnear.me"
  },
  "timestamp": "2025-06-25T06:57:06.225Z",
  "webhookId": "1a533f85-1f1e-4886-891e-ee0cf4666e90"
}
```

---

## Scopes Reference

Scopes define what resources and operations your app can access. Request only the minimum scopes necessary.

### Business Scopes

| Scope | Endpoints | Access Type |
|-------|-----------|-------------|
| `businesses.readonly` | `GET /businesses`, `GET /businesses/:businessId` | Sub-Account |
| `businesses.write` | `POST /businesses`, `PUT /businesses/:businessId`, `DELETE /businesses/:businessId` | Sub-Account |

### Calendar Scopes

| Scope | Endpoints | Access Type |
|-------|-----------|-------------|
| `calendars.readonly` | `GET /calendars/`, `GET /calendars/:calendarId`, `GET /calendars/:calendarId/free-slots` | Sub-Account |
| `calendars.write` | `POST /calendars/`, `PUT /calendars/:calendarId`, `DELETE /calendars/:calendarId` | Sub-Account |
| `calendars/groups.readonly` | `GET /calendars/groups` | Sub-Account |
| `calendars/groups.write` | `POST /calendars/groups`, `DELETE /calendars/groups/:groupId`, `PUT /calendars/groups/:groupId` | Sub-Account |
| `calendars/resources.readonly` | `GET /calendars/resources/:resourceType` | Sub-Account |
| `calendars/resources.write` | `POST /calendars/resources`, `DELETE /calendars/resources/:resourceType/:id` | Sub-Account |
| `calendars/events.readonly` | `GET /calendars/events/appointments/:eventId`, `GET /calendars/events`, `GET /calendars/blocked-slots` | Sub-Account |
| `calendars/events.write` | `DELETE /calendars/events/:eventId`, `POST /calendars/events/appointments` | Sub-Account |

### Contact Scopes

| Scope | Endpoints | Webhook Events | Access Type |
|-------|-----------|----------------|-------------|
| `contacts.readonly` | `GET /contacts/:contactId`, `GET /contacts/:contactId/tasks`, `GET /contacts/:contactId/notes` | ContactCreate, ContactDelete, ContactDndUpdate, ContactTagUpdate, NoteCreate, NoteDelete, TaskCreate, TaskDelete | Sub-Account |
| `contacts.write` | `POST /contacts/`, `PUT /contacts/:contactId`, `DELETE /contacts/:contactId`, `POST /contacts/:contactId/tags`, `POST /contacts/:contactId/notes` | | Sub-Account |

### Conversation Scopes

| Scope | Endpoints | Webhook Events | Access Type |
|-------|-----------|----------------|-------------|
| `conversations.readonly` | `GET /conversations/:conversationsId`, `GET /conversations/search` | ConversationUnreadWebhook | Sub-Account |
| `conversations.write` | `POST /conversations/`, `PUT /conversations/:conversationsId`, `DELETE /conversations/:conversationsId` | | Sub-Account |
| `conversations/message.readonly` | `GET conversations/messages/:messageId/locations/:locationId/recording`, `GET conversations/locations/:locationId/messages/:messageId/transcription` | InboundMessage, OutboundMessage | Sub-Account |
| `conversations/message.write` | `POST /conversations/messages`, `POST /conversations/messages/inbound`, `PUT /conversations/messages/:messageId/status` | ConversationProviderOutboundMessage | Sub-Account |

### Invoice Scopes

| Scope | Endpoints | Access Type |
|-------|-----------|-------------|
| `invoices.readonly` | `GET /invoices/`, `GET /invoices/:invoiceId`, `GET /invoices/generate-invoice-number` | Sub-Account |
| `invoices.write` | `POST /invoices`, `PUT /invoices/:invoiceId`, `DELETE /invoices/:invoiceId`, `POST /invoices/:invoiceId/send`, `POST /invoices/:invoiceId/void` | Sub-Account |
| `invoices/schedule.readonly` | `GET /invoices/schedule/`, `GET /invoices/schedule/:scheduleId` | Sub-Account |
| `invoices/schedule.write` | `POST /invoices/schedule`, `PUT /invoices/schedule/:scheduleId`, `DELETE /invoices/schedule/:scheduleId` | Sub-Account |
| `invoices/template.readonly` | `GET /invoices/template/`, `GET /invoices/template/:templateId` | Sub-Account |
| `invoices/template.write` | `POST /invoices/template/`, `PUT /invoices/template/:templateId`, `DELETE /invoices/template/:templateId` | Sub-Account |
| `invoices/estimate.readonly` | `GET /invoices/estimate/number/generate`, `GET /invoices/estimate/list` | Sub-Account |
| `invoices/estimate.write` | `POST /invoices/estimate`, `POST /invoices/estimate/:estimateId/send`, `DELETE /invoices/estimate/:estimateId` | Sub-Account |

### Location Scopes

| Scope | Endpoints | Webhook Events | Access Type |
|-------|-----------|----------------|-------------|
| `locations.readonly` | `GET /locations/:locationId`, `GET /locations/search`, `GET /locations/timeZones` | LocationCreate, LocationUpdate | Sub-Account, Agency |
| `locations.write` | `POST /locations/`, `PUT /locations/:locationId`, `DELETE /locations/:locationId` | | Agency |
| `locations/customValues.readonly` | `GET /locations/:locationId/customValues` | | Sub-Account |
| `locations/customValues.write` | `POST /locations/:locationId/customValues`, `PUT /locations/:locationId/customValues/:id` | | Sub-Account |
| `locations/customFields.readonly` | `GET /locations/:locationId/customFields` | | Sub-Account |
| `locations/customFields.write` | `POST /locations/:locationId/customFields`, `DELETE /locations/:locationId/customFields/:id` | | Sub-Account |
| `locations/tags.readonly` | `GET /locations/:locationId/tags` | | Sub-Account |
| `locations/tags.write` | `POST /locations/:locationId/tags/`, `DELETE /locations/:locationId/tags/:tagId` | | Sub-Account |

### Opportunity Scopes

| Scope | Endpoints | Webhook Events | Access Type |
|-------|-----------|----------------|-------------|
| `opportunities.readonly` | `GET /opportunities/search`, `GET /opportunities/:id`, `GET /opportunities/pipelines` | OpportunityCreate, OpportunityDelete, OpportunityStageUpdate, OpportunityStatusUpdate, OpportunityMonetaryValueUpdate | Sub-Account |
| `opportunities.write` | `DELETE /opportunities/:id`, `PUT /opportunities/:id/status`, `POST /opportunities`, `PUT /opportunities/:id` | | Sub-Account |

### Payment Scopes

| Scope | Endpoints | Access Type |
|-------|-----------|-------------|
| `payments/integration.readonly` | `GET /payments/integrations/provider/whitelabel` | Sub-Account |
| `payments/integration.write` | `POST /payments/integrations/provider/whitelabel` | Sub-Account |
| `payments/orders.readonly` | `GET /payments/orders/`, `GET /payments/orders/:orderId` | Sub-Account |
| `payments/orders.write` | `POST /payments/orders/:orderId/fulfillments` | Sub-Account |
| `payments/transactions.readonly` | `GET /payments/transactions/`, `GET /payments/transactions/:transactionId` | Sub-Account |
| `payments/subscriptions.readonly` | `GET /payments/subscriptions/`, `GET /payments/subscriptions/:subscriptionId` | Sub-Account |
| `payments/coupons.readonly` | `GET /payments/coupon/list`, `GET /payments/coupon` | Sub-Account |
| `payments/coupons.write` | `POST /payments/coupon`, `PUT /payments/coupon`, `DELETE /payments/coupon` | Sub-Account |
| `payments/custom-provider.readonly` | `GET /payments/custom-provider/connect` | Sub-Account |
| `payments/custom-provider.write` | `POST /payments/custom-provider/provider`, `POST /payments/custom-provider/connect`, `DELETE /payments/custom-provider/provider` | Sub-Account |

### Product Scopes

| Scope | Endpoints | Access Type |
|-------|-----------|-------------|
| `products.readonly` | `GET /products/`, `GET /products/:productId`, `GET /products/reviews` | Sub-Account |
| `products.write` | `POST /products/`, `PUT /products/:productId`, `DELETE /products/:productId`, `POST /products/bulk-update` | Sub-Account |
| `products/prices.readonly` | `GET /products/:productId/price/` | Sub-Account |
| `products/prices.write` | `POST /products/:productId/price/`, `DELETE /products/:productId/price/:priceId` | Sub-Account |
| `products/collection.readonly` | `GET /products/collections` | Sub-Account |
| `products/collection.write` | `POST /products/collections`, `DELETE /products/collections/:collectionId` | Sub-Account |

### OAuth & Agency Scopes

| Scope | Endpoints | Access Type |
|-------|-----------|-------------|
| `oauth.readonly` | `GET /oauth/installedLocations` | Agency |
| `oauth.write` | `POST /oauth/locationToken` | Agency |
| `companies.readonly` | `GET /companies/:companyId` | Agency |
| `snapshots.readonly` | `GET /snapshots`, `GET /snapshots/snapshot-status/:snapshotId` | Agency |
| `snapshots.write` | `POST /snapshots/share/link` | Agency |

### Social Media Planner Scopes

| Scope | Endpoints | Access Type |
|-------|-----------|-------------|
| `socialplanner/account.readonly` | `GET /social-media-posting/:locationId/accounts` | Sub-Account |
| `socialplanner/account.write` | `DELETE /social-media-posting/:locationId/accounts/:id` | Sub-Account |
| `socialplanner/post.readonly` | `GET /social-media-posting/:locationId/posts/:id` | Sub-Account |
| `socialplanner/post.write` | `POST /social-media-posting/:locationId/posts`, `DELETE /social-media-posting/:locationId/posts/:id` | Sub-Account |

### User Scopes

| Scope | Endpoints | Access Type |
|-------|-----------|-------------|
| `users.readonly` | `GET /users/`, `GET /users/:userId` | Sub-Account, Agency |
| `users.write` | `POST /users/`, `DELETE /users/:userId`, `PUT /users/:userId` | Sub-Account, Agency |

### Voice AI Scopes

| Scope | Endpoints | Webhook Events | Access Type |
|-------|-----------|----------------|-------------|
| `voice-ai-dashboard.readonly` | `GET /voice-ai/dashboard/call-logs`, `GET /voice-ai/dashboard/call-logs/:callId` | VoiceAiCallEnd | Sub-Account |
| `voice-ai-agents.readonly` | `GET /voice-ai/agents`, `GET /voice-ai/agents/:agentId` | | Sub-Account |
| `voice-ai-agents.write` | `POST /voice-ai/agents`, `DELETE /voice-ai/agents/:agentId` | | Sub-Account |

### Additional Scopes

| Scope | Endpoints | Access Type |
|-------|-----------|-------------|
| `campaigns.readonly` | `GET /campaigns/` | Sub-Account |
| `forms.readonly` | `GET /forms/`, `GET /forms/submissions` | Sub-Account |
| `forms.write` | `POST /forms/upload-custom-files` | Sub-Account |
| `links.readonly` | `GET /links/` | Sub-Account |
| `links.write` | `POST /links/`, `DELETE /links/:linkId` | Sub-Account |
| `medias.readonly` | `GET /medias/files` | Sub-Account |
| `medias.write` | `POST /medias/upload-file`, `DELETE /medias/:fileId` | Sub-Account |
| `surveys.readonly` | `GET /surveys/`, `GET /surveys/submissions` | Sub-Account |
| `workflows.readonly` | `GET /workflows/` | Sub-Account |
| `funnels/redirect.readonly` | `GET /funnels/lookup/redirect/list` | Sub-Account |
| `funnels/redirect.write` | `POST /funnels/lookup/redirect`, `DELETE /funnels/lookup/redirect/:id` | Sub-Account |
| `funnels/page.readonly` | `GET /funnels/page` | Sub-Account |
| `funnels/funnel.readonly` | `GET /funnels/funnel/list` | Sub-Account |
| `blogs/post.write` | `POST /blogs/posts` | Sub-Account |
| `blogs/post-update.write` | `PUT /blogs/posts/:postId` | Sub-Account |
| `blogs/category.readonly` | `GET /blogs/categories` | Sub-Account |
| `blogs/author.readonly` | `GET /blogs/authors` | Sub-Account |
| `associations.readonly` | `GET /associations/key/:key_name`, `GET /associations/` | Sub-Account |
| `associations.write` | `POST /associations/`, `DELETE /associations/:associationId` | Sub-Account |
| `phonenumbers.read` | `GET /phone-system/numbers/location/:locationId` | Sub-Account |
| `numberpools.read` | `GET /phone-system/number-pools` | Sub-Account |

---

## Using Access Tokens

Include the access token in the Authorization header for all API requests:

```bash
curl -X GET "https://services.leadconnectorhq.com/v2/contacts" \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Accept: application/json"
```

---

## Error Responses

| Status Code | Description |
|-------------|-------------|
| `200` | Successful response |
| `400` | Bad request - invalid parameters |
| `401` | Unauthorized - invalid or expired token |
| `422` | Unprocessable entity - validation error |

---

## Best Practices

1. **Store tokens securely** - Access tokens and refresh tokens should be encrypted and stored securely
2. **Implement token refresh logic** - Automatically refresh tokens before expiration
3. **Request minimal scopes** - Only request the scopes your application actually needs
4. **Handle webhook events** - Process the INSTALL webhook to capture installation details
5. **Use appropriate token type** - Use Location tokens for sub-account operations, Agency tokens for agency-level operations
6. **Secure client secrets** - Never expose client secrets in client-side code

---

## Additional Resources

- Official Documentation: [https://marketplace.gohighlevel.com/docs](https://marketplace.gohighlevel.com/docs)
- OAuth 2.0 Reference: [https://marketplace.gohighlevel.com/docs/Authorization/OAuth2.0](https://marketplace.gohighlevel.com/docs/Authorization/OAuth2.0)
- Scopes Documentation: [https://marketplace.gohighlevel.com/docs/Authorization/Scopes](https://marketplace.gohighlevel.com/docs/Authorization/Scopes)
- Webhook Events: [https://marketplace.gohighlevel.com/docs/category/webhook](https://marketplace.gohighlevel.com/docs/category/webhook)

---

## Notes

- OAuth 2.0 is recommended for marketplace apps, multi-location access, and webhook integrations
- Access tokens should be refreshed before expiration (24-hour window)
- Refresh tokens are single-use - each refresh returns a new refresh token
- Agency tokens can generate Location tokens for sub-account operations
- Webhooks require a publicly accessible URL