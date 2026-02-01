# Fissio — Pipeline-First Agent Framework

## Vision

Transform the backend into **fissio**, a pipeline-first Rust framework for building LLM-powered agent systems. Unlike agent-centric frameworks (Helios, LangGraph), fissio treats **declarative pipeline definitions** as the primary abstraction.

**Tagline:** *"Pipelines as code — declarative, visual, composable agent orchestration"*

**Name:** `fissio` (evokes fission — splitting/branching of execution paths)

---

## Differentiators vs Helios/Others

| Aspect | Helios/LangGraph | This Framework |
|--------|------------------|----------------|
| Primary abstraction | Agent (ReAct loop) | **Pipeline (DAG)** |
| Definition style | Code (builder pattern) | **Declarative JSON/YAML** |
| Orchestration | Agent delegation | **Graph topology** |
| Node types | Generic | **Specialized (Router, Evaluator, Aggregator)** |
| Visual tooling | None | **Pipeline Editor (separate)** |
| Execution model | ReAct loop | **DAG traversal (parallel/conditional)** |

---

## Current Crate Structure

```
agent/crates/
├── agent-config/    # Pipeline schema, node/edge types
├── agent-core/      # Error types, Message, ModelConfig
├── agent-engine/    # DAG execution engine
├── agent-network/   # LLM clients (OpenAI, Anthropic)
├── agent-tools/     # Tool registry + implementations
├── agent-server/    # Axum HTTP server (app-specific)
└── ... others
```

---

## Framework Structure (Workspace)

```
fissio/
├── Cargo.toml                  # Workspace root
├── crates/
│   ├── fissio-config/          # Pipeline schema, validation, node/edge types
│   ├── fissio-engine/          # Core DAG execution engine
│   ├── fissio-llm/             # LLM provider abstraction (OpenAI, Anthropic)
│   ├── fissio-tools/           # Tool registry + built-in tools
│   └── fissio/                 # Facade crate (re-exports all)
├── examples/                   # Example pipelines + Rust code
└── README.md
```

**Future:** `fissio-cli/` crate for command-line pipeline execution

---

## Core Public API Design

### 1. Pipeline Definition (JSON)
```json
{
  "id": "research-pipeline",
  "name": "Research Assistant",
  "nodes": [
    { "id": "router", "type": "router", "prompt": "Classify as technical or general" },
    { "id": "researcher", "type": "worker", "model": "gpt-4", "tools": ["web_search"] },
    { "id": "writer", "type": "llm", "prompt": "Summarize findings" }
  ],
  "edges": [
    { "from": "input", "to": "router" },
    { "from": "router", "to": ["researcher", "writer"], "edge_type": "conditional" },
    { "from": "researcher", "to": "output" }
  ]
}
```

### 2. Rust API
```rust
use fissio::{Pipeline, ToolRegistry, NodeType};

// Load from JSON
let pipeline = Pipeline::from_file("research.json")?;

// Or build programmatically
let pipeline = Pipeline::builder()
    .node("router", NodeType::Router)
        .prompt("Classify input")
        .build()
    .node("worker", NodeType::Worker)
        .model("gpt-4")
        .tools(["web_search"])
        .build()
    .edge("input", "router")
    .edge("router", "worker").conditional()
    .build()?;

// Execute
let result = pipeline.execute("What is Rust?").await?;

// Or stream
let stream = pipeline.execute_stream("What is Rust?").await?;
```

### 3. Custom Tools
```rust
use fissio::Tool;

#[derive(Tool)]
#[tool(name = "calculator", description = "Performs math")]
async fn calculator(
    #[arg(description = "Math expression")] expression: String,
) -> Result<String, ToolError> {
    // implementation
}

// Register
let registry = ToolRegistry::new()
    .register(calculator)
    .register(web_search);
```

### 4. Custom Node Types
```rust
use fissio::{NodeExecutor, NodeContext, NodeOutput};

struct MyCustomNode;

#[async_trait]
impl NodeExecutor for MyCustomNode {
    async fn execute(&self, ctx: &NodeContext) -> Result<NodeOutput, Error> {
        // custom logic
    }
}

pipeline.register_node_type("my_custom", MyCustomNode);
```

---

## Extension Points

1. **Custom Node Types** — Implement `NodeExecutor` trait
2. **Custom Tools** — Implement `Tool` trait or use `#[derive(Tool)]`
3. **Custom LLM Providers** — Implement `LlmProvider` trait
4. **Custom Edge Types** — Implement `EdgeHandler` trait
5. **Middleware/Hooks** — Before/after node execution

---

## Feature Flags

```toml
[features]
default = ["openai", "anthropic", "tools-web"]
openai = []
anthropic = []
tools-web = ["reqwest"]      # web_search, fetch_url
tools-fs = []                # file read/write
local-llm = ["llama-cpp"]    # local model support
```

---

## Implementation Roadmap

### Phase 1: Restructure as Workspace
- [ ] Create `fissio/` directory with workspace Cargo.toml
- [ ] Rename/move crates: agent-* → fissio-*
- [ ] Create facade crate `fissio/` that re-exports all
- [ ] Update agent-server to depend on fissio workspace

### Phase 2: API Cleanup
- [ ] Define stable public API surface
- [ ] Add `#[doc(hidden)]` to internals
- [ ] Create `prelude` module with common re-exports
- [ ] Add comprehensive doc comments

### Phase 3: Decouple from Server
- [ ] Remove any server-specific code from engine
- [ ] Ensure engine is pure library (no HTTP/framework concerns)
- [ ] agent-server becomes a consumer of fissio

### Phase 4: Improve Ergonomics
- [ ] Add `Pipeline::builder()` API for programmatic construction
- [ ] Better error messages with context (thiserror + miette?)
- [ ] Consider YAML support alongside JSON

### Phase 5: Documentation & Examples
- [ ] Write README with quick start
- [ ] Add examples/ directory with common patterns
- [ ] Generate rustdoc documentation

### Phase 6: Publish
- [ ] Reserve crate names on crates.io
- [ ] Publish initial version (0.1.0)

### Future: CLI
- [ ] Create `fissio-cli/` crate
- [ ] `fissio run pipeline.json` command
- [ ] `fissio validate pipeline.json` command

---

## Decisions Made

- **Name:** fissio
- **Structure:** Workspace with multiple crates
- **CLI:** Future phase (fissio-cli crate)

## Open Questions

1. **Support YAML configs?** (in addition to JSON)
2. **Proc macro crate?** (for `#[derive(Tool)]` — requires separate crate)

---

## Crate Mapping

| Current | Fissio Crate | Notes |
|---------|--------------|-------|
| `agent-config/` | `fissio-config/` | Pipeline schema, node/edge types |
| `agent-engine/` | `fissio-engine/` | Core DAG execution |
| `agent-network/` | `fissio-llm/` | LLM provider abstraction |
| `agent-tools/` | `fissio-tools/` | Tool registry + built-ins |
| `agent-core/` | merged | Error types → fissio-engine, ModelConfig → fissio-llm |
| `agent-server/` | **stays separate** | App-specific, depends on fissio |

---

## Verification

1. Existing app still works after refactor (agent-server uses fissio)
2. Can `cargo add fissio` and build a pipeline from scratch
3. Documentation builds (`cargo doc --open`)
4. Feature flags work correctly
5. Examples compile and run
