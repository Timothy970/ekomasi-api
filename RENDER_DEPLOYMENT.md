# Deploying ekomasi-api to Render with Docker

This guide explains how to deploy `ekomasi-api` to [Render](https://render.com) using Docker, a Managed Render Redis instance, and Aiven MySQL.

---

## 1. Prerequisites

- A [Render](https://dashboard.render.com) account.
- An [Aiven MySQL](https://aiven.io) database instance running.
- A [Cloudinary](https://cloudinary.com) account for media image uploads.

---

## 2. Render Blueprint File (`render.yaml`)

The repository includes a `render.yaml` file in the root directory:

```yaml
services:
  - type: web
    name: ekomasi-api
    runtime: docker
    dockerfilePath: ./Dockerfile
    context: .
    plan: starter
    region: oregon
    healthCheckPath: /api/health
    autoDeploy: true
    envVars:
      - key: ENVIRONMENT
        value: production
      - key: PORT
        value: 10000

      # Aiven MySQL Database Configuration
      - key: MYSQL_HOST
        sync: false
      - key: MYSQL_PORT
        value: "3306"
      - key: MYSQL_USER
        sync: false
      - key: MYSQL_PASS
        sync: false
      - key: DB_NAME
        value: ekomasi

      # Render Managed Redis
      - key: REDIS_HOST
        fromKeyVal:
          name: ekomasi-redis
          property: host
      - key: REDIS_PORT
        fromKeyVal:
          name: ekomasi-redis
          property: port

      # Cloudinary Configuration
      - key: USE_CLOUDINARY
        value: "true"
      - key: CLOUDINARY_CLOUD_NAME
        sync: false
      - key: CLOUDINARY_API_KEY
        sync: false
      - key: CLOUDINARY_API_SECRET
        sync: false

  - type: redis
    name: ekomasi-redis
    plan: free
    region: oregon
    maxmemoryPolicy: allkeys-lru
    ipAllowList: []
```

---

## 3. Step-by-Step Deployment Instructions

### Step 1: Connect Repository to Render
1. Push your code with `render.yaml` to GitHub or GitLab.
2. Go to [Render Dashboard](https://dashboard.render.com).
3. Click **New +** -> **Blueprint**.
4. Connect your GitHub/GitLab repository.
5. Render will automatically detect `render.yaml` and parse the Web Service and Redis Instance.

### Step 2: Configure Environment Variables in Render
In the Render Dashboard (under Environment Variables for `ekomasi-api`), fill in the `sync: false` variables:

1. **Aiven MySQL**:
   - `MYSQL_HOST`: `<your-aiven-mysql-host>.aivencloud.com`
   - `MYSQL_PORT`: `3306` (or Aiven port)
   - `MYSQL_USER`: `<your-aiven-db-user>`
   - `MYSQL_PASS`: `<your-aiven-db-password>`
   - `DB_NAME`: `ekomasi`

2. **Cloudinary**:
   - `CLOUDINARY_CLOUD_NAME`: `<your-cloudinary-cloud-name>`
   - `CLOUDINARY_API_KEY`: `<your-cloudinary-api-key>`
   - `CLOUDINARY_API_SECRET`: `<your-cloudinary-api-secret>`

3. **Application URLs & Credentials**:
   - `BASE_URL`: `https://ekomasi-api.onrender.com`
   - `FRONT_END_BASE_URL`: `https://your-frontend-domain.com`

### Step 3: Trigger Deployment
1. Click **Apply** in Render.
2. Render will automatically:
   - Provision the free `ekomasi-redis` key-value store.
   - Build the Docker container using `Dockerfile`.
   - Start the service and test `/api/health`.

---

## 4. Verification

Verify deployment health by visiting:
`https://<your-render-app-name>.onrender.com/api/health`
