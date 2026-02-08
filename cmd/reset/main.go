package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ResetGenerator отвечает за генерацию методов Reset() для структур
type ResetGenerator struct {
	fset       *token.FileSet
	structures []StructureInfo
	rootPath   string
}

// StructureInfo хранит информацию о структуре для генерации метода Reset()
type StructureInfo struct {
	Name     string
	Package  string
	Fields   []FieldInfo
	FilePath string
	DirPath  string
}

// FieldInfo хранит информацию о поле структуры
type FieldInfo struct {
	Name     string
	TypeExpr ast.Expr
	TypeStr  string
}

// NewResetGenerator создаёт новый экземпляр генератора
func NewResetGenerator(rootPath string) *ResetGenerator {
	return &ResetGenerator{
		fset:     token.NewFileSet(),
		rootPath: rootPath,
	}
}

// main является точкой входа в утилиту
func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run . <project-root-path>")
	}

	rootPath := os.Args[1]
	generator := NewResetGenerator(rootPath)

	if err := generator.Generate(); err != nil {
		log.Fatalf("Error generating reset methods: %v", err)
	}

	fmt.Println("Reset methods generated successfully!")
}

// Generate запускает полный процесс генерации методов Reset()
func (g *ResetGenerator) Generate() error {
	// 1. Сканирование директорий и поиск Go файлов
	if err := g.scanDirectories(); err != nil {
		return fmt.Errorf("failed to scan directories: %w", err)
	}

	fmt.Printf("Found %d structures with // generate:reset comment\n", len(g.structures))
	for _, structInfo := range g.structures {
		fmt.Printf("  - %s in package %s (%s)\n", structInfo.Name, structInfo.Package, structInfo.DirPath)
	}

	// 2. Генерация и запись файлов
	if err := g.generateResetFiles(); err != nil {
		return fmt.Errorf("failed to generate reset files: %w", err)
	}

	return nil
}

// scanDirectories сканирует все директории и ищет Go файлы
func (g *ResetGenerator) scanDirectories() error {
	return filepath.Walk(g.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Пропускаем директории, которые не нужно сканировать
		if info.IsDir() {
			dirName := filepath.Base(path)
			if dirName == "vendor" || dirName == ".git" || dirName == "cmd" && !strings.Contains(path, "cmd/reset") {
				return filepath.SkipDir
			}
			return nil
		}

		// Обрабатываем только Go файлы
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			return g.parseFile(path)
		}

		return nil
	})
}

// parseFile парсит Go файл и ищет структуры с комментарием // generate:reset
func (g *ResetGenerator) parseFile(filePath string) error {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Парсим файл
	file, err := parser.ParseFile(g.fset, filePath, src, parser.ParseComments)
	if err != nil {
		return err
	}

	// Ищем структуры с нужным комментарием
	ast.Inspect(file, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			if g.hasResetComment(genDecl.Doc) {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							info := g.extractStructureInfo(file, typeSpec, structType, filePath)
							g.structures = append(g.structures, info)
						}
					}
				}
			}
		}
		return true
	})

	return nil
}

// hasResetComment проверяет, есть ли у объявления комментарий // generate:reset
func (g *ResetGenerator) hasResetComment(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}

	for _, comment := range doc.List {
		if strings.Contains(comment.Text, "// generate:reset") {
			return true
		}
	}
	return false
}

// extractStructureInfo извлекает информацию о структуре
func (g *ResetGenerator) extractStructureInfo(file *ast.File, typeSpec *ast.TypeSpec, structType *ast.StructType, filePath string) StructureInfo {
	info := StructureInfo{
		Name:     typeSpec.Name.Name,
		Package:  file.Name.Name,
		FilePath: filePath,
		DirPath:  filepath.Dir(filePath),
	}

	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			fieldInfo := FieldInfo{
				Name:     name.Name,
				TypeExpr: field.Type,
				TypeStr:  g.getTypeString(field.Type),
			}
			info.Fields = append(info.Fields, fieldInfo)
		}
	}

	return info
}

// getTypeString получает строковое представление типа
func (g *ResetGenerator) getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + g.getTypeString(t.X)
	case *ast.ArrayType:
		return "[]" + g.getTypeString(t.Elt)
	case *ast.MapType:
		return "map[" + g.getTypeString(t.Key) + "]" + g.getTypeString(t.Value)
	case *ast.SelectorExpr:
		return g.getTypeString(t.X) + "." + t.Sel.Name
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// generateResetFiles генерирует файлы с методами Reset()
func (g *ResetGenerator) generateResetFiles() error {
	// Группировка структур по директориям
	dirMap := make(map[string][]StructureInfo)
	for _, structInfo := range g.structures {
		dirMap[structInfo.DirPath] = append(dirMap[structInfo.DirPath], structInfo)
	}

	// Генерация файлов для каждой директории
	for dirPath, structures := range dirMap {
		fmt.Printf("Generating reset file for directory: %s\n", dirPath)
		if err := g.generateResetFile(dirPath, structures); err != nil {
			return fmt.Errorf("failed to generate reset file for directory %s: %w", dirPath, err)
		}
	}

	return nil
}

// generateResetFile генерирует файл reset.gen.go для указанной директории
func (g *ResetGenerator) generateResetFile(dirPath string, structures []StructureInfo) error {
	var builder strings.Builder

	// Заголовок файла
	builder.WriteString("// Code generated by reset generator. DO NOT EDIT.\n")
	builder.WriteString("//go:generate go run ../../cmd/reset\n\n")

	// Определяем имя пакета из первой структуры
	pkgName := structures[0].Package
	builder.WriteString(fmt.Sprintf("package %s\n\n", pkgName))

	// Генерация методов для каждой структуры
	for _, structInfo := range structures {
		method := g.generateResetMethod(structInfo)
		builder.WriteString(method)
	}

	content := builder.String()
	formatted, err := format.Source([]byte(content))
	if err != nil {
		return err
	}

	filePath := filepath.Join(dirPath, "reset.gen.go")
	return os.WriteFile(filePath, formatted, 0644)
}

// generateResetMethod генерирует код метода Reset() для структуры
func (g *ResetGenerator) generateResetMethod(structInfo StructureInfo) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("func (s *%s) Reset() {\n", structInfo.Name))
	builder.WriteString("    if s == nil {\n")
	builder.WriteString("        return\n")
	builder.WriteString("    }\n\n")

	for _, field := range structInfo.Fields {
		code := g.generateFieldReset(field)
		if code != "" {
			builder.WriteString("    " + code + "\n")
		}
	}

	builder.WriteString("}\n\n")
	return builder.String()
}

// generateFieldReset генерирует код сброса для поля
func (g *ResetGenerator) generateFieldReset(field FieldInfo) string {
	typeStr := field.TypeStr

	// Примитивные типы
	if g.isPrimitiveType(typeStr) {
		return g.generatePrimitiveReset(field)
	}

	// Слайсы
	if strings.HasPrefix(typeStr, "[]") {
		return fmt.Sprintf("s.%s = s.%s[:0]", field.Name, field.Name)
	}

	// Мапы
	if strings.HasPrefix(typeStr, "map[") {
		return fmt.Sprintf("clear(s.%s)", field.Name)
	}

	// Указатели
	if strings.HasPrefix(typeStr, "*") {
		return g.generatePointerReset(field)
	}

	return ""
}

// isPrimitiveType проверяет, является ли тип примитивным
func (g *ResetGenerator) isPrimitiveType(typeStr string) bool {
	primitiveTypes := []string{
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "string", "bool",
	}

	for _, pt := range primitiveTypes {
		if typeStr == pt {
			return true
		}
	}
	return false
}

// generatePrimitiveReset генерирует код для примитивных типов
func (g *ResetGenerator) generatePrimitiveReset(field FieldInfo) string {
	typeStr := field.TypeStr

	switch {
	case strings.Contains(typeStr, "int"):
		return fmt.Sprintf("s.%s = 0", field.Name)
	case strings.Contains(typeStr, "float"):
		return fmt.Sprintf("s.%s = 0.0", field.Name)
	case typeStr == "string":
		return fmt.Sprintf("s.%s = \"\"", field.Name)
	case typeStr == "bool":
		return fmt.Sprintf("s.%s = false", field.Name)
	}

	return ""
}

// generatePointerReset генерирует код для указателей
func (g *ResetGenerator) generatePointerReset(field FieldInfo) string {
	underlyingType := strings.TrimPrefix(field.TypeStr, "*")

	if g.isPrimitiveType(underlyingType) {
		switch {
		case strings.Contains(underlyingType, "int"):
			return fmt.Sprintf("if s.%s != nil {\n        *s.%s = 0\n    }", field.Name, field.Name)
		case strings.Contains(underlyingType, "float"):
			return fmt.Sprintf("if s.%s != nil {\n        *s.%s = 0.0\n    }", field.Name, field.Name)
		case underlyingType == "string":
			return fmt.Sprintf("if s.%s != nil {\n        *s.%s = \"\"\n    }", field.Name, field.Name)
		case underlyingType == "bool":
			return fmt.Sprintf("if s.%s != nil {\n        *s.%s = false\n    }", field.Name, field.Name)
		}
	}

	// Для указателей на структуры
	return fmt.Sprintf("if s.%s != nil {\n        s.%s.Reset()\n    }", field.Name, field.Name)
}
