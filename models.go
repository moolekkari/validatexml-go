// Package xmlparser provides XML Schema (XSD) validation functionality in pure Go.
// It supports parsing XSD schemas and validating XML documents against them without
// requiring external C libraries or dependencies.
package xmlparser

import (
	"encoding/xml"
	"strings"
)

// Schema represents a parsed XML Schema Definition (XSD).
// It contains all the type definitions and validation rules from the schema.
type Schema struct {
	XMLName            xml.Name `xml:"http://www.w3.org/2001/XMLSchema schema"`
	TargetNamespace    string   `xml:"targetNamespace,attr"`
	ElementFormDefault string   `xml:"elementFormDefault,attr"`

	// Namespace declarations
	Xmlns map[string]string `xml:"-"` // Namespace prefix mappings

	// XSD definitions
	Elements     []Element     `xml:"element"`
	ComplexTypes []ComplexType `xml:"complexType"`
	SimpleTypes  []SimpleType  `xml:"simpleType"`
	Imports      []Import      `xml:"import"`
	Includes     []Include     `xml:"include"`

	// Internal lookup maps (populated during parsing)
	ElementMap     map[string]*Element
	ComplexTypeMap map[string]*ComplexType
	SimpleTypeMap  map[string]*SimpleType
}

// Element represents an XSD element definition.
// Elements define the structure and constraints for XML elements.
type Element struct {
	Name      string `xml:"name,attr"`
	Type      string `xml:"type,attr"`      // Reference to a type (e.g., "xs:string")
	MinOccurs string `xml:"minOccurs,attr"` // Minimum occurrences (default: 1)
	MaxOccurs string `xml:"maxOccurs,attr"` // Maximum occurrences ("unbounded" or number)

	// Inline type definitions (alternative to Type reference)
	ComplexType *ComplexType `xml:"complexType"`
	SimpleType  *SimpleType  `xml:"simpleType"`
}

// ComplexType represents an XSD complex type definition.
// Complex types define elements that can contain other elements or attributes.
type ComplexType struct {
	Name       string      `xml:"name,attr"`
	Sequence   *Sequence   `xml:"sequence"`  // Ordered sequence of child elements
	Choice     *Choice     `xml:"choice"`    // Choice between alternative elements
	All        *All        `xml:"all"`       // Unordered group of elements
	Attributes []Attribute `xml:"attribute"` // Element attributes
}

// Sequence represents an ordered sequence of elements in a complex type.
type Sequence struct {
	Elements  []Element `xml:"element"`
	MinOccurs string    `xml:"minOccurs,attr"`
	MaxOccurs string    `xml:"maxOccurs,attr"`
}

// Choice represents a choice between alternative elements.
type Choice struct {
	Elements  []Element  `xml:"element"`
	Sequences []Sequence `xml:"sequence"`
	Choices   []Choice   `xml:"choice"`
	MinOccurs string     `xml:"minOccurs,attr"`
	MaxOccurs string     `xml:"maxOccurs,attr"`
}

// All represents an unordered group of elements (each appears 0 or 1 times).
type All struct {
	Elements  []Element `xml:"element"`
	MinOccurs string    `xml:"minOccurs,attr"`
}

// SimpleType represents an XSD simple type definition.
// Simple types define constraints for text content and primitive values.
type SimpleType struct {
	Name        string       `xml:"name,attr"`
	Restriction *Restriction `xml:"restriction"` // Value restrictions/constraints
	// TODO: Add support for List and Union types
}

// Restriction defines validation constraints for simple types.
type Restriction struct {
	Base string `xml:"base,attr"` // Base type (e.g., "xs:string", "xs:integer")

	// String constraints
	MinLength *Facet `xml:"minLength"`
	MaxLength *Facet `xml:"maxLength"`
	Pattern   *Facet `xml:"pattern"`

	// Numeric constraints
	MinInclusive *Facet `xml:"minInclusive"`
	MaxInclusive *Facet `xml:"maxInclusive"`

	// Enumeration constraints
	Enumeration []*Facet `xml:"enumeration"`
}

// Facet represents a single validation constraint with its value.
type Facet struct {
	Value string `xml:"value,attr"`
}

// Attribute represents an XSD attribute definition.
type Attribute struct {
	Name       string      `xml:"name,attr"`
	Type       string      `xml:"type,attr"`
	Use        string      `xml:"use,attr"` // required, optional, prohibited
	Default    string      `xml:"default,attr"`
	Fixed      string      `xml:"fixed,attr"`
	SimpleType *SimpleType `xml:"simpleType"` // Inline simple type definition
}

// Document represents a parsed XML document as a tree structure.
type Document struct {
	Root *Node // Root element of the document
}

// Node represents a single XML element in the document tree.
type Node struct {
	Parent   *Node      // Parent node (nil for root)
	Name     xml.Name   // Element name with namespace
	Attrs    []xml.Attr // Element attributes
	Children []*Node    // Child elements
	Content  string     // Text content (for leaf nodes)
}

// QName represents a qualified name with namespace prefix and local name.
type QName struct {
	Prefix    string // Namespace prefix (empty for default namespace)
	LocalName string // Local element name
	Namespace string // Full namespace URI
}

// ParseQName parses a qualified name string into prefix and local name parts.
func ParseQName(qname string) QName {
	if strings.Contains(qname, ":") {
		parts := strings.SplitN(qname, ":", 2)
		return QName{
			Prefix:    parts[0],
			LocalName: parts[1],
		}
	}
	return QName{
		Prefix:    "",
		LocalName: qname,
	}
}

// ResolveQName resolves a qualified name using the schema's namespace mappings.
func (s *Schema) ResolveQName(qname string) QName {
	parsed := ParseQName(qname)
	if s.Xmlns != nil {
		if namespace, exists := s.Xmlns[parsed.Prefix]; exists {
			parsed.Namespace = namespace
		}
	}
	return parsed
}

// IsQualified returns true if the element should be namespace-qualified.
func (s *Schema) IsQualified(elementName string) bool {
	// Elements are qualified if elementFormDefault="qualified" or if they have a prefix
	return s.ElementFormDefault == "qualified" || strings.Contains(elementName, ":")
}

// GetElementKey returns the appropriate key for element lookup based on namespace rules.
func (s *Schema) GetElementKey(name xml.Name) string {
	// For qualified elements, use the full name with namespace
	if s.IsQualified(name.Local) && name.Space != "" {
		if name.Space == s.TargetNamespace {
			return name.Local // Use local name for target namespace elements
		}
		return name.Space + ":" + name.Local // Use full qualified name for other namespaces
	}
	return name.Local // Use local name for unqualified elements
}

// Import represents an xs:import element for including external schemas from different namespaces.
type Import struct {
	Namespace      string `xml:"namespace,attr"`      // Target namespace of the imported schema
	SchemaLocation string `xml:"schemaLocation,attr"` // URL/path to the schema file
}

// Include represents an xs:include element for including external schemas from the same namespace.
type Include struct {
	SchemaLocation string `xml:"schemaLocation,attr"` // URL/path to the schema file
}
