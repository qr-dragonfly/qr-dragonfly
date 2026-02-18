# Development Environment Configuration

The application now supports separate development and production environments for all services.

## Environment Variables

### Frontend

The frontend uses three base URLs that can be configured per environment:

- **VITE_API_BASE_URL**: User service (auth, subscriptions)
- **VITE_QR_API_BASE_URL**: QR code generation service
- **VITE_CLICK_BASE_URL**: Click tracking service

### Environment Files

- **.env**: Development environment (localhost)
- **.env.production**: Production environment (Heroku)
- **.env.example**: Template with all variables

### Development Setup

Your [.env](frontend/.env) file is already configured for local development:

```bash
VITE_API_BASE_URL=http://localhost:8081    # user-service
VITE_QR_API_BASE_URL=http://localhost:8080  # qr-service
VITE_CLICK_BASE_URL=http://localhost:8082   # click-service
```

All services run via Docker Compose on these ports.

### Production Setup

Update [.env.production](frontend/.env.production) with your Heroku app URLs:

```bash
VITE_API_BASE_URL=https://your-user-service.herokuapp.com
VITE_QR_API_BASE_URL=https://your-qr-service.herokuapp.com
VITE_CLICK_BASE_URL=https://your-click-service.herokuapp.com
```

## How QR Codes Work

### Generation Flow

1. **Frontend** calls QR service (`VITE_QR_API_BASE_URL`) to create QR code
2. **QR Service** generates code and stores metadata in database
3. **QR Service** returns QR code with embedded URL pointing to click service
4. The embedded URL uses `VITE_CLICK_BASE_URL` format: `{CLICK_BASE_URL}/r/{qr-id}`

### Click Tracking Flow

1. User scans QR code
2. Request goes to **Click Service** at `{CLICK_BASE_URL}/r/{qr-id}`
3. **Click Service** records the click
4. **Click Service** calls **QR Service** to get destination URL
5. User is redirected to final destination

### Environment Separation

**Development:**

- QR codes embed: `http://localhost:8082/r/{qr-id}`
- All data stays in local Docker databases
- No cross-service pollution

**Production:**

- QR codes embed: `https://your-click-service.herokuapp.com/r/{qr-id}`
- Separate Heroku Postgres databases
- Completely isolated from dev data

## Running Locally

```bash
# Start all services
docker-compose up -d

# Start frontend dev server
cd frontend
npm run dev
```

Frontend runs on http://localhost:5173 and connects to:

- User service: http://localhost:8081
- QR service: http://localhost:8080
- Click service: http://localhost:8082

## Building for Production

```bash
cd frontend
npm run build
```

Vite automatically uses `.env.production` when building for production.

## Testing Different Environments

### Local Development

```bash
npm run dev
```

Uses `.env` (localhost URLs)

### Production Build Locally

```bash
npm run build
npm run preview
```

Uses `.env.production` (Heroku URLs)

### Override Env File

```bash
vite --mode staging
```

Would use `.env.staging` if you create one

## CORS Configuration

Make sure each backend service allows requests from the frontend:

**Development:**

```bash
CORS_ALLOW_ORIGINS=http://localhost:5173
```

**Production (via Heroku config):**

```bash
heroku config:set CORS_ALLOW_ORIGINS="https://your-frontend-app.herokuapp.com" -a your-qr-service
heroku config:set CORS_ALLOW_ORIGINS="https://your-frontend-app.herokuapp.com" -a your-click-service
heroku config:set CORS_ALLOW_ORIGINS="https://your-frontend-app.herokuapp.com" -a your-user-service
```

## Service Communication

The click service also needs to know where the QR service is:

**Development (docker-compose):**

```yaml
QR_SERVICE_BASE_URL: "http://qr-service:8080"
```

**Production (Heroku):**

```bash
heroku config:set QR_SERVICE_BASE_URL="https://your-qr-service.herokuapp.com" -a your-click-service
```

## Summary

✅ All API calls now use environment-specific URLs
✅ QR codes embed the correct click service URL for each environment
✅ Development and production data are completely separate
✅ Easy to switch between environments
✅ Production builds automatically use production URLs
