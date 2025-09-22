# Pure Go XML Schema (XSD) Validator

A fast, lightweight XML Schema validation library written in pure Go. No CGO or external dependencies required.

## Features

✅ **Pure Go** - No CGO or external C library dependencies
✅ **XSD Parsing** - Parse XML Schema Definition files into Go structs
✅ **XML Validation** - Validate XML documents against XSD schemas
✅ **Comprehensive Validation** - Pattern, length, range, enumeration, and occurrence constraints
✅ **Built-in Types** - Support for xs:string, xs:integer, xs:decimal, xs:boolean, xs:date, etc.
✅ **Fast Performance** - Optimized with internal lookup maps for efficient validation
✅ **Detailed Errors** - Clear, actionable validation error messages

## Installation

```bash
go get github.com/moolekkari/validatexml-go
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/moolekkari/validatexml-go"
)

func main() {
    // Your XSD schema
    xsdData := []byte(`
    <xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
        <xs:element name="person">
            <xs:complexType>
                <xs:sequence>
                    <xs:element name="name" type="xs:string"/>
                    <xs:element name="age" type="xs:integer"/>
                </xs:sequence>
            </xs:complexType>
        </xs:element>
    </xs:schema>`)

    // Parse schema
    schema, err := xmlparser.ParseXSD(xsdData)
    if err != nil {
        log.Fatal(err)
    }

    // Your XML document
    xmlData := []byte(`
    <person>
        <name>John Doe</name>
        <age>30</age>
    </person>`)

    // Parse and validate
    document, err := xmlparser.Parse(xmlData)
    if err != nil {
        log.Fatal(err)
    }

    if err := schema.Validate(document); err != nil {
        fmt.Printf("Validation failed: %v\n", err)
    } else {
        fmt.Println("✓ Valid!")
    }
}
```

## Supported XSD Features

### ✅ Fully Implemented
- **Elements**: `<xs:element>` with name, type, minOccurs, maxOccurs
- **Complex Types**: `<xs:complexType>` with all content models
- **Content Models**:
  - `<xs:sequence>` - Ordered child elements
  - `<xs:choice>` - Alternative child elements (pick one)
  - `<xs:all>` - Unordered child elements (each appears 0 or 1 times)
- **Simple Types**: `<xs:simpleType>` with restrictions
- **Attributes**: Full attribute validation with use, default, and fixed values
- **Comprehensive Built-in Types**:
  - **Integers**: xs:integer, xs:int, xs:long, xs:short, xs:byte, xs:nonNegativeInteger, xs:positiveInteger, xs:unsignedInt
  - **Decimals**: xs:decimal, xs:double, xs:float
  - **Strings**: xs:string, xs:normalizedString, xs:token, xs:Name, xs:NCName, xs:ID, xs:IDREF
  - **Boolean**: xs:boolean
  - **Dates/Times**: xs:date, xs:dateTime, xs:time, xs:gYear, xs:gMonth, xs:gDay, xs:duration
  - **URIs**: xs:anyURI
  - **Binary**: xs:base64Binary, xs:hexBinary
- **Facets**:
  - `xs:pattern` - Regular expression validation
  - `xs:enumeration` - Allowed value lists
  - `xs:minLength` / `xs:maxLength` - String length constraints
  - `xs:minInclusive` / `xs:maxInclusive` - Numeric range constraints
- **Occurrence**: `minOccurs`, `maxOccurs` (including "unbounded")

### ✅ Advanced Features (New!)
- **Enhanced namespace support**: Full `targetNamespace` and qualified element handling
- **`xs:import` and `xs:include`**: Automatic processing of external schema references with circular reference protection

## Examples

### Content Models

#### xs:sequence (Ordered Elements)
```go
xsd := `<xs:complexType name="personType">
    <xs:sequence>
        <xs:element name="firstName" type="xs:string"/>
        <xs:element name="lastName" type="xs:string"/>
        <xs:element name="age" type="xs:integer"/>
    </xs:sequence>
</xs:complexType>`
```

#### xs:choice (Alternative Elements)
```go
xsd := `<xs:complexType name="contactType">
    <xs:choice>
        <xs:element name="email" type="xs:string"/>
        <xs:element name="phone" type="xs:string"/>
        <xs:element name="address" type="xs:string"/>
    </xs:choice>
</xs:complexType>`
```

#### xs:all (Unordered Elements)
```go
xsd := `<xs:complexType name="productType">
    <xs:all>
        <xs:element name="name" type="xs:string"/>
        <xs:element name="price" type="xs:decimal"/>
        <xs:element name="category" type="xs:string" minOccurs="0"/>
    </xs:all>
</xs:complexType>`
```

### Attribute Validation
```go
xsd := `<xs:complexType name="itemType">
    <xs:sequence>
        <xs:element name="description" type="xs:string"/>
    </xs:sequence>
    <xs:attribute name="id" type="xs:integer" use="required"/>
    <xs:attribute name="category" type="xs:string" use="optional"/>
    <xs:attribute name="status" type="xs:string" fixed="active"/>
</xs:complexType>`
```

### Extended Built-in Types
```go
xsd := `<xs:element name="event">
    <xs:complexType>
        <xs:sequence>
            <xs:element name="timestamp" type="xs:dateTime"/>
            <xs:element name="duration" type="xs:duration"/>
            <xs:element name="attendees" type="xs:positiveInteger"/>
            <xs:element name="website" type="xs:anyURI"/>
            <xs:element name="published" type="xs:boolean"/>
        </xs:sequence>
    </xs:complexType>
</xs:element>`
```

### Pattern Validation
```go
xsd := `<xs:element name="email">
    <xs:simpleType>
        <xs:restriction base="xs:string">
            <xs:pattern value="[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}"/>
        </xs:restriction>
    </xs:simpleType>
</xs:element>`
```

### Numeric Range Validation
```go
xsd := `<xs:element name="age">
    <xs:simpleType>
        <xs:restriction base="xs:integer">
            <xs:minInclusive value="0"/>
            <xs:maxInclusive value="120"/>
        </xs:restriction>
    </xs:simpleType>
</xs:element>`
```

### Length Constraints
```go
xsd := `<xs:element name="username">
    <xs:simpleType>
        <xs:restriction base="xs:string">
            <xs:minLength value="3"/>
            <xs:maxLength value="20"/>
        </xs:restriction>
    </xs:simpleType>
</xs:element>`
```

### Occurrence Constraints
```go
xsd := `<xs:complexType name="bookType">
    <xs:sequence>
        <xs:element name="title" type="xs:string"/>
        <xs:element name="author" type="xs:string" maxOccurs="5"/>
        <xs:element name="isbn" type="xs:string" minOccurs="0"/>
    </xs:sequence>
</xs:complexType>`
```

### Working with External Schemas (xs:import and xs:include)

The `ParseXSD` function automatically processes external schema references:

```go
// Main schema that includes/imports other schemas
mainSchema := []byte(`
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/main">
    <!-- Include schema from same namespace -->
    <xs:include schemaLocation="common-types.xsd"/>

    <!-- Import schema from different namespace -->
    <xs:import namespace="http://example.com/address"
               schemaLocation="address.xsd"/>

    <xs:element name="person" type="PersonType"/>
</xs:schema>`)

// Parse with base path for resolving relative schema locations
schema, err := xmlparser.ParseXSD(mainSchema, "/path/to/schemas")
if err != nil {
    log.Fatal(err)
}

// The schema now includes all types from external files
// Validation works seamlessly across all included/imported schemas
```

**Key features:**
- **Automatic processing**: No need for separate APIs - `ParseXSD` handles everything
- **Circular reference protection**: Prevents infinite loops in schema dependencies
- **Relative path resolution**: Uses the provided base path to resolve `schemaLocation` attributes
- **Namespace consistency**: Validates that imported schemas match expected namespaces

## Error Handling

The library provides detailed validation errors:

```go
if err := schema.Validate(document); err != nil {
    if validationErr, ok := err.(*xmlparser.ValidationError); ok {
        fmt.Printf("Found %d validation errors:\n", len(validationErr.Errors))
        for _, errMsg := range validationErr.Errors {
            fmt.Printf("  - %s\n", errMsg)
        }
    }
}
```

Example output:
```
Found 2 validation errors:
  - in element <age>: value '150' exceeds maximum allowed value 120
  - in element <email>: value 'invalid-email' does not match pattern '[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}'
```

## Testing

```bash
go test -v
```

All validation features are thoroughly tested with comprehensive test coverage.

## Performance

The library is optimized for performance:
- Schema parsing builds internal lookup maps for O(1) element/type resolution
- Streaming XML parser with minimal memory allocation
- Efficient validation algorithms with early termination on errors

## Contributing

Contributions are welcome! Please feel free to submit issues, feature requests, or pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

This library is inspired by and functionally compatible with the Rust `xmlschema-rs` library, adapted for the Go ecosystem with Go-specific optimizations and idioms.
