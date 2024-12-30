package table

import (
	"log"
	"math"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type PdfHTMLElement struct {
	Data     string
	Text     string
	Width    float64
	Height   float64
	OffsetX  float64
	OffsetY  float64
	Style    Style
	Children []*PdfHTMLElement
}

type AddHTMLOptions struct {
	Width float64
}

func (t *Table) parseHTMLNode(node *html.Node, element *PdfHTMLElement) {
	if node.Type == html.TextNode {
		element.Data = "text"
		node.Data = strings.Trim(node.Data, "\n\t")
		if node.Data == "" {
			// Ignore empty text elements
			return
		}

		t.pdf.SetFontStyle("")
		if len(element.Style.Format) > 0 {
			if element.Style.Format == "-" {
				t.pdf.SetFontStyle("")
			} else {
				t.pdf.SetFontStyle(element.Style.Format)
			}
		}
		if element.Style.Size > 0 {
			t.pdf.SetFontSize(element.Style.Size)
		}
		availableWidth := element.Width
		if element.OffsetX > 0 {
			availableWidth -= element.OffsetX
		}
		if availableWidth <= 0 {
			_, fontHeight := t.pdf.GetFontSize()
			element.Height = fontHeight + element.Style.Ln
			off := element.OffsetX
			element.OffsetX = availableWidth + element.Height + 1
			availableWidth = math.Abs(element.Width - off)
			element.OffsetY = fontHeight * float64(len(t.pdf.SplitText(node.Data, availableWidth))) / 1.925
		}
		lines := t.pdf.SplitText(node.Data, availableWidth)
		node.Data = ""
		if len(lines) > 1 {
			node.Data = strings.Join(lines[1:], " ")
		}
		element.Width = t.pdf.GetStringWidth(lines[0])
		element.Text = lines[0]
		_, fontHeight := t.pdf.GetFontSize()
		element.Height = fontHeight + element.Style.Ln
	} else {
		element.Data = node.Data

		t.parseHTMLAttributes(node, element)
		t.parseChildHTMLNodes(node, element)

		if element.Style.inline {
			element.Width = 0
			for _, child := range element.Children {
				childWidth := child.OffsetX + child.Width
				if element.Width < childWidth {
					element.Width = childWidth
				}
			}
		} else {
			if element.Style.Align == "C" || element.Style.Align == "R" {
				childrenByOffsetY := map[float64][]*PdfHTMLElement{}
				for i := range element.Children {
					if _, ok := childrenByOffsetY[element.Children[i].OffsetY]; !ok {
						childrenByOffsetY[element.Children[i].OffsetY] = []*PdfHTMLElement{}
					}
					childrenByOffsetY[element.Children[i].OffsetY] = append(childrenByOffsetY[element.Children[i].OffsetY], element.Children[i])
				}
				for _, children := range childrenByOffsetY {
					var childrenWidth float64
					for _, child := range children {
						childrenWidth += child.Width
					}
					availableWidth := element.Width - childrenWidth
					if element.Style.Align == "C" {
						availableWidth = availableWidth / 2
					}
					for _, child := range children {
						child.OffsetX += availableWidth
					}
				}
			}
		}

		element.Height = 0
		for _, child := range element.Children {
			childHeight := child.OffsetY + child.Height
			if element.Height < childHeight {
				element.Height = childHeight
			}
		}
		element.Height += element.Style.PaddingBottom
	}
}

func (t *Table) parseChildHTMLNodes(node *html.Node, element *PdfHTMLElement) {
	var relX, relY float64
	relX = element.Style.PaddingLeft
	relY = element.Style.PaddingTop

	availableWidth := element.Width - element.Style.PaddingLeft - element.Style.PaddingRight

	for childNode := node.FirstChild; childNode != nil; childNode = childNode.NextSibling {
		switch childNode.Type {
		case html.DocumentNode, html.ElementNode, html.TextNode:
			if childNode.Type == html.ElementNode && childNode.Data == "br" {
				_, fontHeight := t.pdf.GetFontSize()
				relX = element.Style.PaddingLeft
				relY += fontHeight
				break
			}
			for {
				childElement := PdfHTMLElement{
					Width:   availableWidth,
					OffsetX: relX,
					OffsetY: relY,
					Style:   *element.Style.Inherit(),
				}
				childElement.Style.Fill = &[]int{}
				switch childNode.Type {
				case html.ElementNode:
					childElement.Data = childNode.Data
					switch childNode.Data {
					case "b", "strong":
						childElement.Style.inline = true
						childElement.Style.Format = "B"
						if element.Style.Format != "-" && !strings.Contains(element.Style.Format, childElement.Style.Format) {
							childElement.Style.Format += element.Style.Format
						}
					case "center":
						childElement.Style.Align = "C"
					case "span":
						childElement.Style.inline = true
						if element.Style.Format != "" {
							childElement.Style.Format = element.Style.Format
						} else {
							childElement.Style.Format = "-"
						}
					case "p":
						if element.Style.Format != "" {
							childElement.Style.Format = element.Style.Format
						} else {
							childElement.Style.Format = "-"
						}
						childElement.Style.Size = 9
					case "h1":
						childNode.Attr = append(childNode.Attr, html.Attribute{Namespace: "test", Key: "style", Val: "font-size: 14px; font-weight: bold; padding: 5 0 5 0 ;"})
					case "h2":
						childNode.Attr = append(childNode.Attr, html.Attribute{Namespace: "test", Key: "style", Val: "font-size: 14px; font-weight: bold; padding: 3 0 3 0 ;"})
					case "h3":
						childNode.Attr = append(childNode.Attr, html.Attribute{Namespace: "test", Key: "style", Val: "font-size: 12px; font-weight: bold; padding: 2 0 2 0 ;"})
					case "h4":
						childNode.Attr = append(childNode.Attr, html.Attribute{Namespace: "test", Key: "style", Val: "font-size: 11px; font-weight: bold; padding: 2.5 0 2.5 0 ;"})
					case "h5":
						childNode.Attr = append(childNode.Attr, html.Attribute{Namespace: "test", Key: "style", Val: "font-size: 9px; font-weight: bold; padding: 2 0 2 0 ;"})
					case "h6":
						childNode.Attr = append(childNode.Attr, html.Attribute{Namespace: "test", Key: "style", Val: "font-size: 7.5px; font-weight: bold; padding: 2.5 0 2.5 0 ;"})
					}
				case html.TextNode:
					childElement.Data = "text"
					childElement.Style.inline = true
				}
				if childElement.Style.inline {
					childElement.Width -= childElement.OffsetX
				} else {
					childElement.OffsetX = 0
					childElement.OffsetY = 0
					if len(element.Children) > 0 {
						prevChild := element.Children[len(element.Children)-1]
						childElement.OffsetY = prevChild.OffsetY + prevChild.Height
					}
				}
				t.parseHTMLNode(childNode, &childElement)
				if childNode.Type == html.TextNode && childElement.Text == "" {
					break
				}

				if childElement.Style.inline {
					relX = childElement.OffsetX + childElement.Width
				}
				relY = childElement.OffsetY
				element.Children = append(element.Children, &childElement)
				// Loop until all text has been segmented
				if childNode.Type == html.TextNode {
					if childNode.Data != "" {
						relY += childElement.Height
						relX = 0
						if element.Style.inline {
							relX = element.OffsetX * -1
						}
						continue
					}
				}
				break
			}
		}
	}
}

func (t *Table) parseHTMLAttributes(node *html.Node, element *PdfHTMLElement) {
	for _, attr := range node.Attr {
		switch attr.Key {
		case "style":
			styles := strings.Split(attr.Val, ";")
			for _, s := range styles {
				pieces := strings.Split(s, ":")
				val := strings.Join(pieces[1:], ":")
				switch strings.TrimSpace(pieces[0]) {
				case "background-color":
					if strings.HasPrefix(val, "rgb(") {
						val = val[4 : len(val)-1]
						rgb := strings.Split(val, ",")
						element.Style.Fill = &[]int{}
						for _, color := range rgb {
							c, err := strconv.Atoi(strings.TrimSpace(color))
							if err != nil {
								log.Println("AddHTML: error parsing background color", err)
								continue
							}
							*element.Style.Fill = append(*element.Style.Fill, c)
						}
					}
				case "color":
					if strings.HasPrefix(val, "rgb(") {
						val = val[4 : len(val)-1]
						rgb := strings.Split(val, ",")
						element.Style.Color = &[]int{}
						for _, color := range rgb {
							c, err := strconv.Atoi(strings.TrimSpace(color))
							if err != nil {
								log.Println("AddHTML: error parsing color", err)
								continue
							}
							*element.Style.Color = append(*element.Style.Color, c)
						}
					}
				case "font-weight":
					switch strings.TrimSpace(val) {
					case "bold":
						element.Style.Format = "B"
					case "normal":
						element.Style.Format = "-"
					}
				case "padding":
					paddingPieces := strings.Split(val, " ")
					padding := []float64{}
					for _, piece := range paddingPieces {
						if v, err := strconv.ParseFloat(piece, 64); err == nil {
							padding = append(padding, v)
						}
					}
					element.Style.Padding(padding...)
				case "text-align":
					switch strings.TrimSpace(val) {
					case "left":
						element.Style.Align = "L"
					case "center":
						element.Style.Align = "C"
					case "right":
						element.Style.Align = "R"
					}
				case "font-size":
					val = strings.ReplaceAll(val, " ", "")
					if strings.HasSuffix(val, "px") {
						element.Style.Size, _ = strconv.ParseFloat(strings.TrimSuffix(val, "px"), 64)
					}
					if strings.HasSuffix(val, "pt") {
						element.Style.Size, _ = strconv.ParseFloat(strings.TrimSuffix(val, "pt"), 64)
					}
				case "line-height":
					element.Style.Ln, _ = strconv.ParseFloat(val, 64)
				case "text-decoration":
					switch strings.TrimSpace(val) {
					case "underline":
						if element.Style.Format != "-" {
							element.Style.Format += "U"
						} else {
							element.Style.Format = "U"
						}
					}
				}
			}
		}
	}
}
