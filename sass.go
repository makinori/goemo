package goemo

import (
	"errors"
	"path/filepath"
	"strings"

	sass "github.com/bep/godartsass/v2"
)

var (
	scssTranspiler *sass.Transpiler
)

type SassImport struct {
	name     string
	Filename string
	Content  string
}

type embeddedImportResolver struct {
	imports []SassImport
}

func (importResolver embeddedImportResolver) CanonicalizeURL(url string) (string, error) {
	return "embed://" + url, nil
}

func (importResolver embeddedImportResolver) Load(canonicalizedURL string) (sass.Import, error) {
	if !strings.HasPrefix(canonicalizedURL, "embed://") {
		return sass.Import{}, errors.New("invalid url")
	}

	name := canonicalizedURL[8:]

	// i dont know and i dont care
	nameSplit := strings.Split(name, "/")
	name = nameSplit[len(nameSplit)-1]

	for _, sassImport := range importResolver.imports {
		if sassImport.name == name {
			sourceSyntax := sass.SourceSyntaxSCSS
			switch {
			case strings.HasPrefix(sassImport.Filename, ".sass"):
				sourceSyntax = sass.SourceSyntaxSASS
			case strings.HasPrefix(sassImport.Filename, ".css"):
				sourceSyntax = sass.SourceSyntaxCSS
			}

			return sass.Import{
				SourceSyntax: sourceSyntax,
				Content:      sassImport.Content,
			}, nil
		}
	}

	return sass.Import{}, errors.New("failed to find " + name)
}

func RenderSCSS(source string, imports ...SassImport) (string, error) {
	if scssTranspiler == nil {
		return "", errors.New("scss transpiler not initialized")
	}

	for i := range imports {
		filename := imports[i].Filename
		ext := filepath.Ext(filename)
		imports[i].name = filename[:len(filename)-len(ext)]
	}

	res, err := scssTranspiler.Execute(sass.Args{
		ImportResolver: embeddedImportResolver{
			imports: imports,
		},
		Source:          source,
		OutputStyle:     sass.OutputStyleCompressed,
		SourceSyntax:    sass.SourceSyntaxSCSS,
		EnableSourceMap: false,
	})

	if err != nil {
		return "", err
	}

	return res.CSS, nil
}

func InitSCSS(options sass.Options) error {
	var err error
	scssTranspiler, err = sass.Start(options)
	return err
}
