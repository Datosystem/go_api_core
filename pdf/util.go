package pdf

import (
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/phpdave11/gofpdf"
)

func LoadPngFromUrl(p *gofpdf.Fpdf, registerName, url string, wg *sync.WaitGroup) {
	resp, err := http.Get(url)
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			//			p.RegisterImageReader(registerName, "png", resp.Body)
			p.RegisterImageOptionsReader(registerName, gofpdf.ImageOptions{ImageType: "png"}, resp.Body)

		}
		resp.Body.Close()
	} else {
		err = errors.New("failed to load image " + url + ": " + err.Error())
		log.Println(err)
		// p.SetError(err)
	}
	if wg != nil {
		wg.Done()
	}
}

func Box(p *gofpdf.Fpdf, x, y, w, h float64, styleStr string, checked bool) {
	p.Rect(x, y, w, h, styleStr)
	if checked {
		p.Line(x, y, x+w, y+h)
		p.Line(x, y+h, x+w, y)
	}
}
