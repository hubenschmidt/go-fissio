package pipelines

import (
	"fmt"

	"github.com/hubenschmidt/go-fissio/config"
)

// NewCodeGenPipeline creates a code generation pipeline with context retrieval and validation.
// The pipeline retrieves relevant schemas/docs, generates code, then validates it.
func NewCodeGenPipeline(language string) *config.PipelineConfig {
	contextPrompt := "Search for relevant schemas, documentation, and code examples. " +
		"Use similarity_search to find context that will help generate accurate code."

	generatorPrompt := fmt.Sprintf(
		"Generate %s code based on the user's request and the provided context. "+
			"Follow best practices and ensure the code is correct and well-formatted.",
		language,
	)

	validatorPrompt := fmt.Sprintf(
		"Review the generated %s code for:\n"+
			"1. Syntax errors\n"+
			"2. Logic issues\n"+
			"3. Missing error handling\n"+
			"4. Style violations\n\n"+
			"If issues are found, provide corrected code. "+
			"If the code is correct, return it unchanged with a brief confirmation.",
		language,
	)

	return config.NewPipeline("codegen", fmt.Sprintf("%s Code Generator", language)).
		Node("context", config.NodeWorker).
		Prompt(contextPrompt).
		Tools("similarity_search").
		MaxIterations(3).
		Done().
		Node("generator", config.NodeLLM).
		Prompt(generatorPrompt).
		Done().
		Node("validator", config.NodeEvaluator).
		Prompt(validatorPrompt).
		Done().
		Edge("context", "generator").
		Edge("generator", "validator").
		Build()
}

// NewSQLGenPipeline creates a pipeline specifically for SQL generation.
func NewSQLGenPipeline() *config.PipelineConfig {
	contextPrompt := "Search for relevant table schemas, column definitions, and SQL examples. " +
		"Use similarity_search to find database documentation."

	generatorPrompt := "Generate SQL code based on the user's request and the provided schema context. " +
		"Use standard SQL syntax. Include comments explaining complex queries."

	validatorPrompt := "Review the generated SQL for:\n" +
		"1. Syntax correctness\n" +
		"2. Table and column name accuracy (based on provided context)\n" +
		"3. Join conditions\n" +
		"4. Potential performance issues\n\n" +
		"Return the final SQL, correcting any issues found."

	return config.NewPipeline("sql-gen", "SQL Generator").
		Node("context", config.NodeWorker).
		Prompt(contextPrompt).
		Tools("similarity_search").
		MaxIterations(3).
		Done().
		Node("generator", config.NodeLLM).
		Prompt(generatorPrompt).
		Done().
		Node("validator", config.NodeEvaluator).
		Prompt(validatorPrompt).
		Done().
		Edge("context", "generator").
		Edge("generator", "validator").
		Build()
}
