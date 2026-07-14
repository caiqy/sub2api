package handler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIRecordUsageInputsCarryQuotaPlatform(t *testing.T) {
	files := []string{
		"openai_gateway_handler.go",
		"openai_chat_completions.go",
		"openai_embeddings.go",
		"openai_images.go",
	}

	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
			require.NoError(t, err)

			var missing []token.Position
			ast.Inspect(file, func(node ast.Node) bool {
				literal, ok := node.(*ast.CompositeLit)
				if !ok || !isOpenAIRecordUsageInputLiteral(literal.Type) {
					return true
				}
				if !compositeLiteralHasKey(literal, "QuotaPlatform") {
					missing = append(missing, fset.Position(literal.Lbrace))
				}
				return true
			})

			require.Empty(t, missing, "OpenAI usage post-billing must receive request-time QuotaPlatform")
		})
	}
}

func TestCyberPolicyUsageInputsCarryQuotaPlatform(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(".", "openai_gateway_handler.go"), nil, 0)
	require.NoError(t, err)

	missing := map[string][]token.Position{}
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok || !isServiceInputLiteral(literal.Type, "CyberPolicyUsageInput") {
			return true
		}
		for _, key := range []string{"QuotaPlatform", "DetailSnapshot"} {
			if !compositeLiteralHasKey(literal, key) {
				missing[key] = append(missing[key], fset.Position(literal.Lbrace))
			}
		}
		return true
	})

	require.Empty(t, missing, "cyber 后扣必须携带请求时解析的平台和审计详情快照")
}

func isOpenAIRecordUsageInputLiteral(expr ast.Expr) bool {
	return isServiceInputLiteral(expr, "OpenAIRecordUsageInput")
}

func isServiceInputLiteral(expr ast.Expr, name string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "service" && selector.Sel.Name == name
}

func compositeLiteralHasKey(literal *ast.CompositeLit, key string) bool {
	for _, elt := range literal.Elts {
		pair, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		ident, ok := pair.Key.(*ast.Ident)
		if ok && ident.Name == key {
			return true
		}
	}
	return false
}
