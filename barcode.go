package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/datamatrix"
	"github.com/boombuler/barcode/ean"
	"github.com/boombuler/barcode/pdf417"
	"github.com/boombuler/barcode/qr"
	"github.com/boombuler/barcode/twooffive"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// BarcodeOptions описывает параметры генерации этикетки со штрихкодом или QR-кодом.
type BarcodeOptions struct {
	Data           string
	Format         string // auto | code128 | qr | qrl | qrr
	LengthMM       float64
	TopText        string
	BottomText     string
	FontName       string
	IsMainTextBold bool
	FontSize       string // -1 (авто), 0 (скрыть), N (pt), Np (%)
	OffsetPx       int
	SideMarginPx   int
	MarginPx       int
	GapPx          int
	TextGapPx      int
	SafeMode       bool
}

var (
	reOnlyDigits = regexp.MustCompile(`^[0-9]+$`)
	reAsciiPrint = regexp.MustCompile(`^[\x20-\x7E]+$`)
)

// RenderBarcodeLabel строит готовое изображение этикетки со штрихкодом (ITF / Code 128) или QR-кодом.
func RenderBarcodeLabel(opts BarcodeOptions, cfg AppConfig) (*image.Gray, error) {
	raw := strings.TrimSpace(opts.Data)
	if raw == "" {
		return nil, fmt.Errorf("не заданы данные для штрихкода")
	}

	fmtMode := strings.ToLower(strings.TrimSpace(opts.Format))
	if fmtMode == "" {
		fmtMode = cfg.BcFormat
	}
	if fmtMode == "" {
		fmtMode = "auto"
	}

	switch fmtMode {
	case "auto", "code128", "code39", "ean13", "ean8", "pdf417":
		if !reAsciiPrint.MatchString(raw) {
			return nil, fmt.Errorf("линейный штрихкод поддерживает только латиницу, цифры и спецсимволы ASCII")
		}
		return renderLinearBarcodeLabel(raw, fmtMode, opts)
	case "qr", "qrl", "qrr":
		side := cfg.DefaultQrSide
		if fmtMode == "qrl" {
			side = "left"
		} else if fmtMode == "qrr" {
			side = "right"
		}
		return render2dLabel(raw, side, opts, false)
	case "datamatrix", "dm", "dml", "dmr":
		side := cfg.DefaultQrSide
		if fmtMode == "qrl" {
			side = "left"
		} else if fmtMode == "qrr" {
			side = "right"
		}
		if fmtMode == "dml" {
			side = "left"
		} else if fmtMode == "dmr" {
			side = "right"
		}
		return render2dLabel(raw, side, opts, true)
	default:
		return nil, fmt.Errorf("неизвестный формат штрихкода %q (доступны: auto, code128, qr, qrl, qrr)", fmtMode)
	}
}

// encodeLinear кодирует строку в ITF (при auto и чётных цифрах) или Code 128.
func encodeLinear(raw, fmtMode string) (barcode.Barcode, string, error) {
	switch fmtMode {
	case "code39":
		norm := strings.ToUpper(raw)
		bc, err := code39.Encode(norm, false, true)
		return bc, norm, err
	case "ean13":
		norm, err := normalizeEan13(raw)
		if err != nil {
			return nil, raw, err
		}
		bc, err := ean.Encode(norm)
		return bc, norm, err
	case "ean8":
		norm, err := normalizeEan8(raw)
		if err != nil {
			return nil, raw, err
		}
		bc, err := ean.Encode(norm)
		return bc, norm, err
	case "pdf417":
		bc, err := pdf417.Encode(raw, 2)
		return bc, raw, err
	default:
		if fmtMode == "auto" && reOnlyDigits.MatchString(raw) && len(raw)%2 == 0 {
			bc, err := twooffive.Encode(raw, true)
			return bc, raw, err
		}
		bc, err := code128.Encode(raw)
		return bc, raw, err
	}
}

// renderLinearBarcodeLabel рисует горизонтальный штрихкод с верхним, основным и нижним текстом.
func renderLinearBarcodeLabel(raw, fmtMode string, opts BarcodeOptions) (*image.Gray, error) {
	bc, disp, err := encodeLinear(raw, fmtMode)
	defer func() { _ = disp }()
	if err != nil {
		return nil, fmt.Errorf("ошибка кодирования штрихкода: %w", err)
	}

	lengthMM := opts.LengthMM
	if lengthMM <= 0 {
		lengthMM = 50
	}
	canvas := NewLabelImage(lengthMM)
	W := canvas.Bounds().Dx()
	H := canvas.Bounds().Dy()

	topMargin := maxInt(0, opts.MarginPx)
	bottomMargin := maxInt(0, opts.MarginPx)
	sideMargin := maxInt(0, opts.SideMarginPx)
	gap := maxInt(0, opts.GapPx)

	netH := H - topMargin - bottomMargin
	if netH < 10 {
		netH = 10
	}

	hasTop := strings.TrimSpace(opts.TopText) != ""
	hasBottom := strings.TrimSpace(opts.BottomText) != ""

	maxAllowedGap := int(math.Floor(float64(netH) * 0.05))
	textGap := minInt(maxInt(0, opts.TextGapPx), maxAllowedGap)

	topTextH := 0
	if hasTop {
		topTextH = int(math.Round(float64(netH) * 0.15))
	}
	topGap := 0
	if hasTop {
		topGap = textGap
	}

	bottomTextH := 0
	if hasBottom {
		bottomTextH = int(math.Round(float64(netH) * 0.15))
	}
	bottomGap := 0
	if hasBottom {
		bottomGap = textGap
	}
	bottomTextY := H - bottomMargin - bottomTextH

	centerY := topMargin + topTextH + topGap
	centerH := bottomTextY - bottomGap - centerY
	if centerH < 6 {
		centerH = 6
	}

	fontSize := strings.ToLower(strings.TrimSpace(opts.FontSize))
	if fontSize == "" {
		fontSize = "-1"
	}

	renderBarcode := fontSize != "only_text"
	renderMain := fontSize != "0"
	isPercent := strings.HasSuffix(fontSize, "p")
	fontVal := -1
	if isPercent {
		v, err := strconv.Atoi(strings.TrimSuffix(fontSize, "p"))
		if err != nil {
			return nil, fmt.Errorf("неверный процент шрифта %q", opts.FontSize)
		}
		if v > 95 {
			renderBarcode = false
		} else if v < 5 {
			renderMain = false
		}
		fontVal = v
	} else if fontSize != "only_text" {
		v, err := strconv.Atoi(fontSize)
		if err != nil {
			return nil, fmt.Errorf("неверный размер шрифта %q", opts.FontSize)
		}
		fontVal = v
	}

	var barcodeY, barcodeH, mainY, mainH int
	switch {
	case !renderBarcode:
		mainY = centerY
		mainH = centerH
	case !renderMain:
		barcodeY = centerY
		barcodeH = centerH
	case fontVal == -1 && !isPercent:
		avail := maxInt(2, centerH-gap)
		barcodeH = avail / 2
		mainH = avail - barcodeH
		barcodeY = centerY
		mainY = barcodeY + barcodeH + gap
	case isPercent:
		mainH = int(math.Round(float64(centerH) * (float64(fontVal) / 100.0)))
		if mainH > centerH-gap-2 {
			mainH = maxInt(1, centerH-gap-2)
		}
		mainY = centerY + centerH - mainH
		barcodeY = centerY
		barcodeH = maxInt(1, mainY-gap-barcodeY)
	default:
		mainH = minInt(fontVal, maxInt(1, centerH-gap))
		mainY = centerY + centerH - mainH
		barcodeY = centerY
		barcodeH = maxInt(1, mainY-gap-barcodeY)
	}

	offX := opts.OffsetPx
	if hasTop && topTextH > 0 {
		_ = drawFittedLine(canvas, opts.TopText, opts.FontName, false, offX+sideMargin, topMargin, W-2*sideMargin, topTextH)
	}

	if renderBarcode && barcodeH > 0 {
		bcSide := sideMargin
		targetW := maxInt(10, W-2*bcSide)
		if opts.SafeMode {
			bcSide = maxInt(bcSide, 12)
			targetW = maxInt(10, W-2*bcSide)
		}
		drawSharpBarcodeEx(canvas, bc, offX+bcSide, barcodeY, targetW, barcodeH, opts.SafeMode)
	}

	if renderMain && mainH > 0 {
		_ = drawFittedLine(canvas, raw, opts.FontName, opts.IsMainTextBold, offX+sideMargin, mainY, W-2*sideMargin, mainH)
	}

	if hasBottom && bottomTextH > 0 {
		_ = drawFittedLine(canvas, opts.BottomText, opts.FontName, false, offX+sideMargin, bottomTextY, W-2*sideMargin, bottomTextH)
	}

	return canvas, nil
}

// renderQrLabel рисует QR-код слева или справа, а текстовые блоки — с противоположной стороны.
func render2dLabel(raw, side string, opts BarcodeOptions, isDM bool) (*image.Gray, error) {
	var qrCode barcode.Barcode
	var err error
	if isDM {
		qrCode, err = datamatrix.Encode(raw)
	} else {
		qrCode, err = qr.Encode(raw, qr.M, qr.Auto)
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка кодирования QR: %w", err)
	}

	lengthMM := opts.LengthMM
	if lengthMM <= 0 {
		lengthMM = 50
	}
	canvas := NewLabelImage(lengthMM)
	W := canvas.Bounds().Dx()
	H := canvas.Bounds().Dy()

	margin := maxInt(0, opts.MarginPx)
	sideMargin := maxInt(0, opts.SideMarginPx)
	gap := maxInt(0, opts.GapPx)

	qrSize := maxInt(16, H-2*margin)
	if qrSize > W-2*sideMargin-10 {
		qrSize = maxInt(16, W-2*sideMargin-10)
	}

	offX := opts.OffsetPx
	var qrX, textX, textW int
	if side == "right" {
		qrX = offX + W - sideMargin - qrSize
		textX = offX + sideMargin
		textW = maxInt(10, qrX-gap-textX)
	} else {
		qrX = offX + sideMargin
		textX = qrX + qrSize + gap
		textW = maxInt(10, offX+W-sideMargin-textX)
	}

	drawSharpBarcode(canvas, qrCode, qrX, margin, qrSize, qrSize)

	type textLine struct {
		text   string
		bold   bool
		weight float64
	}
	var lines []textLine
	if strings.TrimSpace(opts.TopText) != "" {
		lines = append(lines, textLine{text: opts.TopText, bold: false, weight: 0.25})
	}
	if strings.TrimSpace(opts.FontSize) != "0" {
		lines = append(lines, textLine{text: raw, bold: opts.IsMainTextBold, weight: 0.50})
	}
	if strings.TrimSpace(opts.BottomText) != "" {
		lines = append(lines, textLine{text: opts.BottomText, bold: false, weight: 0.25})
	}
	if len(lines) == 0 {
		return canvas, nil
	}

	totalWeight := 0.0
	for _, l := range lines {
		totalWeight += l.weight
	}
	textGap := maxInt(0, opts.TextGapPx)
	availH := maxInt(10, (H-2*margin)-textGap*(len(lines)-1))

	curY := margin
	for i, l := range lines {
		h := int(math.Round(float64(availH) * (l.weight / totalWeight)))
		if i == len(lines)-1 {
			h = maxInt(4, H-margin-curY)
		}
		_ = drawFittedLine(canvas, l.text, opts.FontName, l.bold, textX, curY, textW, h)
		curY += h + textGap
	}

	return canvas, nil
}

// drawSharpBarcode вписывает штрихкод/QR без сглаживания (строго чёрные и белые модули),
// обрезая встроенную тихую зону, чтобы отступы задавались исключительно параметром sideMargin.
func drawSharpBarcode(dst *image.Gray, bc barcode.Barcode, x, y, boxW, boxH int) {
	drawSharpBarcodeEx(dst, bc, x, y, boxW, boxH, false)
}

func drawSharpBarcodeEx(dst *image.Gray, bc barcode.Barcode, x, y, boxW, boxH int, safeMode bool) {
	b := bc.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 || boxW <= 0 || boxH <= 0 {
		return
	}

	minX, maxX := b.Max.X, b.Min.X
	minY, maxY := b.Max.Y, b.Min.Y
	for sy := b.Min.Y; sy < b.Max.Y; sy++ {
		for sx := b.Min.X; sx < b.Max.X; sx++ {
			r, g, bl, _ := bc.At(sx, sy).RGBA()
			if (r+g+bl)/3 < 0x8000 {
				if sx < minX {
					minX = sx
				}
				if sx > maxX {
					maxX = sx
				}
				if sy < minY {
					minY = sy
				}
				if sy > maxY {
					maxY = sy
				}
			}
		}
	}
	if minX > maxX || minY > maxY {
		return
	}
	srcRect := image.Rect(minX, minY, maxX+1, maxY+1)
	srcW, srcH := srcRect.Dx(), srcRect.Dy()

	drawW := boxW
	drawH := boxH
	if bc.Metadata().Dimensions == 2 {
		s := minInt(boxW/srcW, boxH/srcH)
		if s < 1 {
			s = 1
		}
		drawW = srcW * s
		drawH = srcH * s
	} else if safeMode {
		// Подбираем масштаб модулей кратно 0.5 (1.0x, 1.5x, 2.0x, 2.5x...), чтобы штрихи и пробелы не слипались
		halfSteps := int(math.Floor(float64(boxW) / (float64(srcW) * 0.5)))
		if halfSteps >= 2 {
			drawW = int(math.Round(float64(srcW) * (float64(halfSteps) * 0.5)))
		}
	}

	offX := x + (boxW-drawW)/2
	offY := y + (boxH-drawH)/2

	for py := 0; py < drawH; py++ {
		sy := srcRect.Min.Y + (py*srcH)/drawH
		for px := 0; px < drawW; px++ {
			sx := srcRect.Min.X + (px*srcW)/drawW
			r, g, bl, _ := bc.At(sx, sy).RGBA()
			if (r+g+bl)/3 < 0x8000 {
				if image.Pt(offX+px, offY+py).In(dst.Bounds()) {
					dst.SetGray(offX+px, offY+py, color.Gray{Y: 0})
				}
			}
		}
	}
}

// drawFittedLine подбирает максимальный кегль по реальным границам чёрных пикселей глифов
// (аналогично path.GetBounds() в C#), чтобы текст заполнял максимум высоты и ширины boxW × boxH.
func drawFittedLine(dst *image.Gray, text, fontName string, bold bool, x, y, boxW, boxH int) error {
	text = strings.TrimSpace(text)
	if text == "" || boxW < 4 || boxH < 4 {
		return nil
	}
	fontPath, err := pickFont(fontName, bold)
	if err != nil {
		return err
	}

	// Рендерим эталонный контур крупным кеглем (64 pt) и находим точный прямоугольник чёрных пикселей
	const refSize = 64.0
	face, err := loadFace(fontPath, refSize)
	if err != nil {
		return err
	}
	defer face.Close()

	advW := font.MeasureString(face, text).Ceil()
	m := face.Metrics()
	tmpW := advW + 20
	tmpH := (m.Ascent + m.Descent).Ceil() + 20
	tmp := image.NewGray(image.Rect(0, 0, tmpW, tmpH))
	for i := range tmp.Pix {
		tmp.Pix[i] = 255
	}

	d := &font.Drawer{
		Dst:  tmp,
		Src:  image.NewUniform(color.Black),
		Face: face,
		Dot:  fixed.P(10, 10+m.Ascent.Ceil()),
	}
	d.DrawString(text)

	minX, maxX := tmpW, 0
	minY, maxY := tmpH, 0
	for py := 0; py < tmpH; py++ {
		for px := 0; px < tmpW; px++ {
			if tmp.GrayAt(px, py).Y < 200 {
				if px < minX {
					minX = px
				}
				if px > maxX {
					maxX = px
				}
				if py < minY {
					minY = py
				}
				if py > maxY {
					maxY = py
				}
			}
		}
	}
	if minX > maxX || minY > maxY {
		return nil
	}

	glyphW := float64(maxX - minX + 1)
	glyphH := float64(maxY - minY + 1)

	scaleX := float64(boxW) / glyphW
	scaleY := float64(boxH) / glyphH
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}

	targetSize := refSize * scale
	if targetSize < 4.0 {
		targetSize = 4.0
	}

	finalFace, err := loadFace(fontPath, targetSize)
	if err != nil {
		return err
	}
	defer finalFace.Close()

	// Повторно измеряем точные пиксельные границы уже на финальном кегле для идеального центрирования
	fAdvW := font.MeasureString(finalFace, text).Ceil()
	fm := finalFace.Metrics()
	fTmpW := fAdvW + 16
	fTmpH := (fm.Ascent + fm.Descent).Ceil() + 16
	fTmp := image.NewGray(image.Rect(0, 0, fTmpW, fTmpH))
	for i := range fTmp.Pix {
		fTmp.Pix[i] = 255
	}
	fd := &font.Drawer{
		Dst:  fTmp,
		Src:  image.NewUniform(color.Black),
		Face: finalFace,
		Dot:  fixed.P(8, 8+fm.Ascent.Ceil()),
	}
	fd.DrawString(text)

	fMinX, fMaxX := fTmpW, 0
	fMinY, fMaxY := fTmpH, 0
	for py := 0; py < fTmpH; py++ {
		for px := 0; px < fTmpW; px++ {
			if fTmp.GrayAt(px, py).Y < 220 {
				if px < fMinX {
					fMinX = px
				}
				if px > fMaxX {
					fMaxX = px
				}
				if py < fMinY {
					fMinY = py
				}
				if py > fMaxY {
					fMaxY = py
				}
			}
		}
	}
	if fMinX > fMaxX || fMinY > fMaxY {
		return nil
	}

	realW := fMaxX - fMinX + 1
	realH := fMaxY - fMinY + 1
	dstX := x + (boxW-realW)/2
	dstY := y + (boxH-realH)/2

	for py := 0; py < realH; py++ {
		for px := 0; px < realW; px++ {
			v := fTmp.GrayAt(fMinX+px, fMinY+py).Y
			if v < 255 && image.Pt(dstX+px, dstY+py).In(dst.Bounds()) {
				if v < dst.GrayAt(dstX+px, dstY+py).Y {
					dst.SetGray(dstX+px, dstY+py, color.Gray{Y: v})
				}
			}
		}
	}
	return nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ean13CheckDigit(d12 string) byte {
	sum := 0
	for i := 0; i < 12; i++ {
		v := int(d12[i] - '0')
		if i%2 == 1 {
			v *= 3
		}
		sum += v
	}
	return byte('0' + ((10 - (sum % 10)) % 10))
}

func ean8CheckDigit(d7 string) byte {
	sum := 0
	for i := 0; i < 7; i++ {
		v := int(d7[i] - '0')
		if i%2 == 0 {
			v *= 3
		}
		sum += v
	}
	return byte('0' + ((10 - (sum % 10)) % 10))
}

func normalizeEan13(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if !reOnlyDigits.MatchString(s) || (len(s) != 12 && len(s) != 13) {
		return "", fmt.Errorf("EAN-13 требует 12 или 13 цифр (получено %d)", len(s))
	}
	cd := ean13CheckDigit(s[:12])
	if len(s) == 12 {
		return s + string(cd), nil
	}
	if s[12] != cd {
		return "", fmt.Errorf("неверная контрольная цифра EAN-13: ожидается %c, в коде %c", cd, s[12])
	}
	return s, nil
}

func normalizeEan8(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if !reOnlyDigits.MatchString(s) || (len(s) != 7 && len(s) != 8) {
		return "", fmt.Errorf("EAN-8 требует 7 или 8 цифр (получено %d)", len(s))
	}
	cd := ean8CheckDigit(s[:7])
	if len(s) == 7 {
		return s + string(cd), nil
	}
	if s[7] != cd {
		return "", fmt.Errorf("неверная контрольная цифра EAN-8: ожидается %c, в коде %c", cd, s[7])
	}
	return s, nil
}
