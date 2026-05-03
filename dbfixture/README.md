# dbfixture

Database fixture loading for testing and development. Loads test data into PostgreSQL and ClickHouse
from YAML files or programmatic definitions.

## Overview

The package provides a complete fixture system for populating databases with test data:

- **Fixture**: Central registry of model loaders and loaded data, keyed by name
- **Loader[Input, Model]**: Per-model pipeline that resolves, defaults, validates, and inserts data
- **pgLoaderBase / chSpanLoaderBase**: Shared base implementations for PostgreSQL and ClickHouse
  loaders

## Usage

### Programmatic Fixtures

```go
fixture := dbfixture.NewFixture(params,
    dbfixture.WithPresets("default"),  // adds default user/org/project
    dbfixture.WithFakeData(true),     // auto-populate with gofakeit
)

seed := &seedinput.SeedData{
    Orgs: []*seedinput.Org{{Key: "myorg", Name: ptr("My Org")}},
}

// Load into database
err := fixture.Seed(ctx, seed)

// Access loaded models by key
user := dbfixture.MustGet[*models.User](fixture, "default")
project := dbfixture.MustGet[*models.Project](fixture, "default")
```

### YAML Fixtures

```go
fixture, seedData, err := dbfixture.ParseFile(params, "testdata/fixtures.yaml", nil,
    dbfixture.WithPresets("default"),
)
if err != nil {
    return err
}

err = fixture.Seed(ctx, seedData)
```

Example YAML file:

```yaml
update: true # upsert instead of insert
delete: false # delete orphaned records

orgs:
  - key: testorg
    name: 'Test Organization'
    budget: 100.0

users:
  - key: testuser
    name: 'Test User'
    email: 'test@example.com'
    password: 'secret'
    orgs:
      - org_key: testorg
        role: owner

projects:
  - key: testproject
    org_key: testorg
    name: 'Test Project'

metric_monitors:
  - key: cpu-monitor
    project_key: testproject
    name: 'CPU Monitor'
    params:
      metric_exprs:
        - $cpu = system_cpu_utilization
      query_parts:
        - group by host_name

spans:
  - id: 1
    project_key: testproject
    name: 'HTTP GET /api/users'
    type: http-server
    duration: 150000 # microseconds
    attrs:
      http.method: GET
      http.route: /api/users
```

### YAML Template Functions

Templates use [sprig](https://masterminds.github.io/sprig/) functions plus custom time helpers:

```yaml
spans:
  - id: 1
    start_time: {{ now }}
    name: "span-{{ randInt 1 100 }}"

# Generate time series data
datapoints:
  {{- range $i, $t := interval 10 "1m" }}
  - project_key: default
    metric: system_cpu_utilization
    time: {{ $t }}
    sum: {{ randFloat 0.1 0.9 }}
  {{- end }}
```

Available custom functions:

| Function                    | Description                     | Example                         |
| --------------------------- | ------------------------------- | ------------------------------- |
| `now`                       | Current time as `unixtime.Nano` | `{{ now }}`                     |
| `interval n duration`       | Generate n timestamps           | `{{ interval 10 "1m" }}`        |
| `addDuration time duration` | Add duration to time            | `{{ addDuration (now) "-1h" }}` |
| `seq n`                     | Generate sequence [0, n)        | `{{ range seq 5 }}...{{ end }}` |
| `randInt min max`           | Random integer                  | `{{ randInt 1 100 }}`           |
| `randFloat min max`         | Random float                    | `{{ randFloat 0.0 1.0 }}`       |

## Options

```go
fixture := dbfixture.NewFixture(params,
    dbfixture.WithClock(clock),           // custom clock for timestamps
    dbfixture.WithPresets("default"),      // add default user/org/project
    dbfixture.WithFakeData(true),         // auto-populate empty fields with random data
    dbfixture.WithValidation(true),       // validate models before insert (default: true)
    dbfixture.WithPersistKeys(true),      // load/save keys from fixture_keys table
    dbfixture.WithProjectOverride("default"), // force project (resolved from KeyStore by key)
    dbfixture.WithAssignGroupID(true),    // compute group hashes for spans
    dbfixture.AllowConfirmedEmails(true), // allow email_confirmed field on users
)

// Update/Delete are controlled via SeedData fields:
seed.Update = true // upsert mode
seed.Delete = true // delete orphans not in fixture
fixture.Seed(ctx, seed)
```

## Supported Types

### PostgreSQL (Metadata)

| Type             | Loader Constant          |
| ---------------- | ------------------------ |
| Org              | `ModelOrg`               |
| User             | `ModelUser`              |
| UserToken        | `ModelUserToken`         |
| OrgUser          | `ModelOrgUser`           |
| Project          | `ModelProject`           |
| ProjectToken     | `ModelProjectToken`      |
| ProjectUser      | `ModelProjectUser`       |
| Team             | `ModelTeam`              |
| TeamUser         | `ModelTeamUser`          |
| TeamProject      | `ModelTeamProject`       |
| Dashboard        | `ModelDashboard`         |
| ProjectTag       | `ModelProjectTag`        |
| TaggedDashboard  | `ModelTaggedDashboard`   |
| MetricMonitor    | `ModelMetricMonitor`     |
| ErrorMonitor     | `ModelErrorMonitor`      |
| MetricAlert      | `ModelMetricAlert`       |
| ErrorAlert       | `ModelErrorAlert`        |
| MetricAlertEvent | `ModelMetricAlertEvent`  |
| ErrorAlertEvent  | `ModelErrorAlertEvent`   |
| AlertNote        | `ModelAlertNote`         |
| AlertAssignment  | `ModelAlertAssignment`   |
| NotifChannels    | `ModelSlackNotifChannel`, `ModelWebhookNotifChannel`, etc. |

### ClickHouse (Telemetry)

| Type           | Loader Constant      |
| -------------- | -------------------- |
| Span           | `ModelSpan`          |
| Log            | `ModelLog`           |
| Event          | `ModelEvent`         |
| TraceOverride  | `ModelTraceOverride` |
| Datapoint      | `ModelDatapoint`     |

## Cleanup

```go
// Delete all loaded fixtures from the database in reverse dependency order.
err := fixture.Clear(ctx)
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Fixture                             │
│  (registry of model loaders + loaded data, keyed by name)   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              Loader[Input, Model] interface                  │
│  (per-model pipeline: resolve, defaults,                     │
│   validate, insert/update, select)                          │
│                                                             │
│  pgLoaderBase[Input, Model]  — PostgreSQL base              │
│  chSpanLoaderBase            — ClickHouse base              │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   AnyLoader (type-erased)                    │
│  (Wrap[Input, Model] adapts typed → type-erased for the     │
│   registry; Get[T]/MustGet[T] provide typed access back)    │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Loaded Models                          │
│  (maps of loaded model values indexed by fixture key,       │
│   provides typed getters via Get[T] / MustGet[T])           │
└─────────────────────────────────────────────────────────────┘
```
