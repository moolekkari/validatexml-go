package xmlparser

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// ValidationError aggregates all validation errors found during validation.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%d validation errors found:\n - %s",
		len(e.Errors), strings.Join(e.Errors, "\n - "))
}

// Validate checks if the XML document conforms to the schema.
// Returns ValidationError if validation fails, nil if valid.
func (s *Schema) Validate(doc *Document) error {
	if doc == nil || doc.Root == nil {
		return &ValidationError{Errors: []string{"XML document is empty"}}
	}

	// Use namespace-aware element lookup
	elementKey := s.GetElementKey(doc.Root.Name)
	rootDef, exists := s.ElementMap[elementKey]
	if !exists {
		// Fallback to local name for compatibility
		if rootDef, exists = s.ElementMap[doc.Root.Name.Local]; !exists {
			return &ValidationError{Errors: []string{
				fmt.Sprintf("root element <%s> is not defined in the schema", doc.Root.Name.Local),
			}}
		}
	}

	if errors := s.validateNode(doc.Root, rootDef); len(errors) > 0 {
		return &ValidationError{Errors: errors}
	}
	return nil
}

// validateNode recursively validates a node and its children against the schema.
func (s *Schema) validateNode(node *Node, def *Element) []string {
	var errors []string

	// Validate text content for leaf nodes
	if len(node.Children) == 0 {
		errors = append(errors, s.validateTextContent(node, def)...)
	}

	// Validate complex type structure
	if complexType := s.getComplexType(def); complexType != nil {
		errors = append(errors, s.validateComplexType(node, complexType)...)
	} else if len(node.Children) > 0 {
		errors = append(errors, fmt.Sprintf("element <%s> should be empty but has children", node.Name.Local))
	}

	return errors
}

// validateTextContent validates the text content of a leaf node.
func (s *Schema) validateTextContent(node *Node, def *Element) []string {
	var errors []string
	content := strings.TrimSpace(node.Content)

	// Validate built-in types
	if def.Type != "" && strings.HasPrefix(def.Type, "xs:") {
		if err := validateBuiltInType(content, def.Type); err != nil {
			errors = append(errors, fmt.Sprintf("in element <%s>: %s", def.Name, err.Error()))
		}
	}

	// Validate simple type constraints
	if simpleType, err := s.findSimpleType(def); err != nil {
		errors = append(errors, fmt.Sprintf("in element <%s>: %v", def.Name, err))
	} else if simpleType != nil {
		for _, validationErr := range validateSimpleTypeConstraints(content, simpleType) {
			errors = append(errors, fmt.Sprintf("in element <%s>: %s", def.Name, validationErr))
		}
	}

	return errors
}

// validateComplexType validates a complex type's structure and occurrence constraints.
func (s *Schema) validateComplexType(node *Node, complexType *ComplexType) []string {
	var errors []string

	// Validate attributes
	errors = append(errors, s.validateAttributes(node, complexType.Attributes)...)

	// Validate content model
	if complexType.Sequence != nil {
		errors = append(errors, s.validateSequence(node, complexType.Sequence)...)
	} else if complexType.Choice != nil {
		errors = append(errors, s.validateChoice(node, complexType.Choice)...)
	} else if complexType.All != nil {
		errors = append(errors, s.validateAll(node, complexType.All)...)
	}

	return errors
}

// validateOccurrenceConstraints checks minOccurs and maxOccurs constraints.
func (s *Schema) validateOccurrenceConstraints(node *Node, sequence *Sequence, childCounts map[string]int) []string {
	var errors []string

	for _, element := range sequence.Elements {
		count := childCounts[element.Name]

		// Check minOccurs
		if element.MinOccurs != "" {
			if min, _ := strconv.Atoi(element.MinOccurs); count < min {
				errors = append(errors, fmt.Sprintf(
					"element <%s> requires at least %d <%s> child, but found %d",
					node.Name.Local, min, element.Name, count))
			}
		}

		// Check maxOccurs
		if element.MaxOccurs != "" && element.MaxOccurs != "unbounded" {
			if max, err := strconv.Atoi(element.MaxOccurs); err != nil {
				errors = append(errors, fmt.Sprintf(
					"invalid maxOccurs value in schema for element <%s>: %s",
					element.Name, element.MaxOccurs))
			} else if count > max {
				errors = append(errors, fmt.Sprintf(
					"element <%s> allows at most %d <%s> child, but found %d",
					node.Name.Local, max, element.Name, count))
			}
		}
	}

	return errors
}

// validateSimpleTypeConstraints validates content against simple type restrictions.
func validateSimpleTypeConstraints(content string, simpleType *SimpleType) []string {
	if simpleType == nil || simpleType.Restriction == nil {
		return nil
	}

	var errors []string
	restriction := simpleType.Restriction

	// Pattern validation
	if restriction.Pattern != nil && restriction.Pattern.Value != "" {
		if err := validatePattern(content, restriction.Pattern.Value); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Enumeration validation
	if len(restriction.Enumeration) > 0 {
		if err := validateEnumeration(content, restriction.Enumeration); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Length validation
	errors = append(errors, validateLengthConstraints(content, restriction)...)

	// Numeric range validation
	errors = append(errors, validateNumericConstraints(content, restriction)...)

	return errors
}

// Helper functions for getting types and elements

func (s *Schema) getComplexType(def *Element) *ComplexType {
	if def.ComplexType != nil {
		return def.ComplexType
	}
	if complexType, exists := s.ComplexTypeMap[def.Type]; exists {
		return complexType
	}
	return nil
}

func (s *Schema) findSimpleType(def *Element) (*SimpleType, error) {
	if def.SimpleType != nil {
		return def.SimpleType, nil
	}
	if def.Type != "" {
		if simpleType, exists := s.SimpleTypeMap[def.Type]; exists {
			return simpleType, nil
		}
		if strings.HasPrefix(def.Type, "xs:") {
			return nil, nil // Built-in type, no additional constraints
		}
		return nil, fmt.Errorf("type definition '%s' not found in schema", def.Type)
	}
	return nil, nil
}

func (s *Schema) countChildren(node *Node) map[string]int {
	childCounts := make(map[string]int)
	for _, child := range node.Children {
		childCounts[child.Name.Local]++
	}
	return childCounts
}

func (s *Schema) findChildElement(childName xml.Name, sequence *Sequence) *Element {
	// Try exact namespace-aware match first
	for i := range sequence.Elements {
		element := &sequence.Elements[i]
		// Check if element matches considering namespace
		if s.elementsMatch(childName, element.Name) {
			return element
		}
	}
	return nil
}

// elementsMatch checks if a child element matches a schema element definition considering namespaces.
func (s *Schema) elementsMatch(childName xml.Name, schemaElementName string) bool {
	// If schema element has no prefix, use local name comparison
	if !strings.Contains(schemaElementName, ":") {
		// For unqualified schema elements, match against local name
		return childName.Local == schemaElementName
	}

	// For qualified schema elements, resolve the namespace
	resolved := s.ResolveQName(schemaElementName)
	return childName.Local == resolved.LocalName &&
		(childName.Space == resolved.Namespace ||
			(childName.Space == s.TargetNamespace && resolved.Namespace == s.TargetNamespace))
}

// validateSequence validates an xs:sequence content model.
func (s *Schema) validateSequence(node *Node, sequence *Sequence) []string {
	var errors []string
	childCounts := s.countChildren(node)

	// Validate each child element
	for _, child := range node.Children {
		if childDef := s.findChildElement(child.Name, sequence); childDef != nil {
			errors = append(errors, s.validateNode(child, childDef)...)
		} else {
			errors = append(errors, fmt.Sprintf("element <%s> is not a valid child of <%s>",
				child.Name.Local, node.Name.Local))
		}
	}

	// Validate occurrence constraints
	errors = append(errors, s.validateSequenceOccurrences(node, sequence, childCounts)...)

	return errors
}

// validateChoice validates an xs:choice content model.
func (s *Schema) validateChoice(node *Node, choice *Choice) []string {
	var errors []string

	if len(node.Children) == 0 {
		// Check if choice is required
		if choice.MinOccurs == "" || choice.MinOccurs != "0" {
			errors = append(errors, fmt.Sprintf("element <%s> must contain at least one choice element", node.Name.Local))
		}
		return errors
	}

	// In a choice, only one alternative should be present (default behavior)
	maxOccurs := 1
	if choice.MaxOccurs != "" {
		if choice.MaxOccurs == "unbounded" {
			maxOccurs = -1 // unlimited
		} else if max, err := strconv.Atoi(choice.MaxOccurs); err == nil {
			maxOccurs = max
		}
	}

	// Count valid choice elements
	choiceElementCounts := make(map[string]int)
	for _, child := range node.Children {
		if childDef := s.findChoiceElement(child.Name, choice); childDef != nil {
			errors = append(errors, s.validateNode(child, childDef)...)
			choiceElementCounts[child.Name.Local]++
		} else {
			errors = append(errors, fmt.Sprintf("element <%s> is not a valid choice for <%s>",
				child.Name.Local, node.Name.Local))
		}
	}

	// Check choice constraints - by default, only one choice type is allowed
	if maxOccurs == 1 && len(choiceElementCounts) > 1 {
		choiceNames := make([]string, 0, len(choiceElementCounts))
		for name := range choiceElementCounts {
			choiceNames = append(choiceNames, name)
		}
		errors = append(errors, fmt.Sprintf("element <%s> choice allows only one alternative, but found: [%s]",
			node.Name.Local, strings.Join(choiceNames, ", ")))
	}

	return errors
}

// validateAll validates an xs:all content model.
func (s *Schema) validateAll(node *Node, all *All) []string {
	var errors []string
	childCounts := s.countChildren(node)

	// In xs:all, each element can appear at most once
	for childName, count := range childCounts {
		if count > 1 {
			errors = append(errors, fmt.Sprintf("element <%s> appears %d times in xs:all group, but maximum is 1",
				childName, count))
		}
	}

	// Validate each child element
	for _, child := range node.Children {
		if childDef := s.findAllElement(child.Name, all); childDef != nil {
			errors = append(errors, s.validateNode(child, childDef)...)
		} else {
			errors = append(errors, fmt.Sprintf("element <%s> is not allowed in xs:all group of <%s>",
				child.Name.Local, node.Name.Local))
		}
	}

	// Check required elements in xs:all
	for _, element := range all.Elements {
		if element.MinOccurs == "" || element.MinOccurs != "0" {
			if childCounts[element.Name] == 0 {
				errors = append(errors, fmt.Sprintf("required element <%s> is missing from xs:all group in <%s>",
					element.Name, node.Name.Local))
			}
		}
	}

	return errors
}

// validateAttributes validates XML attributes against XSD attribute definitions.
func (s *Schema) validateAttributes(node *Node, attributeDefs []Attribute) []string {
	var errors []string

	// Create maps for easier lookup
	attrValues := make(map[string]string)
	for _, attr := range node.Attrs {
		attrValues[attr.Name.Local] = attr.Value
	}

	// Validate each defined attribute
	for _, attrDef := range attributeDefs {
		value, present := attrValues[attrDef.Name]

		// Check required attributes
		if attrDef.Use == "required" && !present {
			errors = append(errors, fmt.Sprintf("required attribute '%s' is missing from element <%s>",
				attrDef.Name, node.Name.Local))
			continue
		}

		// Skip validation if attribute is not present and not required
		if !present {
			continue
		}

		// Validate fixed value
		if attrDef.Fixed != "" && value != attrDef.Fixed {
			errors = append(errors, fmt.Sprintf("attribute '%s' in element <%s> has fixed value '%s', but got '%s'",
				attrDef.Name, node.Name.Local, attrDef.Fixed, value))
		}

		// Validate attribute type
		if attrDef.Type != "" && strings.HasPrefix(attrDef.Type, "xs:") {
			if err := validateBuiltInType(value, attrDef.Type); err != nil {
				errors = append(errors, fmt.Sprintf("attribute '%s' in element <%s>: %s",
					attrDef.Name, node.Name.Local, err.Error()))
			}
		}

		// Validate inline simple type constraints
		if attrDef.SimpleType != nil {
			for _, validationErr := range validateSimpleTypeConstraints(value, attrDef.SimpleType) {
				errors = append(errors, fmt.Sprintf("attribute '%s' in element <%s>: %s",
					attrDef.Name, node.Name.Local, validationErr))
			}
		}
	}

	// Check for prohibited attributes (attributes not defined in schema)
	for _, attr := range node.Attrs {
		// Skip namespace declarations
		if s.isNamespaceDeclaration(attr) {
			continue
		}

		found := false
		for _, attrDef := range attributeDefs {
			if attrDef.Name == attr.Name.Local {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, fmt.Sprintf("unexpected attribute '%s' in element <%s>",
				attr.Name.Local, node.Name.Local))
		}
	}

	return errors
}

// isNamespaceDeclaration checks if an attribute is a namespace declaration.
func (s *Schema) isNamespaceDeclaration(attr xml.Attr) bool {
	return attr.Name.Space == "xmlns" || attr.Name.Local == "xmlns"
}
