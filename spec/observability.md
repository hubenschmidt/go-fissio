# Observability Specification

## Overview

Add `fissio-monitor` crate for end-to-end agent observability with per-node opt-in via `NodeConfig.observe`. Includes a `/observe` UI route for viewing detailed agent traces (LangSmith-style).

## 1. fissio-monitor Crate

**Location:** `fissio/crates/fissio-monitor/`

### Core Types

```rust
// ObserveConfig - per-node opt-in settings
pub struct ObserveConfig {
    pub enabled: bool,
    pub tokens: bool,      // default true
    pub latency: bool,     // default true
    pub tool_calls: bool,  // default true
    pub cost: bool,        // default false
}

// NodeMetrics - single node execution
pub struct NodeMetrics {
    pub node_id: String,
    pub input_tokens: u32,
    pub output_tokens: u32,
    pub elapsed_ms: u64,
    pub tool_call_count: u32,
    pub iteration_count: u32,
    pub estimated_cost_usd: Option<f64>,
}

// PipelineMetrics - aggregated for full run
pub struct PipelineMetrics {
    pub pipeline_id: String,
    pub total_input_tokens: u32,
    pub total_output_tokens: u32,
    pub total_elapsed_ms: u64,
    pub total_tool_calls: u32,
    pub node_metrics: Vec<NodeMetrics>,
}

// MetricsCollector trait
pub trait MetricsCollector: Send + Sync {
    fn record(&self, metrics: NodeMetrics);
    fn flush(&self) -> PipelineMetrics;
}

// InMemoryCollector - default implementation
pub struct InMemoryCollector { ... }
```

**Dependencies:** `serde`, `serde_json`, `tracing`

## 2. Trace Storage

For the UI to display historical traces, we need persistent storage:

```rust
// TraceRecord - persisted execution trace
pub struct TraceRecord {
    pub trace_id: String,
    pub pipeline_id: String,
    pub timestamp: DateTime<Utc>,
    pub input: String,
    pub output: String,
    pub metrics: PipelineMetrics,
    pub spans: Vec<SpanRecord>,
}

// SpanRecord - individual node execution within a trace
pub struct SpanRecord {
    pub span_id: String,
    pub parent_id: Option<String>,
    pub node_id: String,
    pub node_type: String,
    pub start_time: DateTime<Utc>,
    pub end_time: DateTime<Utc>,
    pub input: String,
    pub output: String,
    pub tool_calls: Vec<ToolCallRecord>,
    pub metrics: NodeMetrics,
}

// ToolCallRecord - tool invocation details
pub struct ToolCallRecord {
    pub tool_name: String,
    pub arguments: serde_json::Value,
    pub result: String,
    pub elapsed_ms: u64,
}
```

## 3. Config Changes

**File:** `crates/fissio-config/src/lib.rs`

Add to `NodeConfig`:
```rust
#[serde(default)]
pub observe: Option<fissio_monitor::ObserveConfig>,
```

Add builder methods to `NodeBuilder`:
- `.observe()` - enable with defaults
- `.observe_config(cfg)` - custom settings

## 4. Engine Changes

**File:** `crates/fissio-engine/src/lib.rs`

- Add `collector: Option<Arc<dyn MetricsCollector>>` to `PipelineEngine`
- Add `.with_collector(collector)` builder method
- Pass collector + observe config to `execute_node()`
- Record metrics after each node completes
- Emit trace spans with input/output for each node

## 5. UI Route: `/observe`

**Route:** `GET /observe` - Trace viewer UI (LangSmith-style)

### Features

1. **Trace List View**
   - List of recent pipeline executions
   - Columns: timestamp, pipeline name, latency, token count, status
   - Filter by pipeline, date range, status
   - Sort by any column

2. **Trace Detail View**
   - Timeline/waterfall visualization of node executions
   - Expandable spans showing:
     - Node type and ID
     - Input prompt / context
     - Output / response
     - Token usage (in/out)
     - Latency
     - Tool calls with arguments and results
   - Parent-child relationships for nested calls
   - Cost breakdown per node

3. **Metrics Dashboard**
   - Total tokens over time
   - Latency percentiles (p50, p95, p99)
   - Tool call frequency
   - Cost accumulation

### API Endpoints

```
GET  /api/traces              - List traces (paginated)
GET  /api/traces/:id          - Get single trace with spans
GET  /api/traces/:id/spans    - Get spans for a trace
GET  /api/metrics/summary     - Aggregated metrics
```

### Storage

Traces stored in SQLite (existing `rusqlite` dependency):
- `traces` table - trace metadata
- `spans` table - individual node executions
- `tool_calls` table - tool invocations within spans

## 6. Workspace Updates

**File:** `fissio/Cargo.toml`

Add to members:
```toml
"crates/fissio-monitor",
```

Add to workspace.dependencies:
```toml
fissio-monitor = { path = "crates/fissio-monitor" }
```

## Critical Files

| File | Change |
|------|--------|
| `crates/fissio-monitor/src/lib.rs` | New - monitor types + collector |
| `crates/fissio-monitor/Cargo.toml` | New - dependencies |
| `crates/fissio-config/src/lib.rs` | Add `observe` field to NodeConfig |
| `crates/fissio-engine/src/lib.rs` | Collector + span emission |
| `crates/fissio-server/src/handlers/` | New trace API handlers |
| `crates/fissio-editor/` | New observe UI component |
| `fissio/Cargo.toml` | Workspace member |

## Usage Example

```rust
let config = PipelineConfig::builder("demo", "Demo")
    .node("llm", NodeType::Llm)
        .prompt("You are helpful.")
        .observe()  // enable metrics
        .done()
    .edge("input", "llm")
    .build();

let collector = Arc::new(InMemoryCollector::new("demo"));
let engine = PipelineEngine::new(config, models, model, overrides)
    .with_collector(collector.clone());

engine.execute_stream("Hello", &[]).await?;
let metrics = collector.flush();
// Trace automatically persisted to SQLite for UI viewing
```

## Verification

1. Create test pipeline with `.observe()` on nodes
2. Run pipeline execution
3. Verify metrics recorded correctly
4. Navigate to `/observe` in browser
5. Confirm trace appears in list
6. Click trace to view detailed span breakdown
7. Verify token counts, latencies, tool calls displayed
