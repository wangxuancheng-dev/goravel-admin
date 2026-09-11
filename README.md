<p align="center"><img src="https://www.goravel.dev/logo.png?v=1.14.x" width="300"></p>

English | [中文](./README_zh.md)

## About Goravel

Goravel is a web application framework with complete functions and good scalability. As a starting scaffolding to help Gopher quickly build their own applications.

The framework style is consistent with [Laravel](https://github.com/laravel/laravel), let Phper don't need to learn a new framework, but also happy to play around Golang! Tribute Laravel!

Welcome to star, PR and issues！

## Admin System

This project includes a complete admin management system built with Goravel framework.

```bash
git clone https://github.com/wangxuancheng-dev/goravel-admin.git
```

> Demo https://admin.xuancheng888.top 

username: demo  
password: demo123

### Intended Use

**Good fit:** internal admin panels, ops backends, Goravel + Vue / React starter for secondary development.

**Not a drop-in fit:** financial trading cores or large-scale commercial SaaS platforms out of the box. Sharding / Elasticsearch / multi-queue drivers are **optional** and need extra ops.

See module tiers and production configs: [docs/OPENSOURCE.md](./docs/OPENSOURCE.md).

### Screenshots

<p align="center">
  <img src="./images/login.png" alt="Login Page" width="800">
  <p align="center">Login Page</p>
</p>

<p align="center">
  <img src="./images/admin.png" alt="Admin Dashboard" width="800">
  <p align="center">Admin Dashboard</p>
</p>

<p align="center">
  <img src="./images/react.jpg" alt="React Admin Dashboard" width="800">
  <p align="center">React Admin Dashboard</p>
</p>

<p align="center">
  <img src="./images/generator.png" alt="Code Generator" width="800">
  <p align="center">Code Generator</p>
</p>

<p align="center">
  <img src="./images/monitor.png" alt="System Monitoring" width="800">
  <p align="center">System Monitoring</p>
</p>

<p align="center">
  <img src="./images/ai.png" alt="AI code generation" width="800">
  <p align="center">AI code generation</p>
</p>

<p align="center">
  <img src="./images/ai-lab.png" alt="AI Lab" width="800">
  <p align="center">AI Lab (text, vision, image, audio; demo-ready with per-admin rate limits)</p>
</p>


<p align="center">
  <img src="./images/pages.png" alt="cloudflare" width="800">
  <p align="center">cloudflare</p>
</p>

### Features

#### Core Modules
- **Authentication & Authorization**
  - JWT-based authentication
  - Role-based access control (RBAC)
  - Permission management
  - Multi-token management
  - Online admin monitoring and kick-out

- **Admin Management**
  - Admin user management
  - Department management
  - Role management
  - Permission assignment
  - Password reset

- **System Configuration**
  - Menu management (dynamic menu)
  - Dictionary management
  - System configuration
  - Blacklist management

- **Logging & Monitoring**
  - Operation logs (with automatic recording)
  - Login logs
  - System logs (with trace ID)
  - Service monitoring

- **Additional Features**
  - Dashboard with statistics
  - Notification center (WebSocket real-time notifications)
  - Data export management
  - **AI Lab** (top-level menu; Goravel AI SDK demos: text chat, vision, image gen, TTS, STT; enabled when `AI_API_KEY` is set, per-admin rate limits)
  - Multi-language support (Chinese/English)
  - Responsive UI design

#### Optional Advanced Modules

Enable only when needed (see [OPENSOURCE.md](./docs/OPENSOURCE.md)):

- Monthly order/payment sharding and balance-log hash sharding
- Elasticsearch order sync / search
- Redis async queues, long-running export jobs, extra queue drivers
- OpenTelemetry export to Jaeger / Grafana
- Payment admin sample (**not a full acquiring stack**: no production notify/refund — see [OPENSOURCE.md](./docs/OPENSOURCE.md))

### Tech Stack

**Backend:**
- Goravel Framework (Go)
- RBAC Permission System
- WebSocket Support
- Database Migrations & Seeders

**Frontend (pick one):**

| | Vue (`html/`) | React (`html-react/`) |
|---|---|---|
| UI | Element Plus + vxe-table | Ant Design 6 |
| State | Pinia | Zustand |
| Router | Vue Router | React Router 7 |
| i18n | vue-i18n | react-i18next |
| Shared | Vite, Axios, ECharts, same Admin API | Vite, Axios, ECharts, same Admin API |

Both frontends talk to `/api/admin` and **must ship new admin features together**. React covers the main modules (including code generator and AI Lab). Details: [html-react/README.md](./html-react/README.md).

### Three-minute Docker (recommended)

```bash
cp .env.docker.example .env
docker compose up -d --build
# When healthy: http://localhost:3000
# Default login: admin / admin123 (change ASAP)
```

For UI development, run Vite separately (Vue `html/` → `:3007`, React `html-react/` → `:3008`) against `http://127.0.0.1:3000`.

### Quick Start (local Go + your own database)

1. **Backend Setup:**
   ```bash
   # Install dependencies
   go mod tidy
   
   # Configure database in .env
   # Run migrations and seeders
   go run . artisan migrate
   go run . artisan db:seed
   
   # Start server
   go run . --no-ansi
   # or use air for live reload
   air
   ```

2. **Frontend Setup (Vue):**
   ```bash
   cd html
   
   # Install dependencies
   npm install
   
   # Configure API address in .env
   # VITE_API_BASE_URL=http://127.0.0.1:3000
   # VITE_API_PREFIX=/api/admin
   
   # Start development server (default http://localhost:3007)
   npm run dev
   ```

3. **Frontend Setup (React, optional):**
   ```bash
   cd html-react
   cp .env.example .env
   npm install
   npm run dev
   ```
   Dev server defaults to `http://localhost:3008`. Allow that origin in root `.env` CORS (and `Accept-Language`), then restart the Go backend. See [html-react/README.md](./html-react/README.md).

4. **Default Login:**
   - Username: `admin`
   - Password: `admin123`
   - (Please change the default password after first login)

### Build & Deployment

For detailed build and deployment instructions, including cross-platform compilation, Docker deployment, and systemd service setup, please refer to [BUILD.md](./docs/BUILD.md).

### API Documentation

The admin API endpoints are prefixed with `/api/admin`. All endpoints require JWT authentication except login and captcha.

For detailed API documentation, see [routes/admin.go](./routes/admin.go)

#### Swagger API Documentation

The project includes Swagger API documentation for interactive API exploration.

Error code overview for frontend integration: [docs/ERROR_CODES.md](./docs/ERROR_CODES.md)

**Access Swagger Documentation:**

The Swagger JSON document is available at:
- Local development: `http://localhost:3000/swagger/index.html`
- Production: `https://your-domain.com/swagger/index.html`

**Regenerate Swagger Documentation:**

After modifying API routes or adding new endpoints, regenerate the Swagger documentation:

```bash
# Generate Swagger documentation
swag init
```

This will regenerate the `docs/docs.go`, `docs/swagger.json`, and `docs/swagger.yaml` files based on the Swagger annotations in your code (see `main.go` for example annotations).

### Project Structure

```
.
├── app/
│   ├── http/
│   │   ├── controllers/admin/    # Admin controllers
│   │   ├── middleware/           # Custom middleware (JWT, Permission, OperationLog)
│   │   └── helpers/              # Helper functions
│   ├── models/                   # Database models
│   └── services/                 # Business logic services
├── routes/
│   └── admin.go                  # Admin routes
├── database/
│   ├── migrations/               # Database migrations
│   └── seeders/                  # Database seeders
├── html/                         # Frontend Vue application
│   └── src/
│       ├── views/                # Page components
│       ├── components/           # Reusable components
│       ├── api/                 # API client
│       └── store/               # Pinia stores
├── html-react/                   # Frontend React application (Vue parity)
│   └── src/
│       ├── pages/                # Page components
│       ├── components/           # Reusable components
│       ├── api/                  # API client
│       └── stores/               # Zustand stores
├── config/                       # Configuration files
├── docs/                         # Documentation
│   ├── API.md                    # API documentation
│   ├── ARCHITECTURE.md           # Architecture documentation
│   ├── BUILD.md                  # Build and deployment
│   ├── SHARDING_MIGRATION.md     # Database sharding guide
│   └── ...                       # Other documentation
├── CONTRIBUTING.md               # Contribution guidelines
├── CHANGELOG.md                  # Version history
└── images/                       # Screenshots
```

### Database Sharding (Advanced)

Monthly sharding (orders, etc.) is supported. **New users can skip this** until data volume requires it.  
See [docs/SHARDING_MIGRATION.md](./docs/SHARDING_MIGRATION.md) and [docs/OPENSOURCE.md](./docs/OPENSOURCE.md).

### Production Config

- **Minimal production** (admin-focused, no sharding / ES): [docs/OPENSOURCE.md](./docs/OPENSOURCE.md#3-最小生产配置) · template [`.env.production.example`](./.env.production.example)
- **Full advanced stack** (queues, sharding, ES, OTEL): [docs/OPENSOURCE.md](./docs/OPENSOURCE.md#4-完整进阶配置可选)

### Security Features

- JWT token-based authentication
- Permission middleware for route protection
- Automatic operation logging
- Sensitive data filtering in logs
- Rate limiting on login endpoints
- Blacklist management for IP/Admin blocking
- Token revocation support

## Getting Started

### Start Service

`go run . --no-ansi` or `air`

[About air]: https://www.goravel.dev/getting-started/installation.html#live-reload


### Cloudflare Workers Deployment

Deploy the frontend to Cloudflare Workers (Vue: `html/`, React: `html-react/`, same setup):

```bash
# Build the frontend (React example; use cd html for Vue)
cd html-react
# Note: Cloudflare Workers build environment automatically runs npm ci
# If you encounter Rollup optional dependency issues, use the following build command:
npm install --include=optional @rollup/rollup-linux-x64-gnu && npm run build

# Or use the project's CI build script:
npm run build:ci

# Deploy to Cloudflare Workers (requires wrangler.toml + worker.js in this directory;
# otherwise refreshing nested routes returns 404)
npx wrangler deploy
```

**Configuration:**

- **Root directory:** `html-react` (Vue: `html`)
- **Environment variables (Variables & Secrets):**
  - `VITE_API_BASE_URL`: `https://api.xuancheng888.top`
  - `VITE_API_PREFIX`: `/api/admin`
- **Custom domain:** `admin.xuancheng888.top`

**Note:** `worker.js` handles SPA routing by falling back to `index.html` for non-asset paths. Uploading only `dist` without the Worker will 404 on refresh of routes like `/admins`.

### Performance Profiling

pprof is available at: http://localhost:3000/debug/pprof/

### Binary Compression

To reduce the binary size, you can use UPX (Ultimate Packer for eXecutables) to compress the compiled executable:

**Windows:**

1. Download UPX (Windows 64-bit version):
   - Official download: https://github.com/upx/upx/releases/latest
   - Select `upx-5.0.2-win64.zip` (or the latest version)
   - Extract the zip to a path without Chinese characters or spaces (e.g., `F:\tools\upx`)
   - Ensure `upx.exe` is accessible

2. Compress the binary (PowerShell):
   ```powershell
   # Option 1: Temporarily add UPX to PATH (recommended)
   $env:PATH += ";F:\tools\upx"
   
   # Navigate to project directory
   cd F:\www\go\admin\goravel-admin
   
   # Maximum compression level (-9)
   upx -9 main
   ```

**Linux/macOS:**

```bash
# Install UPX (if not already installed)
# Ubuntu/Debian: sudo apt-get install upx
# macOS: brew install upx

# Compress the binary
upx -9 main
```


## Documentation

### Project Documentation

| Document | Description |
|----------|-------------|
| [API.md](./docs/API.md) | Complete API reference with examples |
| [ARCHITECTURE.md](./docs/ARCHITECTURE.md) | System architecture and design |
| [DEVELOPMENT_GUIDE.md](./docs/DEVELOPMENT_GUIDE.md) | Development guide: Complete CRUD module development example (using guestbook module as example, includes backend APIs and frontend pages) |
| [SHARDING_MIGRATION.md](./docs/SHARDING_MIGRATION.md) | Database sharding guide (creating, using, and modifying sharding tables) |
| [BUILD.md](./docs/BUILD.md) | Build and deployment |
| [TESTING.md](./docs/TESTING.md) | Testing guide (unit & integration) |
| [OPENSOURCE.md](./docs/OPENSOURCE.md) | Open-source positioning, core vs advanced modules, production configs |
| [QUICKSTART_DOCKER.md](./docs/QUICKSTART_DOCKER.md) | Docker Compose local setup in three commands |
| [CONTRIBUTING.md](./docs/CONTRIBUTING.md) | Contribution guide |
| [CHANGELOG.md](./docs/CHANGELOG.md) | Changelog |
| [Frontend Guide (Vue)](./html/DEVELOPMENT.md) | Vue frontend development guide |
| [Frontend Guide (React)](./html-react/README.md) | React frontend overview & setup |

### Goravel Framework

Online documentation [https://www.goravel.dev](https://www.goravel.dev)

> To optimize the documentation, please submit a PR to the documentation
> repository [https://github.com/goravel/docs](https://github.com/goravel/docs)

## Group

Welcome more discussion in Discord.

[https://discord.gg/cFc5csczzS](https://discord.gg/cFc5csczzS)

## License

The Goravel framework is open-sourced software licensed under the [MIT license](https://opensource.org/licenses/MIT).
