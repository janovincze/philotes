# Research Findings - Issue #12: Setup Wizard

## 1. Existing API Endpoints

### Source Management (`/api/v1/sources`)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/sources` | POST | Create source |
| `/sources/:id` | GET | Get source details |
| `/sources/:id/test` | POST | Test connection |
| `/sources/:id/tables` | GET | Discover tables |

**CreateSourceRequest:**
```typescript
{
  name: string              // Required: 1-255 chars
  type: "postgresql"        // Defaults to "postgresql"
  host: string              // Required
  port: number              // Defaults to 5432
  database_name: string     // Required
  username: string          // Required
  password: string          // Required
  ssl_mode?: string         // Defaults to "prefer"
}
```

**ConnectionTestResult:**
```typescript
{
  success: boolean
  message: string
  latency_ms?: number
  server_info?: string      // PostgreSQL version
  error_detail?: string
}
```

**TableDiscoveryResponse:**
```typescript
{
  tables: TableInfo[]
  count: number
}

interface TableInfo {
  schema: string
  name: string
  columns: ColumnInfo[]
}

interface ColumnInfo {
  name: string
  type: string
  nullable: boolean
  primary_key: boolean
  default?: string
}
```

### Pipeline Management (`/api/v1/pipelines`)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/pipelines` | POST | Create pipeline with table mappings |
| `/pipelines/:id/start` | POST | Start pipeline |

**CreatePipelineRequest:**
```typescript
{
  name: string
  source_id: uuid
  tables?: CreateTableMappingRequest[]
  config?: Record<string, any>
}

interface CreateTableMappingRequest {
  schema?: string           // Defaults to "public"
  table: string             // Required
  enabled?: boolean         // Defaults to true
}
```

## 2. Existing Frontend Infrastructure

### API Client (`/web/src/lib/api/`)
- `sourcesApi.create(input)` - Creates source
- `sourcesApi.testConnection(id)` - Tests connection (returns `{ success, message }`)
- `sourcesApi.discoverTables(id)` - **Gap:** Returns `string[]` but backend returns full `TableDiscoveryResponse`
- `pipelinesApi.create(input)` - Creates pipeline

### React Query Hooks (`/web/src/lib/hooks/`)
- `useSources()` - List sources
- `useSource(id)` - Get single source
- `usePipelines()` - List pipelines
- Pattern: `useQuery` for fetching, `useMutation` for mutations

### UI Components (shadcn/ui)
- `Form` - react-hook-form wrapper with FormField, FormItem, FormLabel, FormControl, FormMessage
- `Input`, `Button`, `Card`, `Select`, `Badge`, `Separator`, `Tabs`, `Skeleton`
- Form validation with `zod`

## 3. Gaps to Address

1. **Frontend API Client:** `discoverTables()` should return full table info with columns
2. **No Multi-Step Form:** Need step indicator component
3. **No Test Connection Hook:** Need `useTestConnection()` mutation hook

## 4. Recommended Wizard Flow

```
Step 1: Welcome
    ↓
Step 2: Database Connection Form
    ↓
    API: POST /sources (create source)
    API: POST /sources/:id/test (test connection)
    ↓
Step 3: Table Selection
    ↓
    API: GET /sources/:id/tables (discover tables)
    ↓
Step 4: Review & Configure
    ↓
Step 5: Create Pipeline
    ↓
    API: POST /pipelines (create with tables)
    API: POST /pipelines/:id/start (start pipeline)
    ↓
Step 6: Success / Watch Sync
```

## 5. Key Files to Reference

- `web/src/components/scaling/scaling-policy-form.tsx` - Complex form pattern
- `web/src/app/sources/page.tsx` - Source listing pattern
- `web/src/lib/api/sources.ts` - API client
- `web/src/lib/hooks/use-sources.ts` - Query hooks
- `internal/api/services/source.go` - Backend connection test logic
