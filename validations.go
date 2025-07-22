package xmlparser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ValidationError holds a list of all validation errors found.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%d validation errors found:\n - %s", len(e.Errors), strings.Join(e.Errors, "\n - "))
}

// Validate checks if the given XML document conforms to the schema.
// It returns a single error (of type ValidationError) if issues are found.
func (s *Schema) Validate(doc *Document) error {
	if doc == nil || doc.Root == nil {
		return &ValidationError{Errors: []string{"XML document is empty"}}
	}
	rootDef, ok := s.ElementMap[doc.Root.Name.Local]
	if !ok {
		return &ValidationError{Errors: []string{fmt.Sprintf("root element <%s> is not defined in the schema", doc.Root.Name.Local)}}
	}
	errors := s.validateNode(doc.Root, rootDef)
	if len(errors) > 0 {
		return &ValidationError{Errors: errors}
	}
	return nil
}

// validateNode is the recursive heart of the validator. It walks the DOM tree.
func (s *Schema) validateNode(node *Node, def *Element) (errors []string) {
	// 1. Validate the node's content against its simpleType definition
	content := strings.TrimSpace(node.Content)
	if len(node.Children) == 0 && content != "" {
		// Find the simpleType definition for the current element
		stDef, err := s.findSimpleTypeForElement(def)
		if err != nil {
			errors = append(errors, fmt.Sprintf("in element <%s>: %v", def.Name, err))
		} else if stDef != nil {
			// Validate the content against the found simpleType rules
			validationErrs := validateSimpleContent(content, stDef)
			for _, e := range validationErrs {
				errors = append(errors, fmt.Sprintf("in element <%s>: %s", def.Name, e))
			}
		}
	}

	// 2. Get the complexType definition for this element
	var complexTypeDef *ComplexType
	if def.ComplexType != nil {
		complexTypeDef = def.ComplexType
	} else if ct, ok := s.ComplexTypeMap[def.Type]; ok {
		complexTypeDef = ct
	}

	if complexTypeDef == nil || complexTypeDef.Sequence == nil {
		if len(node.Children) > 0 {
			errors = append(errors, fmt.Sprintf("element <%s> should be empty but has children", node.Name.Local))
		}
		return
	}

	childCounts := make(map[string]int)
	for _, childNode := range node.Children {
		childCounts[childNode.Name.Local]++
		var childDef *Element
		for i := range complexTypeDef.Sequence.Elements {
			if complexTypeDef.Sequence.Elements[i].Name == childNode.Name.Local {
				childDef = &complexTypeDef.Sequence.Elements[i]
				break
			}
		}
		if childDef == nil {
			errors = append(errors, fmt.Sprintf("element <%s> is not a valid child of <%s>", childNode.Name.Local, node.Name.Local))
			continue
		}
		childErrors := s.validateNode(childNode, childDef)
		errors = append(errors, childErrors...)
	}

	for _, seqEl := range complexTypeDef.Sequence.Elements {
		count := childCounts[seqEl.Name]
		min, _ := strconv.Atoi(seqEl.MinOccurs)
		if seqEl.MinOccurs != "" && count < min {
			errors = append(errors, fmt.Sprintf("element <%s> requires at least %d <%s> child, but found %d", node.Name.Local, min, seqEl.Name, count))
		}
	}

	return errors

}

// This function contains the detailed logic for checking simple content.
func validateSimpleContent(content string, st *SimpleType) (errors []string) {
	if st == nil || st.Restriction == nil {
		return nil // No rules to check.
	}
	r := st.Restriction

	// Check pattern (regular expression)
	if r.Pattern != nil && r.Pattern.Value != "" {
		matched, err := regexp.MatchString(r.Pattern.Value, content)
		if err != nil {
			// This is a schema authoring error, not a validation error.
			errors = append(errors, fmt.Sprintf("invalid pattern in schema: %s", r.Pattern.Value))
			return
		}
		if !matched {
			errors = append(errors, fmt.Sprintf("value '%s' does not match pattern '%s'", content, r.Pattern.Value))
		}
	}

	// Check enumeration
	if len(r.Enumeration) > 0 {
		isAllowed := false
		allowedValues := []string{}
		for _, enum := range r.Enumeration {
			allowedValues = append(allowedValues, enum.Value)
			if content == enum.Value {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			errors = append(errors, fmt.Sprintf("value '%s' is not in the list of allowed values: [%s]", content, strings.Join(allowedValues, ", ")))
		}
	}

	// Other checks like minLength, maxLength would go here.

	return errors
}

// This helper function finds the simpleType definition for an element.
func (s *Schema) findSimpleTypeForElement(el *Element) (*SimpleType, error) {
	// Case 1: The simpleType is defined directly inside the element.
	if el.SimpleType != nil {
		return el.SimpleType, nil
	}

	// Case 2: The element references a built-in or named type.
	if el.Type != "" {
		// It's a named simple type defined at the top level of the schema.
		if st, ok := s.SimpleTypeMap[el.Type]; ok {
			return st, nil
		}
		// It's a built-in type like "xs:string" or "xs:integer".
		// For now, we don't have special rules for these, so we can return nil.
		if strings.HasPrefix(el.Type, "xs:") {
			return nil, nil
		}
		return nil, fmt.Errorf("type definition '%s' not found in schema", el.Type)
	}

	// No simple type is applicable.
	return nil, nil
}
