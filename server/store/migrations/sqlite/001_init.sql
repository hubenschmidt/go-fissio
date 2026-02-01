-- Traces table
CREATE TABLE IF NOT EXISTS traces (
    trace_id TEXT PRIMARY KEY,
    pipeline_id TEXT NOT NULL,
    pipeline_name TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    input TEXT NOT NULL,
    output TEXT NOT NULL,
    total_elapsed_ms INTEGER NOT NULL,
    total_input_tokens INTEGER NOT NULL,
    total_output_tokens INTEGER NOT NULL,
    total_tool_calls INTEGER NOT NULL,
    status TEXT NOT NULL,
    spans TEXT DEFAULT '[]'
);

CREATE INDEX IF NOT EXISTS idx_traces_timestamp ON traces(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_traces_pipeline_id ON traces(pipeline_id);

-- Pipelines table
CREATE TABLE IF NOT EXISTS pipelines (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    nodes TEXT DEFAULT '[]',
    edges TEXT DEFAULT '[]',
    layout TEXT DEFAULT '{}'
);
