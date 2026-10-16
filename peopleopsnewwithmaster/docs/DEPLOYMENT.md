# Deployment notes

The provided Docker Compose deployment builds the Go API and Nginx-hosted SPA as separate services. Only the web container is published to the host. The API and PostgreSQL data remain on the private Compose network/volume.

For a real deployment:

1. Use a managed secret store or injected environment secret, not a checked-in `.env`.
2. Place a TLS-terminating reverse proxy/load balancer in front of the web service; configure forwarded headers and the public origin carefully.
3. Set the API's `CORS_ORIGIN` to the canonical origin. If the app is served only through Nginx on one origin, browser API requests are same-origin.
4. Attach durable encrypted storage for `/data` and the future private document store. Schedule PostgreSQL-consistent backups and verify restores.
5. Build immutable images in CI, scan dependencies and images, run tests, deploy to staging, and run smoke tests before promotion.
6. Keep PostgreSQL on local persistent storage. Do not put the live database on an arbitrary network filesystem. Move to PostgreSQL when concurrent writes or multiple API replicas are needed.
