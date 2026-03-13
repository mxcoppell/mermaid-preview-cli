# Architecture Document

This document shows the system architecture.

## Request Flow

```mermaid
flowchart LR
    Client --> Gateway
    Gateway --> AuthService
    Gateway --> APIServer
    APIServer --> Database
```

## Sequence

```mermaid
sequenceDiagram
    participant User
    participant API
    participant DB

    User->>API: Request
    API->>DB: Query
    DB-->>API: Result
    API-->>User: Response
```

## Notes

Some additional notes about the architecture.
