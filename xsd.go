/*
Package xmlparser provides pure Go XML Schema (XSD) validation functionality.

This package allows you to parse XSD schema files and validate XML documents
against them without requiring external C libraries or dependencies. It implements
a subset of the XML Schema specification covering the most commonly used features.

# Features

• Pure Go implementation - no CGO or external dependencies
• XSD schema parsing and validation
• Support for complex and simple types
• Pattern validation using regular expressions
• Length constraints (minLength, maxLength)
• Numeric range constraints (minInclusive, maxInclusive)
• Enumeration validation
• Occurrence constraints (minOccurs, maxOccurs)
• Built-in XML Schema type validation

# Basic Usage

	// Parse XSD schema
	xsdBytes := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">...</xs:schema>`)
	schema, err := xmlparser.ParseXSD(xsdBytes)
	if err != nil {
		log.Fatal(err)
	}

	// Parse XML document
	xmlBytes := []byte(`<root>...</root>`)
	document, err := xmlparser.Parse(xmlBytes)
	if err != nil {
		log.Fatal(err)
	}

	// Validate document against schema
	if err := schema.Validate(document); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("Document is valid!")
	}

# Supported XSD Features

The package supports a practical subset of XSD 1.0 features:

• Elements and attributes
• Complex types with sequences
• Simple types with restrictions
• Facets: pattern, enumeration, minLength, maxLength, minInclusive, maxInclusive
• Built-in types: xs:string, xs:integer, xs:decimal, xs:boolean, xs:date, etc.
• Occurrence indicators: minOccurs, maxOccurs (including "unbounded")

# Error Handling

Validation errors are returned as ValidationError instances that contain
detailed information about all validation failures found in the document:

	if err := schema.Validate(document); err != nil {
		if validationErr, ok := err.(*xmlparser.ValidationError); ok {
			fmt.Printf("Found %d validation errors:\n", len(validationErr.Errors))
			for _, errMsg := range validationErr.Errors {
				fmt.Printf("- %s\n", errMsg)
			}
		}
	}

# Limitations

This package currently has the following limitations:

• Limited namespace support (basic functionality only)
• No support for xs:choice or xs:all content models (only xs:sequence)
• No support for xs:import or xs:include
• No support for XML Schema 1.1 features
• No support for identity constraints (xs:key, xs:keyref, xs:unique)

For more examples and detailed documentation, see the examples directory
and the individual function documentation.
*/

package xmlparser

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ParseXSD parses an XSD schema from bytes and returns a Schema ready for validation.
// The returned schema includes lookup maps for efficient element and type resolution.
// Automatically processes any xs:import and xs:include elements found in the schema.
//
// Parameters:
//   - xsdBytes: The XSD schema content as bytes
//   - basePath: Optional base path for resolving relative schemaLocation paths (defaults to current directory)
//
// Returns a fully processed schema with all imports and includes resolved.
func ParseXSD(xsdBytes []byte, basePath ...string) (*Schema, error) {
	// Determine base path - use current directory if not provided
	resolvedBasePath := "."
	if len(basePath) > 0 && basePath[0] != "" {
		resolvedBasePath = basePath[0]
	}

	// Always use the full parsing with import/include support and circular reference protection
	return parseXSDWithImportsAndTracker(xsdBytes, resolvedBasePath, make(map[string]bool))
}

// parseBasicXSD parses an XSD schema without processing imports/includes.
// This is used internally by the import/include processing logic.
func parseBasicXSD(xsdBytes []byte) (*Schema, error) {
	schema := &Schema{}
	decoder := xml.NewDecoder(bytes.NewReader(xsdBytes))

	// Parse namespace declarations from the raw XML first
	if err := schema.extractNamespaces(xsdBytes); err != nil {
		return nil, fmt.Errorf("failed to extract namespaces: %w", err)
	}

	if err := decoder.Decode(schema); err != nil {
		return nil, fmt.Errorf("failed to decode XSD schema: %w", err)
	}

	if err := schema.buildLookupMaps(); err != nil {
		return nil, fmt.Errorf("failed to build schema lookup maps: %w", err)
	}

	return schema, nil
}

// buildLookupMaps creates internal maps for fast lookups during validation.
// This optimization avoids linear searches through slices during validation.
func (s *Schema) buildLookupMaps() error {
	s.ElementMap = make(map[string]*Element)
	s.ComplexTypeMap = make(map[string]*ComplexType)
	s.SimpleTypeMap = make(map[string]*SimpleType)

	// Build element lookup map
	if err := s.buildElementMap(); err != nil {
		return err
	}

	// Build complex type lookup map
	if err := s.buildComplexTypeMap(); err != nil {
		return err
	}

	// Build simple type lookup map
	if err := s.buildSimpleTypeMap(); err != nil {
		return err
	}

	return nil
}

// buildElementMap creates a lookup map for schema elements.
func (s *Schema) buildElementMap() error {
	for i := range s.Elements {
		element := &s.Elements[i]
		if element.Name == "" {
			return fmt.Errorf("schema element at index %d is missing required 'name' attribute", i)
		}
		if _, exists := s.ElementMap[element.Name]; exists {
			return fmt.Errorf("duplicate element definition: '%s'", element.Name)
		}
		s.ElementMap[element.Name] = element
	}
	return nil
}

// buildComplexTypeMap creates a lookup map for schema complex types.
func (s *Schema) buildComplexTypeMap() error {
	for i := range s.ComplexTypes {
		complexType := &s.ComplexTypes[i]
		if complexType.Name == "" {
			return fmt.Errorf("schema complexType at index %d is missing required 'name' attribute", i)
		}
		if _, exists := s.ComplexTypeMap[complexType.Name]; exists {
			return fmt.Errorf("duplicate complexType definition: '%s'", complexType.Name)
		}
		s.ComplexTypeMap[complexType.Name] = complexType
	}
	return nil
}

// buildSimpleTypeMap creates a lookup map for schema simple types.
func (s *Schema) buildSimpleTypeMap() error {
	for i := range s.SimpleTypes {
		simpleType := &s.SimpleTypes[i]
		if simpleType.Name == "" {
			return fmt.Errorf("schema simpleType at index %d is missing required 'name' attribute", i)
		}
		if _, exists := s.SimpleTypeMap[simpleType.Name]; exists {
			return fmt.Errorf("duplicate simpleType definition: '%s'", simpleType.Name)
		}
		s.SimpleTypeMap[simpleType.Name] = simpleType
	}
	return nil
}

// extractNamespaces parses namespace declarations from the schema root element.
func (s *Schema) extractNamespaces(xsdBytes []byte) error {
	s.Xmlns = make(map[string]string)

	// Parse the root element to extract namespace declarations
	decoder := xml.NewDecoder(bytes.NewReader(xsdBytes))
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		if startElem, ok := token.(xml.StartElement); ok {
			if startElem.Name.Local == "schema" {
				// Extract namespace declarations from schema element attributes
				for _, attr := range startElem.Attr {
					if attr.Name.Space == "xmlns" || attr.Name.Local == "xmlns" {
						prefix := attr.Name.Local
						if attr.Name.Space == "xmlns" {
							prefix = attr.Name.Local
						} else if attr.Name.Local == "xmlns" {
							prefix = "" // Default namespace
						}
						s.Xmlns[prefix] = attr.Value
					}
				}
				break // We only need the root schema element
			}
		}
	}

	// Ensure we have the standard XML Schema namespace
	if _, exists := s.Xmlns["xs"]; !exists {
		s.Xmlns["xs"] = "http://www.w3.org/2001/XMLSchema"
	}

	return nil
}

// parseXSDWithImportsAndTracker is the internal version with circular reference tracking.
func parseXSDWithImportsAndTracker(xsdBytes []byte, basePath string, visited map[string]bool) (*Schema, error) {
	schema, err := parseBasicXSD(xsdBytes)
	if err != nil {
		return nil, err
	}

	// Process imports and includes with circular reference detection
	if err := schema.processImportsAndIncludesWithTracker(basePath, visited); err != nil {
		return nil, fmt.Errorf("failed to process imports and includes: %w", err)
	}

	// Rebuild lookup maps after merging external schemas
	if err := schema.buildLookupMaps(); err != nil {
		return nil, fmt.Errorf("failed to rebuild lookup maps after import/include processing: %w", err)
	}

	return schema, nil
}

// processImportsAndIncludes loads and merges all external schemas referenced by xs:import and xs:include.
func (s *Schema) processImportsAndIncludes(basePath string) error {
	return s.processImportsAndIncludesWithTracker(basePath, make(map[string]bool))
}

// processImportsAndIncludesWithTracker loads and merges all external schemas with circular reference detection.
func (s *Schema) processImportsAndIncludesWithTracker(basePath string, visited map[string]bool) error {
	// Process includes first (same namespace)
	for _, include := range s.Includes {
		if err := s.processIncludeWithTracker(include, basePath, visited); err != nil {
			return fmt.Errorf("failed to process include '%s': %w", include.SchemaLocation, err)
		}
	}

	// Process imports (different namespaces)
	for _, imp := range s.Imports {
		if err := s.processImportWithTracker(imp, basePath, visited); err != nil {
			return fmt.Errorf("failed to process import '%s': %w", imp.SchemaLocation, err)
		}
	}

	return nil
}

// processInclude loads and merges an included schema (same namespace).
func (s *Schema) processInclude(include Include, basePath string) error {
	return s.processIncludeWithTracker(include, basePath, make(map[string]bool))
}

// processIncludeWithTracker loads and merges an included schema with circular reference detection.
func (s *Schema) processIncludeWithTracker(include Include, basePath string, visited map[string]bool) error {
	if include.SchemaLocation == "" {
		return fmt.Errorf("include element is missing schemaLocation attribute")
	}

	// Create absolute path for circular reference detection
	includedSchemaPath := include.SchemaLocation
	if !filepath.IsAbs(includedSchemaPath) && basePath != "" {
		includedSchemaPath = filepath.Join(basePath, include.SchemaLocation)
	}

	// Clean the path to ensure consistent comparison
	cleanPath, err := filepath.Abs(includedSchemaPath)
	if err != nil {
		cleanPath = includedSchemaPath
	}

	// Check for circular reference
	if visited[cleanPath] {
		return fmt.Errorf("circular reference detected: schema '%s' already being processed", cleanPath)
	}

	// Mark this schema as being processed
	visited[cleanPath] = true
	defer delete(visited, cleanPath)

	schemaBytes, err := loadSchema(include.SchemaLocation, basePath)
	if err != nil {
		return err
	}

	// Use parseXSDWithImportsAndTracker to handle any nested imports/includes consistently
	includedBasePath := filepath.Dir(includedSchemaPath)
	includedSchema, err := parseXSDWithImportsAndTracker(schemaBytes, includedBasePath, visited)
	if err != nil {
		return fmt.Errorf("failed to parse included schema: %w", err)
	}

	// Merge elements, types from included schema (which now includes all nested imports/includes)
	s.Elements = append(s.Elements, includedSchema.Elements...)
	s.ComplexTypes = append(s.ComplexTypes, includedSchema.ComplexTypes...)
	s.SimpleTypes = append(s.SimpleTypes, includedSchema.SimpleTypes...)

	return nil
}

// processImport loads and merges an imported schema (different namespace).
func (s *Schema) processImport(imp Import, basePath string) error {
	return s.processImportWithTracker(imp, basePath, make(map[string]bool))
}

// processImportWithTracker loads and merges an imported schema with circular reference detection.
func (s *Schema) processImportWithTracker(imp Import, basePath string, visited map[string]bool) error {
	if imp.SchemaLocation == "" {
		// Import without schemaLocation is allowed for built-in namespaces
		return nil
	}

	// Create absolute path for circular reference detection
	importedSchemaPath := imp.SchemaLocation
	if !filepath.IsAbs(importedSchemaPath) && basePath != "" {
		importedSchemaPath = filepath.Join(basePath, imp.SchemaLocation)
	}

	// Clean the path to ensure consistent comparison
	cleanPath, err := filepath.Abs(importedSchemaPath)
	if err != nil {
		cleanPath = importedSchemaPath
	}

	// Check for circular reference
	if visited[cleanPath] {
		return fmt.Errorf("circular reference detected: schema '%s' already being processed", cleanPath)
	}

	// Mark this schema as being processed
	visited[cleanPath] = true
	defer delete(visited, cleanPath)

	schemaBytes, err := loadSchema(imp.SchemaLocation, basePath)
	if err != nil {
		return err
	}

	// Use parseXSDWithImportsAndTracker to handle any nested imports/includes consistently
	importedBasePath := filepath.Dir(importedSchemaPath)
	importedSchema, err := parseXSDWithImportsAndTracker(schemaBytes, importedBasePath, visited)
	if err != nil {
		return fmt.Errorf("failed to parse imported schema: %w", err)
	}

	// Verify namespace consistency
	if imp.Namespace != "" && importedSchema.TargetNamespace != imp.Namespace {
		return fmt.Errorf("imported schema target namespace '%s' does not match expected namespace '%s'",
			importedSchema.TargetNamespace, imp.Namespace)
	}

	// Add namespace prefix for imported elements/types if needed
	prefix := s.getNamespacePrefix(imp.Namespace)
	if prefix != "" {
		s.mergeImportedSchemaWithPrefix(importedSchema, prefix)
	} else {
		// Merge directly if no prefix needed
		s.Elements = append(s.Elements, importedSchema.Elements...)
		s.ComplexTypes = append(s.ComplexTypes, importedSchema.ComplexTypes...)
		s.SimpleTypes = append(s.SimpleTypes, importedSchema.SimpleTypes...)
	}

	return nil
}

// loadSchema loads schema content from a file path or URL.
func loadSchema(schemaLocation, basePath string) ([]byte, error) {
	// Handle absolute URLs
	if strings.HasPrefix(schemaLocation, "http://") || strings.HasPrefix(schemaLocation, "https://") {
		resp, err := http.Get(schemaLocation)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch schema from URL '%s': %w", schemaLocation, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch schema from URL '%s': HTTP %d", schemaLocation, resp.StatusCode)
		}

		return io.ReadAll(resp.Body)
	}

	// Handle file paths
	path := schemaLocation
	if !filepath.IsAbs(path) && basePath != "" {
		path = filepath.Join(basePath, schemaLocation)
	}

	return os.ReadFile(path)
}

// getNamespacePrefix returns the prefix used for a given namespace.
func (s *Schema) getNamespacePrefix(namespace string) string {
	if s.Xmlns != nil {
		for prefix, ns := range s.Xmlns {
			if ns == namespace && prefix != "" {
				return prefix
			}
		}
	}
	return ""
}

// mergeImportedSchemaWithPrefix merges an imported schema, adding namespace prefixes to names.
func (s *Schema) mergeImportedSchemaWithPrefix(importedSchema *Schema, prefix string) {
	// Add prefix to element names and merge
	for _, element := range importedSchema.Elements {
		element.Name = prefix + ":" + element.Name
		s.Elements = append(s.Elements, element)
	}

	// Add prefix to complex type names and merge
	for _, complexType := range importedSchema.ComplexTypes {
		complexType.Name = prefix + ":" + complexType.Name
		s.ComplexTypes = append(s.ComplexTypes, complexType)
	}

	// Add prefix to simple type names and merge
	for _, simpleType := range importedSchema.SimpleTypes {
		simpleType.Name = prefix + ":" + simpleType.Name
		s.SimpleTypes = append(s.SimpleTypes, simpleType)
	}
}
