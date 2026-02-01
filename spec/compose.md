# Self-Organizing Agent Feature Gap Analysis

## Executive Summary

The frontend `/compose` mode and Pipeline Editor promise sophisticated agent orchestration capabilities, but the Rust backend only implements a **simple sequential/parallel DAG executor**. Several critical features are missing or non-functional.

---

## Critical Gaps (Backend Limitations)

### 1. Anthropic Tool Calling NOT Implemented
**Impact:** Claude models (Opus, Sonnet, Haiku) cannot use tools like `web_search` or `fetch_url`

**Location:** `agent/crates/agent-network/src/unified.rs:90-94`
```rust
ProviderType::Anthropic => {
    // TODO: Implement Anthropic tool calling
    Err(AgentError::LlmError("Tool calling not yet supported for Anthropic models".to_string()))
}
```

**Fix Required:** Implement Anthropic tool_use format per their API spec.

---

### 2. Non-LLM Node Types Are Pass-Through Only
**Impact:** 7 of 9 node types do nothing — they just forward input unchanged

| Node Type | Frontend Promise | Backend Reality |
|-----------|-----------------|-----------------|
| `llm` | LLM calls | Works |
| `worker` | LLM + tools | Works |
| `router` | Route based on classification | Pass-through |
| `gate` | Validate before proceeding | Pass-through |
| `coordinator` | Distribute to workers | Pass-through |
| `aggregator` | Combine outputs | Pass-through |
| `orchestrator` | Dynamic task decomposition | Pass-through |
| `synthesizer` | Synthesize inputs | Pass-through |
| `evaluator` | Quality evaluation | Pass-through |

**Location:** `agent/crates/agent-engine/src/lib.rs:348-352`
```rust
let content = if node_type.requires_llm() {
    execute_node_with_tools(...).await?
} else {
    input.to_string()  // All non-LLM nodes just pass through
};
```

**Fix Required:** Implement actual logic for each node type.

---

### 3. Conditional & Dynamic Edges Not Implemented
**Impact:** Edge types are parsed but ignored at runtime

| Edge Type | Frontend Promise | Backend Reality |
|-----------|-----------------|-----------------|
| `direct` | Sequential flow | Works |
| `parallel` | Concurrent execution | Works |
| `conditional` | Router chooses path | Treated as direct |
| `dynamic` | Orchestrator picks workers | Treated as direct |
| `feedback` | Loop back for refinement | Impossible |

**Location:** `agent/crates/agent-engine/src/lib.rs:191` — only checks for `Parallel`

---

### 4. Feedback Loops Impossible
**Impact:** Evaluator-optimizer patterns cannot work

The engine uses forward-only DAG traversal:
- Nodes marked in `HashSet<String>` after execution
- Once executed, never re-executed
- Backward edges are completely ignored

**Location:** `agent/crates/agent-engine/src/lib.rs:139-142, 251-256`

---

### 5. Limited Tool Ecosystem
**Current tools:** Only 2 available
- `fetch_url` — HTTP GET + HTML parsing
- `web_search` — Tavily API (requires `TAVILY_API_KEY`)

**Missing for self-organization:**
- File read/write
- Code execution
- Database queries
- API calling (generic HTTP POST)
- Memory/state persistence

---

## What Currently Works

| Feature | Status |
|---------|--------|
| LLM node execution | Works |
| Worker node with tools (OpenAI only) | Works |
| Parallel edge execution | Works |
| Pipeline CRUD (save/load/delete) | Works |
| Model selection per node | Works |
| Tool assignment per node | Works |
| Visual pipeline editor with drag/drop | Works |
| Compose mode LLM-guided design | Works |

---

## Recommended Priority Fixes

### P0 — Essential for Basic Self-Organization
1. **Implement Anthropic tool calling** — unblocks Claude for agentic use
2. **Implement Router node** — actual classification -> conditional edge selection

### P1 — Enables Iterative Workflows
3. **Implement Evaluator node** — quality scoring with pass/fail output
4. **Implement feedback edge support** — allow controlled re-execution loops

### P2 — Enables Complex Orchestration
5. **Implement Orchestrator node** — dynamic task decomposition
6. **Implement Aggregator node** — proper multi-input synthesis

### P3 — Expands Capabilities
7. **Add more tools** — file I/O, code execution, generic HTTP

---

## Files to Modify

| Priority | File | Change |
|----------|------|--------|
| P0 | `agent/crates/agent-network/src/unified.rs` | Add Anthropic tool_use support |
| P0 | `agent/crates/agent-engine/src/lib.rs` | Implement Router node logic |
| P1 | `agent/crates/agent-engine/src/lib.rs` | Implement Evaluator + feedback loops |
| P2 | `agent/crates/agent-engine/src/lib.rs` | Implement Orchestrator/Aggregator |
| P3 | `agent/crates/agent-tools/src/` | Add new tool implementations |

---

## Implementation Plan: P0 — Anthropic Tools + Router

### Task 1: Implement Anthropic Tool Calling

**Files:**
- `agent/crates/agent-network/src/anthropic.rs` — Add tool support structs and method
- `agent/crates/agent-network/src/unified.rs` — Wire up Anthropic branch

**Implementation in anthropic.rs:**

1. Add new request/response structs for tools:
```rust
#[derive(Serialize)]
struct AnthropicTool {
    name: String,
    description: String,
    input_schema: serde_json::Value,
}

#[derive(Serialize)]
struct AnthropicRequestWithTools {
    model: String,
    max_tokens: u32,
    system: String,
    messages: Vec<AnthropicMessageContent>,
    tools: Vec<AnthropicTool>,
}

#[derive(Serialize, Deserialize)]
#[serde(tag = "type")]
enum ContentBlock {
    #[serde(rename = "text")]
    Text { text: String },
    #[serde(rename = "tool_use")]
    ToolUse { id: String, name: String, input: serde_json::Value },
    #[serde(rename = "tool_result")]
    ToolResult { tool_use_id: String, content: String },
}
```

2. Add `chat_with_tools()` method to `AnthropicClient`:
```rust
pub async fn chat_with_tools(
    &self,
    system_prompt: &str,
    messages: Vec<AnthropicMessageContent>,
    tools: &[ToolSchema],
) -> Result<ChatResponse, AgentError>
```

3. Convert `ToolSchema` to `AnthropicTool` format (rename `parameters` -> `input_schema`)

4. Parse response for `tool_use` blocks and return `ChatResponse::ToolCalls`

**Wire up in unified.rs:**
```rust
ProviderType::Anthropic => {
    let client = AnthropicClient::new(&self.model);
    // Convert OpenAI message format to Anthropic format
    let anthropic_messages = convert_messages(messages)?;
    client.chat_with_tools(system_prompt, anthropic_messages, tools).await
}
```

### Task 2: Implement Router Node Logic

**File:** `agent/crates/agent-engine/src/lib.rs`

**Changes:**
1. Make Router node execute an LLM call with a classification prompt
2. LLM output determines which outgoing edge to follow
3. Only activate edges that match the router's decision

**Router Logic:**
```rust
NodeType::Router => {
    // Build classification prompt from node's system prompt
    // Ask LLM to classify input into one of the target node categories
    // Return the classification result
    // Engine then only follows the matching conditional edge
}
```

**Edge Selection:**
- Router outputs a target node ID or category
- Engine filters outgoing edges to only follow the match
- Requires conditional edge support in traversal logic

### Task 3: Implement Conditional Edge Support

**File:** `agent/crates/agent-engine/src/lib.rs`

**Changes:**
1. When source node is Router, check its output against edge targets
2. Only queue edges where target matches router decision
3. Add `conditional` edge type handling alongside `parallel`

---

## Files to Modify

| File | Changes |
|------|---------|
| `agent/crates/agent-network/src/unified.rs` | Anthropic tool_use implementation |
| `agent/crates/agent-network/src/anthropic.rs` | Helper functions for tool format |
| `agent/crates/agent-engine/src/lib.rs` | Router node logic + conditional edge handling |
| `agent/crates/agent-config/src/lib.rs` | May need `requires_routing()` helper |

---

## Verification

1. **Anthropic tools test:**
   - Select Claude Sonnet model
   - Use "Direct Chat" with a worker node that has `web_search` tool
   - Ask "Search for the latest Rust news"
   - Verify tool is called and results returned

2. **Router test:**
   - Create pipeline: input -> router -> [worker_a, worker_b] -> output
   - Router prompt: "Classify as 'technical' or 'general'"
   - Conditional edges: router->worker_a (technical), router->worker_b (general)
   - Send technical question -> verify only worker_a executes
   - Send general question -> verify only worker_b executes

3. **Regression test:**
   - Run existing preset configs (research_assistant, etc.)
   - Verify no breakage in existing functionality
