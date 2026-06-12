package runtime

import (
	"sort"
	"strings"

	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
)

func FindScopeVars(jsCode string, line int) []string {
	program, err := parser.ParseFile(nil, "", jsCode, 0, parser.WithDisableSourceMaps)
	if err != nil || program == nil {
		return nil
	}

	lineStart := lineToOffset(jsCode, line)
	lineEnd := lineToOffset(jsCode, line+1)
	if lineEnd == 0 {
		lineEnd = len(jsCode)
	}

	var result []string
	walkStmtList(program.Body, program.DeclarationList, lineStart, lineEnd, &result)
	sort.Strings(result)
	return result
}

func lineToOffset(code string, line int) int {
	offset := 0
	lines := strings.Split(code, "\n")
	for i := 0; i < line-1 && i < len(lines); i++ {
		offset += len(lines[i]) + 1
	}
	return offset
}

func walkStmtList(body []ast.Statement, decls []*ast.VariableDeclaration, lineStart, lineEnd int, result *[]string) {
	for _, d := range decls {
		collectBindings(d.List, result, lineStart, lineEnd)
	}
	for _, s := range body {
		walkNode(s, lineStart, lineEnd, result)
	}
}

func walkNode(n ast.Node, lineStart, lineEnd int, result *[]string) {
	if n == nil {
		return
	}
	if int(n.Idx0()) >= lineEnd {
		return
	}

	isSameLine := int(n.Idx0()) >= lineStart && int(n.Idx0()) < lineEnd

	switch st := n.(type) {
	case *ast.BlockStatement:
		walkStmtList(st.List, nil, lineStart, lineEnd, result)
	case *ast.VariableStatement:
		if !isSameLine {
			collectBindings(st.List, result, lineStart, lineEnd)
		}
	case *ast.LexicalDeclaration:
		if !isSameLine {
			collectBindings(st.List, result, lineStart, lineEnd)
		}
	case *ast.FunctionDeclaration:
		if st.Function != nil {
			for _, p := range st.Function.ParameterList.List {
				if id, ok := p.Target.(*ast.Identifier); ok {
					addTo(result, id.Name.String())
				}
			}
			if st.Function.Name != nil {
				addTo(result, st.Function.Name.Name.String())
			}
			if st.Function.Body != nil {
				walkNode(st.Function.Body, lineStart, lineEnd, result)
			}
		}
	case *ast.IfStatement:
		walkNode(st.Consequent, lineStart, lineEnd, result)
		walkNode(st.Alternate, lineStart, lineEnd, result)
	case *ast.WhileStatement, *ast.DoWhileStatement:
		if s, ok := n.(*ast.WhileStatement); ok {
			walkNode(s.Body, lineStart, lineEnd, result)
		} else if s, ok := n.(*ast.DoWhileStatement); ok {
			walkNode(s.Body, lineStart, lineEnd, result)
		}
	case *ast.ForStatement:
		if st.Body != nil {
			if vs, ok := st.Initializer.(ast.Node); ok {
				if vstmt, ok := vs.(*ast.VariableStatement); ok {
					collectBindings(vstmt.List, result, lineStart, lineEnd)
				}
				if ldecl, ok := vs.(*ast.LexicalDeclaration); ok {
					collectBindings(ldecl.List, result, lineStart, lineEnd)
				}
			}
			walkNode(st.Body, lineStart, lineEnd, result)
		}
	case *ast.TryStatement:
		walkNode(st.Body, lineStart, lineEnd, result)
		if st.Catch != nil {
			walkNode(st.Catch.Body, lineStart, lineEnd, result)
		}
		if st.Finally != nil {
			walkNode(st.Finally, lineStart, lineEnd, result)
		}
	case *ast.SwitchStatement:
		for _, cs := range st.Body {
			walkNode(cs, lineStart, lineEnd, result)
		}
	case *ast.ExpressionStatement:
		walkExpr(st.Expression, lineStart, lineEnd, result)
	}
}

func walkExpr(e ast.Expression, lineStart, lineEnd int, result *[]string) {
	if e == nil {
		return
	}
	switch ex := e.(type) {
	case *ast.FunctionLiteral:
		for _, p := range ex.ParameterList.List {
			if id, ok := p.Target.(*ast.Identifier); ok {
				addTo(result, id.Name.String())
			}
		}
		if ex.Body != nil {
			walkNode(ex.Body, lineStart, lineEnd, result)
		}
	case *ast.ArrowFunctionLiteral:
		for _, p := range ex.ParameterList.List {
			if id, ok := p.Target.(*ast.Identifier); ok {
				addTo(result, id.Name.String())
			}
		}
		if ex.Body != nil {
			walkNode(ex.Body, lineStart, lineEnd, result)
		}
	case *ast.CallExpression:
		walkExpr(ex.Callee, lineStart, lineEnd, result)
		for _, a := range ex.ArgumentList {
			walkExpr(a, lineStart, lineEnd, result)
		}
	case *ast.AssignExpression:
		walkExpr(ex.Left, lineStart, lineEnd, result)
		walkExpr(ex.Right, lineStart, lineEnd, result)
	case *ast.BinaryExpression:
		walkExpr(ex.Left, lineStart, lineEnd, result)
		walkExpr(ex.Right, lineStart, lineEnd, result)
	}
}

func collectBindings(bindings []*ast.Binding, result *[]string, lineStart, lineEnd int) {
	for _, b := range bindings {
		if id, ok := b.Target.(*ast.Identifier); ok {
			if b.Initializer == nil {
				continue
			}
			if int(id.Idx0()) >= lineStart && int(id.Idx0()) < lineEnd {
				continue
			}
			addTo(result, id.Name.String())
		}
	}
}

func addTo(result *[]string, name string) {
	if name == "" {
		return
	}
	for _, v := range *result {
		if v == name {
			return
		}
	}
	*result = append(*result, name)
}
