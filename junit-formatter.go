package main

import (
	"bufio"
	"encoding/xml"
	"io"
	"runtime"
	"strings"

	"github.com/thsnr/go-junit-report/parser"
)

// JUnitTestSuites is a collection of JUnit test suites.
type JUnitTestSuites struct {
	XMLName xml.Name `xml:"testsuites"`
	Suites  []JUnitTestSuite
}

// JUnitTestSuite is a single JUnit test suite which may contain many
// testcases.
type JUnitTestSuite struct {
	XMLName    xml.Name        `xml:"testsuite"`
	Package    string          `xml:"package,attr"`
	Name       string          `xml:"name,attr"`
	Tests      int             `xml:"tests,attr"`
	Failures   int             `xml:"failures,attr"`
	Time       float64         `xml:"time,attr"`
	Properties []JUnitProperty `xml:"properties>property,omitempty"`
	TestCases  []JUnitTestCase
}

// JUnitTestCase is a single test case with its result.
type JUnitTestCase struct {
	XMLName     xml.Name          `xml:"testcase"`
	Classname   string            `xml:"classname,attr"`
	Name        string            `xml:"name,attr"`
	Time        float64           `xml:"time,attr"`
	SkipMessage *JUnitSkipMessage `xml:"skipped,omitempty"`
	Failure     *JUnitFailure     `xml:"failure,omitempty"`
	SystemOut   string            `xml:"system-out"`
}

// JUnitSkipMessage contains the reason why a testcase was skipped.
type JUnitSkipMessage struct {
	Message string `xml:"message,attr"`
}

// JUnitProperty represents a key/value pair used to define properties.
type JUnitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// JUnitFailure contains data related to a failed test.
type JUnitFailure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

// JUnitReportXML writes a JUnit xml representation of the given report to w
// in the format described at http://windyroad.org/dl/Open%20Source/JUnit.xsd
func JUnitReportXML(report *parser.Report, noXMLHeader bool, w io.Writer) error {
	suites := JUnitTestSuites{}

	// convert Report to JUnit test suites
	for _, pkg := range report.Packages {
		pkgname := ""
		classname := pkg.Name
		if idx := strings.LastIndex(classname, "/"); idx > -1 && idx < len(pkg.Name) {
			pkgname = pkg.Name[:idx]
			classname = pkg.Name[idx+1:]
		}

		ts := JUnitTestSuite{
			Package:    strings.Replace(pkgname, "/", ".", -1),
			Name:       classname,
			Tests:      len(pkg.Tests),
			Failures:   0,
			Time:       pkg.Time,
			Properties: []JUnitProperty{},
			TestCases:  []JUnitTestCase{},
		}

		// properties
		ts.Properties = append(ts.Properties, JUnitProperty{"go.version", runtime.Version()})
		if pkg.CoveragePct != "" {
			ts.Properties = append(ts.Properties, JUnitProperty{"coverage.statements.pct", pkg.CoveragePct})
		}

		// individual test cases
		for _, test := range pkg.Tests {
			testCase := JUnitTestCase{
				Classname: strings.Replace(pkg.Name, "/", ".", -1),
				Name:      test.Name,
				Time:      test.Time,
				Failure:   nil,
			}

			if test.Result == parser.FAIL {
				ts.Failures++
				testCase.Failure = &JUnitFailure{
					Message:  "Failed",
					Type:     "",
					Contents: strings.Join(test.Output, "\n"),
				}
			}

			if test.Result == parser.SKIP {
				testCase.SkipMessage = &JUnitSkipMessage{strings.Join(test.Output, "\n")}
			}

			if test.Result == parser.PASS {
				test.Stdout = append(test.Stdout, test.Output...)
			}

			testCase.SystemOut = strings.Join(test.Stdout, "\n")
			ts.TestCases = append(ts.TestCases, testCase)
		}

		suites.Suites = append(suites.Suites, ts)
	}

	// to xml
	bytes, err := xml.MarshalIndent(suites, "", "\t")
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(w)

	if !noXMLHeader {
		writer.WriteString(xml.Header)
	}

	writer.Write(bytes)
	writer.WriteByte('\n')
	writer.Flush()

	return nil
}

func countFailures(tests []parser.Test) (result int) {
	for _, test := range tests {
		if test.Result == parser.FAIL {
			result++
		}
	}
	return
}
