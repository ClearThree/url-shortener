package main

import (
	"go/ast"

	"github.com/gostaticanalysis/forcetypeassert"
	"github.com/orijtech/httperroryzer"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/gofix"
	"golang.org/x/tools/go/analysis/passes/hostport"
	"golang.org/x/tools/go/analysis/passes/httpmux"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/pkgfact"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"
	"golang.org/x/tools/go/analysis/passes/waitgroup"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

// ExitAnalyzer - The instance of Analyzer to check the os.Exit calls existence.
var ExitAnalyzer = &analysis.Analyzer{
	Name: "exit",
	Doc:  "forbids calls to os.Exit in main package",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		if pass.Pkg.Name() != "main" {
			continue
		}
		for _, decl := range file.Decls {
			function, ok := decl.(*ast.FuncDecl)
			if !ok || function.Name.Name != "main" {
				continue
			}
			ast.Inspect(function, func(node ast.Node) bool {
				call, callOk := node.(*ast.CallExpr)
				if !callOk {
					return true
				}
				selector, selectorOk := call.Fun.(*ast.SelectorExpr)
				if !selectorOk {
					return true
				}
				xIdent, xIdentOk := selector.X.(*ast.Ident)
				if !xIdentOk {
					return true
				}
				if xIdent.Name == "os" && selector.Sel.Name == "Exit" {
					pass.Reportf(call.Pos(), "direct call to os.Exit is forbidden in main")
				}
				return true
			})
		}
	}
	return nil, nil
}

func main() {
	staticcheckAnalyzersLength := len(staticcheck.Analyzers)
	myChecks := make([]*analysis.Analyzer, staticcheckAnalyzersLength+len(stylecheck.Analyzers))
	for i, v := range staticcheck.Analyzers {
		myChecks[i] = v.Analyzer
	}
	for i, v := range stylecheck.Analyzers {
		myChecks[i+staticcheckAnalyzersLength] = v.Analyzer
	}
	myChecks = append(myChecks, appends.Analyzer)
	myChecks = append(myChecks, asmdecl.Analyzer)
	myChecks = append(myChecks, assign.Analyzer)
	myChecks = append(myChecks, atomic.Analyzer)
	myChecks = append(myChecks, atomicalign.Analyzer)
	myChecks = append(myChecks, bools.Analyzer)
	myChecks = append(myChecks, buildssa.Analyzer)
	myChecks = append(myChecks, buildtag.Analyzer)
	myChecks = append(myChecks, cgocall.Analyzer)
	myChecks = append(myChecks, composite.Analyzer)
	myChecks = append(myChecks, copylock.Analyzer)
	myChecks = append(myChecks, ctrlflow.Analyzer)
	myChecks = append(myChecks, deepequalerrors.Analyzer)
	myChecks = append(myChecks, defers.Analyzer)
	myChecks = append(myChecks, directive.Analyzer)
	myChecks = append(myChecks, errorsas.Analyzer)
	myChecks = append(myChecks, fieldalignment.Analyzer)
	myChecks = append(myChecks, findcall.Analyzer)
	myChecks = append(myChecks, framepointer.Analyzer)
	myChecks = append(myChecks, gofix.Analyzer)
	myChecks = append(myChecks, hostport.Analyzer)
	myChecks = append(myChecks, httpmux.Analyzer)
	myChecks = append(myChecks, httpresponse.Analyzer)
	myChecks = append(myChecks, ifaceassert.Analyzer)
	myChecks = append(myChecks, inspect.Analyzer)
	myChecks = append(myChecks, loopclosure.Analyzer)
	myChecks = append(myChecks, lostcancel.Analyzer)
	myChecks = append(myChecks, nilfunc.Analyzer)
	myChecks = append(myChecks, nilness.Analyzer)
	myChecks = append(myChecks, pkgfact.Analyzer)
	myChecks = append(myChecks, printf.Analyzer)
	myChecks = append(myChecks, reflectvaluecompare.Analyzer)
	myChecks = append(myChecks, shadow.Analyzer)
	myChecks = append(myChecks, shift.Analyzer)
	myChecks = append(myChecks, sigchanyzer.Analyzer)
	myChecks = append(myChecks, slog.Analyzer)
	myChecks = append(myChecks, sortslice.Analyzer)
	myChecks = append(myChecks, stdmethods.Analyzer)
	myChecks = append(myChecks, stdversion.Analyzer)
	myChecks = append(myChecks, stringintconv.Analyzer)
	myChecks = append(myChecks, structtag.Analyzer)
	myChecks = append(myChecks, testinggoroutine.Analyzer)
	myChecks = append(myChecks, timeformat.Analyzer)
	myChecks = append(myChecks, unmarshal.Analyzer)
	myChecks = append(myChecks, unreachable.Analyzer)
	myChecks = append(myChecks, unsafeptr.Analyzer)
	myChecks = append(myChecks, unusedresult.Analyzer)
	myChecks = append(myChecks, unusedwrite.Analyzer)
	myChecks = append(myChecks, usesgenerics.Analyzer)
	myChecks = append(myChecks, waitgroup.Analyzer)

	myChecks = append(myChecks, httperroryzer.Analyzer)
	myChecks = append(myChecks, forcetypeassert.Analyzer)

	myChecks = append(myChecks, ExitAnalyzer)

	multichecker.Main(myChecks...)
}
