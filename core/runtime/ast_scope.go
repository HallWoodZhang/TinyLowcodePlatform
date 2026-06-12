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

	offset := lineToOffset(jsCode, line)

	var result []string
	walkStmtList(program.Body, program.DeclarationList, offset, &result)
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

func walkStmtList(body []ast.Statement, decls []*ast.VariableDeclaration, offset int, result *[]string) {
	for _, d := range decls {
		collectBindings(d.List, result)
	}
	for _, s := range body {
		walkNode(s, offset, result)
	}
}

func walkNode(n ast.Node, offset int, result *[]string) {
	if n == nil {
		return
	}
	if int(n.Idx0())-1 > offset {
		return
	}

	switch st := n.(type) {
	case *ast.BlockStatement:
		walkStmtList(st.List, nil, offset, result)
	case *ast.VariableStatement:
		collectBindings(st.List, result)
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
				walkNode(st.Function.Body, offset, result)
			}
		}
	case *ast.IfStatement:
		walkNode(st.Consequent, offset, result)
		walkNode(st.Alternate, offset, result)
	case *ast.WhileStatement, *ast.DoWhileStatement:
		if s, ok := n.(*ast.WhileStatement); ok {
			walkNode(s.Body, offset, result)
		} else if s, ok := n.(*ast.DoWhileStatement); ok {
			walkNode(s.Body, offset, result)
		}
	case *ast.ForStatement:
		if st.Body != nil {
			if vs, ok := st.Initializer.(ast.Node); ok {
				if vstmt, ok := vs.(*ast.VariableStatement); ok {
					collectBindings(vstmt.List, result)
				}
			}
			walkNode(st.Body, offset, result)
		}
	case *ast.TryStatement:
		walkNode(st.Body, offset, result)
		if st.Catch != nil {
			walkNode(st.Catch.Body, offset, result)
		}
		if st.Finally != nil {
			walkNode(st.Finally, offset, result)
		}
	case *ast.SwitchStatement:
		for _, cs := range st.Body {
			walkNode(cs, offset, result)
		}
	case *ast.ExpressionStatement:
		walkExpr(st.Expression, offset, result)
	}
}

func walkExpr(e ast.Expression, offset int, result *[]string) {
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
			walkNode(ex.Body, offset, result)
		}
	case *ast.ArrowFunctionLiteral:
		for _, p := range ex.ParameterList.List {
			if id, ok := p.Target.(*ast.Identifier); ok {
				addTo(result, id.Name.String())
			}
		}
		if ex.Body != nil {
			walkNode(ex.Body, offset, result)
		}
	case *ast.CallExpression:
		walkExpr(ex.Callee, offset, result)
		for _, a := range ex.ArgumentList {
			walkExpr(a, offset, result)
		}
	case *ast.AssignExpression:
		walkExpr(ex.Left, offset, result)
		walkExpr(ex.Right, offset, result)
	case *ast.BinaryExpression:
		walkExpr(ex.Left, offset, result)
		walkExpr(ex.Right, offset, result)
	}
}

func collectBindings(bindings []*ast.Binding, result *[]string) {
	for _, b := range bindings {
		if id, ok := b.Target.(*ast.Identifier); ok {
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
