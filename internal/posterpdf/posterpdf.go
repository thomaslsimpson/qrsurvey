// Package posterpdf renders a printable US Letter poster: header text, a
// centered QR code, and the sweepstakes legal footer text.
package posterpdf

import (
	"bytes"
	"fmt"

	"github.com/go-pdf/fpdf"
)

const headerText = "This is a test poster. Scan it to enter our test contest!"

// legalText is reproduced verbatim from the product requirement — the
// bracketed placeholders are intentionally left as literal text, not
// filled in, until the real contest specifics exist.
const legalText = `NO PURCHASE NECESSARY. Open to legal residents of Mississippi who are 18 years of age or older. One entry per person. Scan the QR code to enter or visit [contest website link]. Completing the survey is optional and is not required to enter or improve your chances of winning. Odds of winning depend on the number of eligible entries received. Sweepstakes begins [date] and ends [date]. Winner will be selected by random drawing on or about [date] and notified using the contact information provided. Prize: [brief prize description] (Approximate Retail Value: $___). Void where prohibited. Full Official Rules are available at [contest website]/rules.`

// qrSizeMM is the printed QR size — large enough to scan reliably from a
// wall-mounted poster at arm's length.
const qrSizeMM = 110

// Generate builds a single-page PDF: header at top, qrPNG centered, and the
// legal text wrapped at the bottom.
func Generate(qrPNG []byte) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "Letter", "")
	pdf.AddPage()

	pageW, _ := pdf.GetPageSize()
	marginL, _, marginR, _ := pdf.GetMargins()
	contentW := pageW - marginL - marginR

	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(marginL, 20)
	pdf.MultiCell(contentW, 8, headerText, "", "C", false)

	qrX := (pageW - qrSizeMM) / 2
	qrY := 70.0
	pdf.RegisterImageOptionsReader("qr", fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(qrPNG))
	pdf.ImageOptions("qr", qrX, qrY, qrSizeMM, qrSizeMM, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetXY(marginL, qrY+qrSizeMM+15)
	pdf.MultiCell(contentW, 4.5, legalText, "", "L", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generate poster pdf: %w", err)
	}
	return buf.Bytes(), nil
}
