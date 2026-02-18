# Three-Tier Environment Setup

This project supports three separate environments:

## 🖥️ Local (Your Machine)

**Purpose:** Development on your laptop using Docker

**Services:**

- Frontend: http://localhost:5173
- User Service: http://localhost:8081
- QR Service: http://localhost:8080
- Click Service: http://localhost:8082

**Data:** Local Docker PostgreSQL databases

**How to run:**

```bash
# Start backend services
docker-compose up -d

# Start frontend
cd frontend
npm run dev
```

**Environment:** Uses [.env](frontend/.env)

**QR codes generated:** Embed `http://localhost:8082/r/{id}`

---

## 🚧 Dev/Staging (Deployed for Testing)

**Purpose:** Deployed environment for testing before production

**Services:**

- All services deployed to Heroku (separate apps from production)
- Example: `your-dev-qr-service.herokuapp.com`

**Data:** Heroku PostgreSQL databases (shared across dev deployments)

**How to run:**

```bash
cd frontend
npm run dev:staging    # Local frontend → Dev backend
npm run build:staging  # Build for dev deployment
```

**Environment:** Uses [.env.staging](frontend/.env.staging)

**QR codes generated:** Embed `https://your-dev-click-service.herokuapp.com/r/{id}`

**Setup:**

1. Create separate Heroku apps with "-dev" suffix:

   ```bash
   heroku create your-app-qr-service-dev
   heroku create your-app-click-service-dev
   heroku create your-app-user-service-dev
   heroku create your-app-frontend-dev
   ```

2. Update `.env.staging` with dev app URLs

3. Can work with dev backend from your local machine:
   ```bash
   npm run dev:staging
   ```

---

## 🚀 Production (Live)

**Purpose:** Customer-facing production environment

**Services:**

- All services deployed to Heroku (separate apps)
- Example: `your-qr-service.herokuapp.com`

**Data:** Heroku PostgreSQL databases (completely isolated)

**How to run:**

```bash
cd frontend
npm run build          # Production build
npm run preview        # Test production build locally
```

**Environment:** Uses [.env.production](frontend/.env.production)

**QR codes generated:** Embed `https://your-click-service.herokuapp.com/r/{id}`

---

## Environment File Summary

| File              | Environment | Services         | Data                   |
| ----------------- | ----------- | ---------------- | ---------------------- |
| `.env`            | Local       | localhost:PORT   | Docker Postgres        |
| `.env.staging`    | Dev/Staging | heroku-dev apps  | Heroku Postgres (dev)  |
| `.env.production` | Production  | heroku prod apps | Heroku Postgres (prod) |

## NPM Scripts

### Local Development

```bash
npm run dev              # Local frontend → Local Docker backend
```

### Dev/Staging

```bash
npm run dev:staging      # Local frontend → Dev Heroku backend
npm run build:staging    # Build frontend for dev deployment
npm run preview:staging  # Test staging build locally
```

### Production

```bash
npm run build            # Build frontend for production
npm run preview          # Test production build locally
```

## Common Workflows

### Working Locally

```bash
docker-compose up -d
cd frontend && npm run dev
```

Everything runs on your machine. QR codes work only on your machine.

### Testing Against Dev Environment

```bash
# Backend already deployed to dev Heroku apps
cd frontend && npm run dev:staging
```

Frontend runs locally but talks to dev backend. QR codes work publicly via dev click service.

### Deploying to Dev

```bash
# Deploy backend services to dev apps (via GitHub Actions or manual)
# Deploy frontend to dev app
npm run build:staging
# Push to dev frontend Heroku app
```

### Deploying to Production

```bash
# Push to main branch → GitHub Actions deploy to prod apps
# Or manually deploy with production config
```

## Data Isolation

✅ **Local:** Uses Docker volumes - completely isolated
✅ **Dev/Staging:** Uses separate Heroku Postgres databases - can share with team
✅ **Production:** Uses separate Heroku Postgres databases - completely isolated from dev

## QR Code Isolation

When you generate QR codes, they embed the click service URL:

- **Local:** `http://localhost:8082/r/{id}` - only works on your machine
- **Dev:** `https://dev-click.herokuapp.com/r/{id}` - works publicly, goes to dev database
- **Prod:** `https://click.herokuapp.com/r/{id}` - works publicly, goes to prod database

**Important:** QR codes from dev won't mix with prod data because they point to different click services!

## Heroku Configuration

### Dev Apps

```bash
heroku config:set CORS_ALLOW_ORIGINS="https://your-frontend-dev.herokuapp.com" -a your-qr-service-dev
heroku config:set DATABASE_URL="..." -a your-qr-service-dev
# etc.
```

### Prod Apps

```bash
heroku config:set CORS_ALLOW_ORIGINS="https://your-frontend.herokuapp.com" -a your-qr-service
heroku config:set DATABASE_URL="..." -a your-qr-service
# etc.
```

## GitHub Actions

Update your workflows to deploy to the right apps based on branch:

- **main branch** → production apps
- **develop branch** → dev apps (if you set this up)
- **manual workflow** → specify environment

## Best Practices

1. ✅ Always test on **local** first
2. ✅ Deploy to **dev/staging** for team testing
3. ✅ Only deploy to **prod** after dev testing passes
4. ✅ Never share credentials between environments
5. ✅ Use separate Stripe accounts for dev/prod
6. ✅ Use separate AWS Cognito pools for dev/prod
