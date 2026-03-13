# Platform Architecture

## System Overview

High-level view of the distributed platform showing all major services and their communication patterns.

```mermaid
flowchart TB
    subgraph clients["Client Layer"]
        web["Web App<br/>(React)"]
        mobile["Mobile App<br/>(React Native)"]
        cli["CLI Tool<br/>(Go)"]
    end

    subgraph edge["Edge Layer"]
        cdn["CDN<br/>(CloudFront)"]
        lb["Load Balancer<br/>(ALB)"]
        waf["WAF"]
    end

    subgraph gateway["API Gateway"]
        gw["Kong Gateway"]
        rl["Rate Limiter<br/>(Redis)"]
        auth["Auth Middleware<br/>(JWT + OAuth2)"]
    end

    subgraph services["Service Layer"]
        direction LR
        user["User Service"]
        order["Order Service"]
        product["Product Service"]
        notify["Notification Service"]
        search["Search Service"]
        analytics["Analytics Service"]
    end

    subgraph data["Data Layer"]
        pg[("PostgreSQL<br/>Primary")]
        pg_ro[("PostgreSQL<br/>Read Replica")]
        redis[("Redis Cluster<br/>Cache + Sessions")]
        es[("Elasticsearch<br/>Full-text Search")]
        s3[("S3<br/>Object Storage")]
    end

    subgraph messaging["Async Messaging"]
        kafka["Kafka"]
        dlq["Dead Letter Queue"]
    end

    subgraph observability["Observability"]
        prom["Prometheus"]
        grafana["Grafana"]
        jaeger["Jaeger<br/>Distributed Tracing"]
        sentry["Sentry<br/>Error Tracking"]
    end

    clients --> cdn --> lb --> waf --> gw
    gw --> rl
    gw --> auth
    auth --> services

    user --> pg & redis
    order --> pg & kafka
    product --> pg & redis & s3
    notify --> kafka & redis
    search --> es
    analytics --> kafka & pg_ro

    kafka --> dlq
    services --> prom
    services --> jaeger
    services --> sentry
    prom --> grafana
```

## Request Lifecycle

Detailed sequence showing how a typical API request flows through the system, including auth, caching, and async processing.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant LB as Load Balancer
    participant GW as API Gateway
    participant Auth as Auth Service
    participant Cache as Redis Cache
    participant API as Order Service
    participant DB as PostgreSQL
    participant MQ as Kafka
    participant Notif as Notification Service
    participant Email as Email Provider

    C->>+LB: POST /api/orders
    LB->>+GW: Forward request

    GW->>+Auth: Validate JWT
    Auth->>Auth: Check token expiry
    Auth->>Auth: Verify signature
    Auth-->>-GW: ✓ Valid (user_id: u-123)

    GW->>GW: Rate limit check (100 req/min)

    GW->>+API: Create order (user: u-123)

    API->>+Cache: GET cart:u-123
    Cache-->>-API: Cart items (3 products)

    API->>+DB: BEGIN TRANSACTION
    API->>DB: INSERT INTO orders
    API->>DB: INSERT INTO order_lines (×3)
    API->>DB: UPDATE inventory SET stock = stock - qty
    DB-->>-API: COMMIT ✓

    API->>+MQ: Publish "order.created"
    MQ-->>-API: ACK

    API->>+Cache: DEL cart:u-123
    Cache-->>-API: OK

    API-->>-GW: 201 Created {order_id: "ord-456"}
    GW-->>-LB: 201 Created
    LB-->>-C: 201 Created

    Note over MQ,Email: Async processing

    MQ->>+Notif: Consume "order.created"
    Notif->>Notif: Render email template
    Notif->>+Email: Send confirmation
    Email-->>-Notif: Delivered
    Notif->>MQ: Publish "notification.sent"
    deactivate Notif
```

## Deployment Pipeline

CI/CD pipeline showing the full path from code commit to production deployment with quality gates.

```mermaid
flowchart LR
    subgraph trigger["Trigger"]
        push["Git Push"]
        pr["Pull Request"]
    end

    subgraph ci["CI Pipeline"]
        lint["Lint &<br/>Format"]
        unit["Unit Tests"]
        build["Build<br/>Container"]
        scan["Security<br/>Scan"]
        integ["Integration<br/>Tests"]
    end

    subgraph cd["CD Pipeline"]
        staging["Deploy to<br/>Staging"]
        smoke["Smoke<br/>Tests"]
        approve{"Manual<br/>Approval"}
        canary["Canary<br/>(5%)"]
        monitor["Monitor<br/>15 min"]
        check{"Errors<br/>< 0.1%?"}
        rollout["Full<br/>Rollout"]
        rollback["Rollback"]
    end

    push & pr --> lint --> unit --> build --> scan --> integ
    integ --> staging --> smoke --> approve
    approve -->|approved| canary --> monitor --> check
    check -->|yes| rollout
    check -->|no| rollback
    rollback -.-> staging
    approve -->|rejected| push

    style rollback fill:#e74c3c,color:#fff
    style rollout fill:#27ae60,color:#fff
    style canary fill:#f39c12,color:#fff
```

## Data Model

Entity relationship diagram for the core domain model.

```mermaid
erDiagram
    TENANT ||--o{ USER : has
    TENANT {
        uuid id PK
        string name
        string slug UK
        enum plan "free, team, enterprise"
        jsonb settings
        timestamp created_at
    }

    USER ||--o{ PROJECT : owns
    USER ||--o{ API_KEY : has
    USER {
        uuid id PK
        uuid tenant_id FK
        string email UK
        string password_hash
        enum role "admin, member, viewer"
        timestamp last_login
    }

    PROJECT ||--o{ ENVIRONMENT : has
    PROJECT ||--o{ WEBHOOK : configures
    PROJECT {
        uuid id PK
        uuid owner_id FK
        string name
        text description
        boolean archived
        timestamp created_at
    }

    ENVIRONMENT ||--o{ DEPLOYMENT : tracks
    ENVIRONMENT {
        uuid id PK
        uuid project_id FK
        string name
        string url
        jsonb variables
        boolean auto_deploy
    }

    DEPLOYMENT ||--o{ LOG_ENTRY : produces
    DEPLOYMENT {
        uuid id PK
        uuid environment_id FK
        string commit_sha
        enum status "pending, building, deploying, live, failed, rolled_back"
        string image_tag
        timestamp started_at
        timestamp finished_at
        int duration_ms
    }

    API_KEY {
        uuid id PK
        uuid user_id FK
        string prefix UK
        string hash
        timestamp expires_at
        timestamp last_used_at
    }

    WEBHOOK {
        uuid id PK
        uuid project_id FK
        string url
        string secret_hash
        string[] events
        boolean active
    }

    LOG_ENTRY {
        uuid id PK
        uuid deployment_id FK
        enum level "debug, info, warn, error"
        text message
        jsonb metadata
        timestamp created_at
    }
```
