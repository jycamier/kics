package report

import (
	"bytes"
	_ "embed" // used for embedding report static files
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/Checkmarx/kics/pkg/model"
	"github.com/tdewolff/minify/v2"
	minifyCSS "github.com/tdewolff/minify/v2/css"
	minifyHtml "github.com/tdewolff/minify/v2/html"
	minifyJS "github.com/tdewolff/minify/v2/js"
)

var (
	//go:embed template/html/report.tmpl
	htmlTemplate string
	//go:embed template/html/report.css
	cssTemplate string
	//go:embed template/html/report.js
	jsTemplate string
	//go:embed template/html/github.svg
	githubSVG string
	//go:embed template/html/info.svg
	infoSVG string
	//go:embed template/html/vulnerability_fill.svg
	vulnerabilityFillSVG string
	//go:embed template/html/vulnerability_out.svg
	vulnerabilityOutSVG string
)

const (
	textHTML = "text/html"
)

var svgMap = map[string]string{
	"github.svg":             githubSVG,
	"info.svg":               infoSVG,
	"vulnerability_fill.svg": vulnerabilityFillSVG,
	"vulnerability_out.svg":  vulnerabilityOutSVG,
}

func includeSVG(name string) template.HTML {
	return template.HTML(svgMap[name]) //nolint
}

func includeCSS(name string) template.HTML {
	minifier := minify.New()
	minifier.AddFunc("text/css", minifyCSS.Minify)
	cssMinified, err := minifier.String("text/css", cssTemplate)
	if err != nil {
		return ""
	}
	return template.HTML("<style>" + cssMinified + "</style>") //nolint
}

func includeJS(name string) template.HTML {
	minifier := minify.New()
	minifier.AddFunc("text/javascript", minifyJS.Minify)
	jsMinified, err := minifier.String("text/javascript", jsTemplate)
	if err != nil {
		return ""
	}
	return template.HTML("<script>" + jsMinified + "</script>") //nolint
}

func getPaths(paths []string) string {
	return strings.Join(paths, ", ")
}

func getPlatforms(queries model.VulnerableQuerySlice) string {
	platforms := make([]string, 0)
	alreadyAdded := make(map[string]string)
	for idx := range queries {
		if _, ok := alreadyAdded[queries[idx].Platform]; !ok {
			alreadyAdded[queries[idx].Platform] = ""
			platforms = append(platforms, queries[idx].Platform)
		}
	}
	return strings.Join(platforms, ", ")
}

// PrintHTMLReport creates a report file on HTML format
func PrintHTMLReport(path, filename string, body interface{}) error {
	if !strings.HasSuffix(filename, ".html") {
		filename += ".html"
	}

	templateFuncs["includeSVG"] = includeSVG
	templateFuncs["includeCSS"] = includeCSS
	templateFuncs["includeJS"] = includeJS
	templateFuncs["getPaths"] = getPaths
	templateFuncs["getPlatforms"] = getPlatforms

	fullPath := filepath.Join(path, filename)
	t := template.Must(template.New("report.tmpl").Funcs(templateFuncs).Parse(htmlTemplate))

	f, err := os.OpenFile(filepath.Clean(fullPath), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer closeFile(fullPath, filename, f)
	var buffer bytes.Buffer

	err = t.Execute(&buffer, body)
	if err != nil {
		return err
	}
	minifier := minify.New()
	minifier.AddFunc(textHTML, minifyHtml.Minify)
	minifier.Add(textHTML, &minifyHtml.Minifier{
		KeepDocumentTags: true,
		KeepEndTags:      true,
		KeepQuotes:       true,
	})

	minifierWriter := minifier.Writer(textHTML, f)
	defer minifierWriter.Close()

	_, err = minifierWriter.Write(buffer.Bytes())
	return err
}
