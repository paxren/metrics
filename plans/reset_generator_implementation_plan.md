# План реализации утилиты генерации методов Reset()

## Этап 1: Создание базовой структуры

### 1.1 Создание директории и основного файла
- Создать директорию `cmd/reset/`
- Создать файл `main.go` с базовой структурой
- Добавить импорт необходимых пакетов:
  ```go
  import (
      "fmt"
      "go/ast"
      "go/parser"
      "go/token"
      "go/types"
      "os"
      "path/filepath"
      "strings"
      "go/format"
      "go/packages"
  )
  ```

### 1.2 Определение структур данных
```go
type ResetGenerator struct {
    fset      *token.FileSet
    packages  []*packages.Package
    structures []StructureInfo
}

type StructureInfo struct {
    Name       string
    Package    string
    Fields     []FieldInfo
    FilePath   string
}

type FieldInfo struct {
    Name     string
    Type     types.Type
    TypeExpr ast.Expr
    Tags     string
}
```

## Этап 2: Создание примеров структур

### 2.1 Примеры в internal/models/reset_examples.go
```go
package models

// generate:reset
type SimpleStruct struct {
    IntField    int
    StringField string
    BoolField   bool
    FloatField  float64
}

// generate:reset
type ComplexStruct struct {
    IntSlice    []int
    StringSlice []string
    IntMap      map[string]int
    StringMap   map[string]string
    IntPtr      *int
    StringPtr   *string
}

// generate:reset
type NestedStruct struct {
    Simple  SimpleStruct
    Complex *ComplexStruct
    Mixed   interface{ Reset() }
}
```

### 2.2 Примеры в internal/repository/reset_examples.go
```go
package repository

// generate:reset
type StorageConfig struct {
    Host     string
    Port     int
    Database string
    Options  map[string]interface{}
}

// generate:reset
type CacheEntry struct {
    Key       string
    Value     []byte
    ExpiresAt int64
    Metadata  map[string]string
}
```

### 2.3 Примеры в internal/handler/reset_examples.go
```go
package handler

// generate:reset
type RequestData struct {
    Method  string
    Path    string
    Headers map[string][]string
    Body    []byte
    Query   map[string]string
}

// generate:reset
type ResponseData struct {
    StatusCode int
    Headers    map[string]string
    Body       []byte
    Error      error
}
```

## Этап 3: Реализация сканера пакетов

### 3.1 Функция сканирования пакетов
```go
func (g *ResetGenerator) ScanPackages(rootPath string) error {
    cfg := &packages.Config{
        Mode:  packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypes | packages.NeedFiles,
        Tests: false,
    }
    
    pkgs, err := packages.Load(cfg, rootPath + "/...")
    if err != nil {
        return err
    }
    
    g.packages = pkgs
    return nil
}
```

### 3.2 Функция поиска структур с комментарием
```go
func (g *ResetGenerator) FindStructuresWithResetComment() error {
    for _, pkg := range g.packages {
        for _, file := range pkg.Syntax {
            g.scanFileForStructures(pkg, file)
        }
    }
    return nil
}

func (g *ResetGenerator) scanFileForStructures(pkg *packages.Package, file *ast.File) {
    ast.Inspect(file, func(n ast.Node) bool {
        if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
            for _, spec := range genDecl.Specs {
                if typeSpec, ok := spec.(*ast.TypeSpec); ok {
                    if structType, ok := typeSpec.Type.(*ast.StructType); ok {
                        if g.hasResetComment(genDecl.Doc) {
                            info := g.extractStructureInfo(pkg, typeSpec, structType)
                            g.structures = append(g.structures, info)
                        }
                    }
                }
            }
        }
        return true
    })
}
```

## Этап 4: Реализация генератора кода

### 4.1 Базовая структура генератора
```go
func (g *ResetGenerator) GenerateResetMethod(structInfo StructureInfo) string {
    var builder strings.Builder
    
    builder.WriteString(fmt.Sprintf("func (s *%s) Reset() {\n", structInfo.Name))
    builder.WriteString("    if s == nil {\n")
    builder.WriteString("        return\n")
    builder.WriteString("    }\n\n")
    
    for _, field := range structInfo.Fields {
        code := g.generateFieldReset(field)
        builder.WriteString("    " + code + "\n")
    }
    
    builder.WriteString("}\n\n")
    return builder.String()
}
```

### 4.2 Обработчики для разных типов

#### Примитивные типы
```go
func (g *ResetGenerator) generatePrimitiveReset(field FieldInfo) string {
    switch t := field.Type.(type) {
    case *types.Basic:
        switch t.Kind() {
        case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
            return fmt.Sprintf("s.%s = 0", field.Name)
        case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
            return fmt.Sprintf("s.%s = 0", field.Name)
        case types.Float32, types.Float64:
            return fmt.Sprintf("s.%s = 0.0", field.Name)
        case types.String:
            return fmt.Sprintf("s.%s = \"\"", field.Name)
        case types.Bool:
            return fmt.Sprintf("s.%s = false", field.Name)
        }
    }
    return ""
}
```

#### Слайсы
```go
func (g *ResetGenerator) generateSliceReset(field FieldInfo) string {
    return fmt.Sprintf("s.%s = s.%s[:0]", field.Name, field.Name)
}
```

#### Мапы
```go
func (g *ResetGenerator) generateMapReset(field FieldInfo) string {
    return fmt.Sprintf("clear(s.%s)", field.Name)
}
```

#### Указатели
```go
func (g *ResetGenerator) generatePointerReset(field FieldInfo) string {
    if ptr, ok := field.Type.(*types.Pointer); ok {
        if basic, ok := ptr.Elem().(*types.Basic); ok {
            switch basic.Kind() {
            case types.String:
                return fmt.Sprintf("if s.%s != nil {\n        *s.%s = \"\"\n    }", field.Name, field.Name)
            case types.Int, types.Int64:
                return fmt.Sprintf("if s.%s != nil {\n        *s.%s = 0\n    }", field.Name, field.Name)
            // ... другие примитивные типы
            }
        }
    }
    return ""
}
```

#### Вложенные структуры
```go
func (g *ResetGenerator) generateStructReset(field FieldInfo) string {
    return fmt.Sprintf("if resetter, ok := s.%s.(interface{ Reset() }); ok && s.%s != nil {\n        resetter.Reset()\n    }", field.Name, field.Name)
}
```

## Этап 5: Реализация записи файлов

### 5.1 Функция генерации файла
```go
func (g *ResetGenerator) GenerateResetFile(pkgPath string, structures []StructureInfo) error {
    var builder strings.Builder
    
    builder.WriteString(fmt.Sprintf("// Code generated by reset generator. DO NOT EDIT.\n"))
    builder.WriteString(fmt.Sprintf("//go:generate go run ../../cmd/reset\n\n"))
    builder.WriteString(fmt.Sprintf("package %s\n\n", filepath.Base(pkgPath)))
    
    for _, structInfo := range structures {
        method := g.GenerateResetMethod(structInfo)
        builder.WriteString(method)
    }
    
    content := builder.String()
    formatted, err := format.Source([]byte(content))
    if err != nil {
        return err
    }
    
    filePath := filepath.Join(pkgPath, "reset.gen.go")
    return os.WriteFile(filePath, formatted, 0644)
}
```

## Этап 6: Тестирование

### 6.1 Модульные тесты
- Тесты для каждого типа обработчика
- Тесты для сканера пакетов
- Тесты для генератора кода

### 6.2 Интеграционные тесты
- Тесты на полных примерах
- Проверка корректности сгенерированного кода
- Тесты компиляции сгенерированного кода

## Этап 7: Документация

### 7.1 README для утилиты
- Описание назначения
- Инструкция по использованию
- Примеры структур и сгенерированного кода

### 7.2 Комментарии в коде
- Подробные комментарии для каждой функции
- Объяснение алгоритмов обработки типов

## Порядок выполнения

1. Создать базовую структуру утилиты
2. Создать примеры структур в разных пакетах
3. Реализовать сканер пакетов и поиск структур
4. Реализовать генерацию кода для примитивных типов
5. Добавить обработку сложных типов (слайсы, мапы, указатели)
6. Добавить обработку вложенных структур
7. Реализовать запись сгенерированных файлов
8. Создать тесты
9. Создать документацию
10. Протестировать на примерах

## Ожидаемый результат

После выполнения утилиты для примеров структур будут созданы файлы:
- `internal/models/reset.gen.go`
- `internal/repository/reset.gen.go`
- `internal/handler/reset.gen.go`

Каждый файл будет содержать сгенерированные методы Reset() для соответствующих структур.