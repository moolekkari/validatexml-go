package xmlparser

import "encoding/xml"

// Schema is the internal top-level structure representing a parsed <xs:schema>.
type Schema struct {
	XMLName            xml.Name `xml:"http://www.w3.org/2001/XMLSchema schema"`
	TargetNamespace    string   `xml:"targetNamespace,attr"`
	ElementFormDefault string   `xml:"elementFormDefault,attr"`

	Elements     []Element     `xml:"element"`
	ComplexTypes []ComplexType `xml:"complexType"`
	SimpleTypes  []SimpleType  `xml:"simpleType"`

	// Internal maps for fast lookups
	ElementMap     map[string]*Element
	ComplexTypeMap map[string]*ComplexType
	SimpleTypeMap  map[string]*SimpleType
}

// Element represents an <xs:element> declaration.
// This is a rule like: "You must have a piece named 'wheel'".
type Element struct {
	Name      string `xml:"name,attr"`
	Type      string `xml:"type,attr"` // e.g., "xs:string", "userType"
	MinOccurs string `xml:"minOccurs,attr"`
	MaxOccurs string `xml:"maxOccurs,attr"` // "unbounded" or a number

	// An element can define its own type right here, instead of referencing one.
	ComplexType *ComplexType `xml:"complexType"`
	SimpleType  *SimpleType  `xml:"simpleType"`
}

// ComplexType represents an <xs:complexType> declaration.
// This is a rule for a piece made of other pieces, like a car body with doors.
type ComplexType struct {
	Name     string    `xml:"name,attr"`
	Sequence *Sequence `xml:"sequence"` // The most common content model
	// Other models like <choice>, <all> can be added here.
}

// Sequence represents an <xs:sequence> declaration.
// This is a rule that says: "The pieces inside must appear in this exact order."
type Sequence struct {
	Elements []Element `xml:"element"`
}

// SimpleType represents an <xs:simpleType> declaration.
// This is a rule for a simple value, like text or a number.
type SimpleType struct {
	Name        string       `xml:"name,attr"`
	Restriction *Restriction `xml:"restriction"`
	// Other models like <list>, <union> can be added here.
}

// Restriction represents an <xs:restriction> declaration.
// This is a rule that puts limits on a simple value.
type Restriction struct {
	Base         string   `xml:"base,attr"` // e.g., "xs:string"
	MinLength    *Facet   `xml:"minLength"`
	MaxLength    *Facet   `xml:"maxLength"`
	Pattern      *Facet   `xml:"pattern"`
	MinInclusive *Facet   `xml:"minInclusive"` // For numbers
	MaxInclusive *Facet   `xml:"maxInclusive"` // For numbers
	Enumeration  []*Facet `xml:"enumeration"`  // List of allowed values
}

// Facet represents a single restriction rule, like <xs:minLength>.
// This is a specific rule like: "The name must be at least 3 letters long."
type Facet struct {
	Value string `xml:"value,attr"`
}

// Document represents a full XML document in memory.
type Document struct {
	Root *Node
}

// Node represents a single element in the XML document tree.
type Node struct {
	Parent   *Node
	Name     xml.Name
	Attrs    []xml.Attr
	Children []*Node
	Content  string
}
