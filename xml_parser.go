package xmlparser

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// Parse parses XML data and constructs a Document tree structure for validation.
// The resulting Document can be validated against an XSD schema.
func Parse(xmlBytes []byte) (*Document, error) {
	decoder := xml.NewDecoder(bytes.NewReader(xmlBytes))
	parser := &xmlParser{decoder: decoder}

	return parser.parseDocument()
}

// xmlParser handles the XML parsing state and logic.
type xmlParser struct {
	decoder     *xml.Decoder
	currentNode *Node
	document    *Document
}

// parseDocument parses the entire XML document into a Document tree.
func (p *xmlParser) parseDocument() (*Document, error) {
	p.document = &Document{}

	for {
		token, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("XML parsing error: %w", err)
		}

		if err := p.processToken(token); err != nil {
			return nil, err
		}
	}

	if p.document.Root == nil {
		return nil, fmt.Errorf("XML document is empty or contains no root element")
	}

	return p.document, nil
}

// processToken processes a single XML token and updates the document tree.
func (p *xmlParser) processToken(token xml.Token) error {
	switch t := token.(type) {
	case xml.StartElement:
		return p.handleStartElement(t)
	case xml.CharData:
		p.handleCharData(t)
	case xml.EndElement:
		p.handleEndElement()
	case xml.Comment:
		// Ignore comments for validation purposes
	case xml.ProcInst:
		// Ignore processing instructions for validation purposes
	default:
		// Other token types are ignored
	}
	return nil
}

// handleStartElement processes an XML start element token.
func (p *xmlParser) handleStartElement(element xml.StartElement) error {
	node := &Node{
		Parent: p.currentNode,
		Name:   element.Name,
		Attrs:  make([]xml.Attr, len(element.Attr)),
	}

	// Copy attributes to avoid referencing the token's memory
	copy(node.Attrs, element.Attr)

	// Set as root if this is the first element
	if p.document.Root == nil {
		p.document.Root = node
	}

	// Add as child to current parent if we have one
	if p.currentNode != nil {
		p.currentNode.Children = append(p.currentNode.Children, node)
	}

	// Move into the new element
	p.currentNode = node
	return nil
}

// handleCharData processes character data (text content) within an element.
func (p *xmlParser) handleCharData(data xml.CharData) {
	if p.currentNode != nil {
		p.currentNode.Content += string(data)
	}
}

// handleEndElement processes an XML end element token.
func (p *xmlParser) handleEndElement() {
	if p.currentNode != nil {
		p.currentNode = p.currentNode.Parent
	}
}
