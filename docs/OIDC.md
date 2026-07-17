# OIDC Single Sign-On

SBV can authenticate against any OIDC-compliant identity provider (Authentik, tinyauth, Pocket ID, Keycloak, ...) using the standard authorization code flow.

## Setup

1. Register SBV as a confidential client (web application) with your identity provider, using the redirect URI:
   ```
   https://your-sbv-host/api/auth/oidc/callback
   ```
2. Set the environment variables below and restart SBV.
3. The login page will show a "Sign in with ..." button alongside the password form.

## Environment Variables

- `OIDC_ISSUER_URL` - Issuer URL used for discovery, e.g. `https://auth.example.com/application/o/sbv/` (required; OIDC is disabled when unset)
- `OIDC_CLIENT_ID` - Client ID registered with the provider (required)
- `OIDC_CLIENT_SECRET` - Client secret (required)
- `OIDC_PROVIDER_NAME` - Label for the login button, e.g. `Authentik` (default: `SSO`)
- `OIDC_REDIRECT_URL` - Callback URL; derived from the request (honoring `X-Forwarded-Proto`/`Host`) when unset
- `OIDC_SCOPES` - Space-separated scopes (default: `openid profile email`)
- `OIDC_USERNAME_CLAIM` - ID token claim used as the SBV username (default: `preferred_username`, falling back to `email`, then `sub`)

## Docker Compose Example

```yaml
services:
  sbv:
    image: ghcr.io/lowcarbdev/sbv:stable
    ports:
      - "8081:8081"
    volumes:
      - ./data:/data
    environment:
      - PUID=1000
      - PGID=1000
      - OIDC_ISSUER_URL=https://auth.example.com/application/o/sbv/
      - OIDC_CLIENT_ID=sbv
      - OIDC_CLIENT_SECRET=your-client-secret
      - OIDC_PROVIDER_NAME=Authentik
    restart: unless-stopped
```

## User Matching and Provisioning

Users are matched by username. First-time OIDC logins create a new SBV account automatically unless `DISABLE_REGISTRATION=true`, in which case only usernames that already exist in SBV can sign in.

Accounts created via OIDC have no password and can only log in through the identity provider.

To let an existing password-based user sign in via OIDC, make sure the identity provider sends a username claim that exactly matches their SBV username (see `OIDC_USERNAME_CLAIM`).

## Notes

- Provider discovery happens lazily on first login, so SBV starts even when the identity provider is temporarily unreachable.
- Signing out of SBV ends only the SBV session, not the identity provider session (no RP-initiated logout). Clicking the SSO button again will typically log straight back in.
- tinyauth: recent versions can act as an OIDC provider. If your tinyauth deployment runs in forward-auth mode only (trusted `Remote-User` headers), that is not covered by this feature.
