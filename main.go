// taggo.go
//
// Create Exuberant-Ctags compatible tags files for Go source.
//
// Arjen Laarhoven, December 2011

package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	TAG_FILE_FORMAT    = "!_TAG_FILE_FORMAT\t2"
	TAG_FILE_SORTED    = "!_TAG_FILE_SORTED\t1"
	TAG_PROGRAM_AUTHOR = "!_TAG_PROGRAM_AUTHOR\tArjen Laarhoven"
	TAG_PROGRAM_NAME   = "!_TAG_PROGRAM_NAME\ttaggo"
	TAG_PROGRAM_URL    = "!_TAG_PROGRAM_URL\thttps://github.com/ArjenL/taggo"

	CLASS  = 'c' // Interface ('class')
	CONST  = 'd' // Constant ('#define')
	FUNC   = 'f' // Function
	MEMBER = 'm' // Structure member
	STRUCT = 's' // Structure
	TYPE   = 't' // Type
	VAR    = 'v' // Variable
)

var (
	recurseSubdirs = flag.Bool("recurse", false, "Recurse into given subdirectories")

	files = make([]string, 0)
	tags  = make([]string, 0)
)

func main() {
	flag.Parse()

	// Parse the given files.
	fset := token.NewFileSet()
	pkgs, _ := parseFiles(fset)

	// Extract toplevel declaration information from the packages.
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			handleDecls(fset, file.Decls)
		}
	}

	// Output the tags sorted alphabetically.
	sort.Strings(tags)
	printTagsHeader()
	for _, t := range tags {
		fmt.Printf("%s\n", t)
	}
}

func handleDecls(fset *token.FileSet, decls []ast.Decl) {
	for _, decl := range decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			funcDecl(fset, decl)
		case *ast.GenDecl:
			genDecl(fset, decl)
		}
	}
}

// Handle a function declaration.
func funcDecl(fset *token.FileSet, decl *ast.FuncDecl) {
	var recvType string
	if decl.Recv != nil {
		// Method definition.  There's always only one receiver.
		recvType = "class:" + typeName(decl.Recv.List[0].Type)
	} else {
		// Normal function
		recvType = ""
	}
	emitTag(decl.Name.Name, decl.Pos(), fset, FUNC, recvType)
}

// Handle CONST, TYPE or VAR declarations
func genDecl(fset *token.FileSet, decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		switch nt := spec.(type) {
		case *ast.TypeSpec:
			typeSpec(fset, nt)

		case *ast.ValueSpec:
			for _, ident := range nt.Names {
				var kind rune
				switch decl.Tok {
				case token.CONST:
					kind = CONST
				case token.VAR:
					kind = VAR
				}
				emitTag(ident.Name, ident.NamePos, fset, kind, "")
			}
		}
	}
}

// Handle structures/"classes" (interfaces)
func typeSpec(fset *token.FileSet, spec *ast.TypeSpec) {
	switch st := spec.Type.(type) {
	case *ast.StructType:
		emitTag(spec.Name.Name, st.Pos(), fset, STRUCT, "")
		for _, f := range st.Fields.List {
			for _, m := range f.Names {
				emitTag(m.Name, m.Pos(), fset, MEMBER, fmt.Sprintf("struct:%s", spec.Name.Name))
			}
		}
	case *ast.InterfaceType:
		emitTag(spec.Name.Name, st.Pos(), fset, CLASS, "")
		for _, f := range st.Methods.List {
			for _, m := range f.Names {
				emitTag(m.Name, m.Pos(), fset, FUNC, fmt.Sprintf("class:%s", spec.Name.Name))
			}
		}
	default:
		emitTag(spec.Name.Name, st.Pos(), fset, TYPE, "")
	}

}

// Add tag to the map of tags
func emitTag(tag string, pos token.Pos, fset *token.FileSet, kind rune, extra string) {
	p := fset.Position(pos)
	searchString := contentOfLine(p.Line, p.Filename)
	tags = append(tags, fmt.Sprintf("%s\t%s\t/^%s$/;\"\t%c\t%s", tag, p.Filename, searchString, kind, extra))
}

// Return the content from the given line number of the given file.
func contentOfLine(line int, file string) []byte {
	var cl []byte
	ln := 1

	f, err := os.Open(file)
	if err != nil {
		// Just skip over the file when we can't open it
		return []byte("")
	}
	defer f.Close()

	r := bufio.NewReader(f)

	for {
		cl, err = r.ReadBytes('\n')
		if err == io.EOF && ln < line {
			// File has fewer lines than <line>
			return []byte("")
		}

		// Are we there yet?
		if ln == line {
			// Remove the trailing newline
			if len(cl) > 0 {
				cl = cl[:len(cl)-1]
			}
			return cl
		}

		ln++
	}
}

// Parse the files given on the command-line
func parseFiles(fset *token.FileSet) (map[string]*ast.Package, error) {
	// Expand the content of given subdirs into a list of files.
	for _, fn := range flag.Args() {
		fi, err := os.Stat(fn)
		if err != nil {
			continue // Skip unreadable or nonexistent files
		}

		if fi.Mode().IsRegular() && filepath.Ext(fn) == ".go" {
			files = append(files, fn)
		}

		if *recurseSubdirs && fi.IsDir() {
			filepath.Walk(fi.Name(), walker)
		}
	}

	var pkgs = make(map[string]*ast.Package)
	var first error

	for _, filename := range files {
		if src, err := parser.ParseFile(fset, filename, nil, parser.SpuriousErrors); err == nil {
			name := src.Name.Name
			pkg, found := pkgs[name]
			if !found {
				pkg = &ast.Package{
					Name:  name,
					Files: make(map[string]*ast.File),
				}
				pkgs[name] = pkg
			}
			pkg.Files[filename] = src
		} else if first == nil {
			first = err
		}
	}
	return pkgs, first
}

// Walker function for filepath.Walk
func walker(path string, fi os.FileInfo, err error) error {
	if fi.Mode()&os.ModeType == 0 && strings.HasSuffix(fi.Name(), ".go") {
		files = append(files, path)
	}
	return nil
}

// Output the tag header to standard output
func printTagsHeader() {
	fmt.Println(TAG_FILE_FORMAT)
	fmt.Println(TAG_FILE_SORTED)
	fmt.Println(TAG_PROGRAM_AUTHOR)
	fmt.Println(TAG_PROGRAM_NAME)
	fmt.Println(TAG_PROGRAM_URL)
}

// Return the name of the type as string.  This routine is borrowed from the
// error.go file of the gofix command.
func typeName(typ ast.Expr) string {
	if p, ok := typ.(*ast.StarExpr); ok {
		typ = p.X
	}
	id, ok := typ.(*ast.Ident)
	if ok {
		return id.Name
	}
	sel, ok := typ.(*ast.SelectorExpr)
	if ok {
		return typeName(sel.X) + "." + sel.Sel.Name
	}
	return ""
}
