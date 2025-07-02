/*
Package staticlint is a multichecker that combines static analyzers from:

- "golang.org/x/tools/go/analysis/passes". Please refer to the documentation of package to read more about all the analyzers;

- all SA analyzers from "honnef.co/go/tools/staticcheck". Please refer to the documentation of staticcheck package to read more about all the analyzers [staticcheck];

- all ST analyzers from "honnef.co/go/tools/stylecheck". Please refer to the documentation of staticcheck package to read more about all the analyzers [stylecheck];

- "github.com/orijtech/httperroryzer" to catch invalid uses of http.Error without a return statement which can cause expected bugs [httperroryzer]. Use flag -httperroryzer to control this analyzer.

- "github.com/gostaticanalysis/forcetypeassert" finds type assertions which did forcely without check if the assertion failed [forcetypeassert]. Use flag -forcetypeassert to control this analyzer.

- a custom "exit" analyzer that checks the calls of os.Exit function in the main function of main package. Use flag -exit to control this analyzer.

To use this multichecker, build it with from the cmd/staticlint directory using go build. Then run the executable, specifying the directories you want to analyze.
*/
package main

import (
	"github.com/gostaticanalysis/forcetypeassert"
	"github.com/orijtech/httperroryzer"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

var _ = staticcheck.Analyzers
var _ = stylecheck.Analyzers
var _ = httperroryzer.Analyzer
var _ = forcetypeassert.Analyzer
