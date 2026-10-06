package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// stmtCallName returns the called function/method name when s is a plain
// call statement (rm.setupTray(app), prepareNativeWindow(win, mode), ...).
func stmtCallName(s ast.Stmt) string {
	es, ok := s.(*ast.ExprStmt)
	if !ok {
		return ""
	}
	ce, ok := es.X.(*ast.CallExpr)
	if !ok {
		return ""
	}
	switch f := ce.Fun.(type) {
	case *ast.SelectorExpr:
		return f.Sel.Name
	case *ast.Ident:
		return f.Name
	}
	return ""
}

// 이 테스트는 구조 회귀 방지용이다: v1.0.8에서 setupTray를 이벤트 루프 시작 뒤로 옮겼는데,
// 이후 오래된 main.go를 덮어쓰면서 조용히 되돌려졌고 v1.0.9가 그대로 배포됐다.
func TestMainStartupOrder(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", nil, 0)
	if err != nil {
		t.Fatalf("main.go 파싱 실패: %v", err)
	}

	var mainFn *ast.FuncDecl
	for _, d := range file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "main" {
			mainFn = fd
		}
	}
	if mainFn == nil {
		t.Fatal("main()을 찾을 수 없음")
	}

	Scenario(t, "GIVEN main.go WHEN 시작 순서를 검사 THEN setupTray는 main() 본문에서 직접 호출되지 않는다 (이벤트 루프 시작 전 호출 방지)", func(t *testing.T) {
		for _, s := range mainFn.Body.List {
			if stmtCallName(s) == "setupTray" {
				t.Errorf("main()이 setupTray를 Run() 앞에서 직접 호출함 — 앱 시작 훅(SetOnStarted) 안으로 옮겨야 함")
			}
		}
	})

	Scenario(t, "GIVEN main.go WHEN 앱 시작 훅을 검사 THEN prepareNativeWindow가 setupTray보다 먼저 호출된다 (창 핸들이 먼저 있어야 함)", func(t *testing.T) {
		prepareIdx, trayIdx, hooks := -1, -1, 0
		ast.Inspect(mainFn.Body, func(n ast.Node) bool {
			lit, ok := n.(*ast.FuncLit)
			if !ok {
				return true
			}
			for i, s := range lit.Body.List {
				switch stmtCallName(s) {
				case "prepareNativeWindow":
					prepareIdx = i
				case "setupTray":
					trayIdx = i
					hooks++
				}
			}
			return true
		})
		if hooks != 1 {
			t.Fatalf("setupTray는 앱 시작 훅 안에서 정확히 한 번 호출돼야 하는데 %d번", hooks)
		}
		if prepareIdx < 0 || prepareIdx > trayIdx {
			t.Errorf("prepareNativeWindow(%d)가 setupTray(%d)보다 먼저여야 함", prepareIdx, trayIdx)
		}
	})
}
