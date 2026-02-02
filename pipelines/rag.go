// Package pipelines provides pre-built pipeline templates for common patterns.
package pipelines

import "github.com/hubenschmidt/go-fissio/config"

// NewRAGPipeline creates a Retrieval-Augmented Generation pipeline.
// The pipeline searches for relevant context, then generates a response.
func NewRAGPipeline(systemPrompt string) *config.PipelineConfig {
	retrieverPrompt := "Search for relevant context to answer the user's question. " +
		"Use the similarity_search tool to find relevant documents. " +
		"Summarize the key information found."

	return config.NewPipeline("rag", "Retrieval-Augmented Generation").
		Node("retriever", config.NodeWorker).
		Prompt(retrieverPrompt).
		Tools("similarity_search").
		MaxIterations(3).
		Done().
		Node("generator", config.NodeLLM).
		Prompt(systemPrompt).
		Done().
		Edge("retriever", "generator").
		Build()
}

// NewSimpleRAGPipeline creates a minimal RAG pipeline with a single LLM node.
// Context should be pre-fetched and included in the input.
func NewSimpleRAGPipeline(systemPrompt string) *config.PipelineConfig {
	return config.NewPipeline("simple-rag", "Simple RAG").
		Node("assistant", config.NodeLLM).
		Prompt(systemPrompt).
		Done().
		Build()
}
