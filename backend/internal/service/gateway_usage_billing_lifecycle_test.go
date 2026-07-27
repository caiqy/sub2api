package service

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFinalizePostUsageBillingDoesNotStartDetachedGoroutines(t *testing.T) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "gateway_usage_billing.go", nil, 0)
	require.NoError(t, err)

	var finalize *ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == "finalizePostUsageBilling" {
			finalize = function
			break
		}
	}
	require.NotNil(t, finalize)

	hasDetachedGoroutine := false
	ast.Inspect(finalize.Body, func(node ast.Node) bool {
		if _, ok := node.(*ast.GoStmt); ok {
			hasDetachedGoroutine = true
		}
		return true
	})
	require.False(t, hasDetachedGoroutine)
}
