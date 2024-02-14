package pdf

import (
	"github.com/iancoleman/orderedmap"
	"github.com/phpdave11/gofpdf"
)

type Registrable interface {
	Headers() map[string]func(*gofpdf.Fpdf)
	Footers() map[string]func(*gofpdf.Fpdf)
	AcceptPageBreak() map[string]func() bool
}

type Document struct {
	HeaderFns          *orderedmap.OrderedMap
	FooterFns          *orderedmap.OrderedMap
	AcceptPageBreakFns *orderedmap.OrderedMap
}

func (d *Document) Init(p *gofpdf.Fpdf) {
	d.HeaderFns = orderedmap.New()
	p.SetHeaderFunc(func() {
		for _, key := range d.HeaderFns.Keys() {
			fn, _ := d.HeaderFns.Get(key)
			fn.(func(*gofpdf.Fpdf))(p)
		}
	})
	d.FooterFns = orderedmap.New()
	p.SetFooterFunc(func() {
		for _, key := range d.FooterFns.Keys() {
			fn, _ := d.FooterFns.Get(key)
			fn.(func(*gofpdf.Fpdf))(p)
		}
	})
	d.AcceptPageBreakFns = orderedmap.New()
	p.SetAcceptPageBreakFunc(func() bool {
		for _, key := range d.AcceptPageBreakFns.Keys() {
			fn, _ := d.AcceptPageBreakFns.Get(key)
			if !fn.(func() bool)() {
				return false
			}
		}
		return true
	})
}

func (d Document) AddHeader(key string, value func(*gofpdf.Fpdf)) {
	d.HeaderFns.Set(key, value)
}

func (d Document) RemoveHeader(key string) {
	d.HeaderFns.Delete(key)
}

func (d Document) AddFooter(key string, value func(*gofpdf.Fpdf)) {
	d.FooterFns.Set(key, value)
}

func (d Document) RemoveFooter(key string) {
	d.FooterFns.Delete(key)
}

func (d Document) AddAcceptPageBreak(key string, value func() bool) {
	d.AcceptPageBreakFns.Set(key, value)
}

func (d Document) RemoveAcceptPageBreak(key string) {
	d.AcceptPageBreakFns.Delete(key)
}

func (d *Document) Register(reg Registrable) {
	headers := reg.Headers()
	for key, handler := range headers {
		d.AddHeader(key, handler)
	}
	footers := reg.Footers()
	for key, handler := range footers {
		d.AddFooter(key, handler)
	}
	acceptPageBreak := reg.AcceptPageBreak()
	for key, handler := range acceptPageBreak {
		d.AddAcceptPageBreak(key, handler)
	}
}

func (d *Document) Unregister(reg Registrable) {
	headers := reg.Headers()
	for key := range headers {
		d.RemoveHeader(key)
	}
	footers := reg.Footers()
	for key := range footers {
		d.RemoveFooter(key)
	}
	acceptPageBreak := reg.AcceptPageBreak()
	for key := range acceptPageBreak {
		d.RemoveAcceptPageBreak(key)
	}
}

func NewDocument(p *gofpdf.Fpdf) *Document {
	doc := Document{}
	doc.Init(p)
	return &doc
}
