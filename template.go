// Конструктор этикеток: шаблон из размещённых элементов.
//
// Этикетка описывается не текстом с переносами, а списком элементов —
// надписей и картинок, у каждого свои координаты в миллиметрах. Координаты
// идут в ориентации чтения:
//
//	X — вдоль этикетки (по подаче), 0 слева
//	Y — поперёк этикетки (по головке), 0 сверху
//
// Ширина этикетки задаётся длиной (X), высота всегда 12 мм — это 96 точек
// печатающей головки.
package main

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Element — элемент этикетки: надпись или картинка.
type Element struct {
	Kind string  `json:"kind"`        // text | image | line | frame
	X    float64 `json:"x"`           // мм вдоль этикетки
	Y    float64 `json:"y"`           // мм поперёк этикетки
	W    float64 `json:"w,omitempty"` // мм: ширина элемента
	H    float64 `json:"h,omitempty"`  // мм: высота элемента
	X2   float64 `json:"x2,omitempty"` // мм: конец X для косой линии
	Y2   float64 `json:"y2,omitempty"` // мм: конец Y для косой линии

	// Надпись
	Text   string  `json:"text,omitempty"`
	FontMM float64 `json:"font,omitempty"`   // высота шрифта в мм
	LineMM float64 `json:"line,omitempty"`   // высота строки в мм; 0 — по шрифту
	Family string  `json:"family,omitempty"` // семейство шрифта; пусто — по умолчанию
	Bold   bool    `json:"bold,omitempty"`   // жирное начертание
	Heavy  bool    `json:"heavy,omitempty"`  // обводка букв: ещё жирнее
	Rotate int     `json:"rotate,omitempty"` // поворот: 0, 90, 180 или 270
	Align  string  `json:"align,omitempty"`  // left | center | right (внутри рамки X..X+W)
	// Толщина линий рамки в мм
	Thickness float64 `json:"thickness,omitempty"`

		// Штрихкод / QR
	BcFormat   string `json:"bcFormat,omitempty"`   // auto | code128 | qr | qrl | qrr
BcSafeMode *bool  `json:"bcSafeMode,omitempty"` // безопасный режим масштабирования штрихкода
	TopText    string `json:"topText,omitempty"`    // верхний текст
	BottomText string `json:"bottomText,omitempty"` // нижний текст
	FontSize   string `json:"fontSize,omitempty"`   // -1 (авто), 0 (скрыть), N, Np
	GapPx      int    `json:"gapPx,omitempty"`      // зазор до подписи, px
	// Картинка: путь к файлу или data:URL с base64
	Image string `json:"image,omitempty"`
	Stretch bool `json:"stretch,omitempty"` // true = растянуть без сохранения пропорций
	Transparent bool `json:"transparent,omitempty"`
	HideIfEmpty string `json:"hideIfEmpty,omitempty"`
	Opaque      bool   `json:"opaque,omitempty"`
}

// LabelTemplate — шаблон этикетки для конструктора.
type LabelTemplate struct {
	LengthMM float64   `json:"length"` // длина этикетки в мм
	Elements []Element `json:"elements"`
}

// labelHeightMM — высота этикетки: 12 мм по печатающей головке.
const labelHeightMM = 12.0

// RenderTemplate рисует этикетку по шаблону, подставляя поля из строки данных.
//
// row — столбцы строки данных: {1} — первый, {2} — второй и так далее.
func RenderTemplate(tpl LabelTemplate, row []string) (*image.Gray, error) { return RenderTemplateCtx(tpl, row, 1, 1) }
func RenderTemplateCtx(tpl LabelTemplate, row []string, counter, total int) (*image.Gray, error) { return renderTemplateInternal(tpl, row, time.Now(), counter, total) }
func RenderTemplateUnused(tpl LabelTemplate, row []string) (*image.Gray, error) {
	return RenderTemplateAt(tpl, row, time.Now())
}

// RenderTemplateAt — то же, но с заданным моментом времени: нужно тестам,
// чтобы дата и время на этикетке были предсказуемы.
func RenderTemplateAt(tpl LabelTemplate, row []string, now time.Time) (*image.Gray, error) { return renderTemplateInternal(tpl, row, now, 1, 1) }
func shouldHideElement(el Element, row []string) bool {
spec := strings.TrimSpace(el.HideIfEmpty)
spec = strings.TrimPrefix(spec, "{")
spec = strings.TrimSuffix(spec, "}")
spec = strings.TrimSpace(spec)
if spec == "" {
return false
}
col, err := strconv.Atoi(spec)
if err != nil || col < 1 {
return false
}
if col > len(row) {
return true
}
return strings.TrimSpace(row[col-1]) == ""
}

func blitElement(dst *image.Gray, dstRect image.Rectangle, src *image.Gray, transparent bool) {
if !transparent {
draw.Draw(dst, dstRect, src, image.Point{}, draw.Src)
return
}
inter := dstRect.Intersect(dst.Bounds())
for dy := inter.Min.Y; dy < inter.Max.Y; dy++ {
sy := dy - dstRect.Min.Y
if sy < 0 || sy >= src.Bounds().Dy() {
continue
}
for dx := inter.Min.X; dx < inter.Max.X; dx++ {
sx := dx - dstRect.Min.X
if sx < 0 || sx >= src.Bounds().Dx() {
continue
}
v := src.GrayAt(sx, sy).Y
if v < 245 && v < dst.GrayAt(dx, dy).Y {
dst.SetGray(dx, dy, color.Gray{Y: v})
}
}
}
}

func renderTemplateInternal(tpl LabelTemplate, row []string, now time.Time, counter, total int) (*image.Gray, error) {
img := NewLabelImage(tpl.LengthMM)

for i, el := range tpl.Elements {
if shouldHideElement(el, row) {
continue
}
		var err error
		switch el.Kind {
		case "text":
			err = drawTextElementCtx(img, el, row, now, counter, total)
		case "image":
			err = drawImageElement(img, el)
		case "line":
			err = drawLineElement(img, el)
		case "frame":
			err = drawFrameElement(img, el)
		case "circle":
			err = drawCircleElement(img, el)
		case "diag":
			err = drawDiagElement(img, el)
		case "barcode":
			err = drawBarcodeElementCtx(img, el, row, now, counter, total)
		case "":			err = fmt.Errorf("у элемента %d не указан вид (kind)", i+1)
		default:
			err = fmt.Errorf("элемент %d: неизвестный вид %q", i+1, el.Kind)
		}
		if err != nil {
			return nil, err
		}
	}
	return img, nil
}

// drawTextElement рисует надпись: если заданы классические FontMM/LineMM — использует
// совместимый рендер в мм, иначе заполняет прямоугольный контейнер (X, Y, W, H) по пиксельным границам.
func drawTextElementCtx(img *image.Gray, el Element, row []string, now time.Time, counter, total int) error {
text, err := substituteFieldsCtx(el.Text, row, now, counter, total)
if err != nil {
return err
}
if strings.TrimSpace(text) == "" {
return nil
}

// Классический режим (для старых шаблонов с FontMM / LineMM и без явного FontSize)
if el.FontMM > 0 || el.LineMM > 0 || strings.TrimSpace(el.FontSize) == "" {
return drawTextElementLegacy(img, el, text)
}

lines := strings.Split(text, "\n")
x0 := mmToDots(el.X)
y0 := mmToDots(el.Y)

boxW := mmToDotsMin(el.W)
if el.W <= 0 {
boxW = maxInt(8, img.Bounds().Dx()-x0)
}
boxH := mmToDotsMin(el.H)

angle := ((el.Rotate % 360) + 360) % 360
workW, workH := boxW, boxH
if angle == 90 || angle == 270 {
workW, workH = boxH, boxW
}

lineGap := 2
if len(lines) <= 1 {
lineGap = 0
}
totalGaps := lineGap * (len(lines) - 1)
slotH := float64(workH-totalGaps) / float64(len(lines))
if slotH < 4 {
slotH = 4
}

targetH := slotH
spec := strings.TrimSpace(strings.ToLower(el.FontSize))
if strings.HasSuffix(spec, "p") {
if pct, err := strconv.ParseFloat(strings.TrimSuffix(spec, "p"), 64); err == nil && pct > 0 {
if pct > 100 {
pct = 100
}
targetH = slotH * (pct / 100.0)
}
} else if spec != "" && spec != "-1" && spec != "0" {
if v, err := strconv.ParseFloat(spec, 64); err == nil && v > 0 {
targetH = v
if targetH > slotH {
targetH = slotH
}
}
}
if targetH < 4 {
targetH = 4
}

rowBoxH := int(math.Round(targetH))
if rowBoxH < 4 {
rowBoxH = 4
}
totalBlockH := rowBoxH*len(lines) + totalGaps
if totalBlockH > workH {
rowBoxH = maxInt(4, (workH-totalGaps)/len(lines))
totalBlockH = rowBoxH*len(lines) + totalGaps
}

boxCanvas := whiteCanvas(workW, workH)
startY := (workH - totalBlockH) / 2
if startY < 0 {
startY = 0
}

for i, line := range lines {
if strings.TrimSpace(line) == "" {
continue
}
tmp := whiteCanvas(workW, rowBoxH)
if err := drawFittedLine(tmp, line, el.Family, el.Bold, 0, 0, workW, rowBoxH); err != nil {
return err
}
minX, maxX := workW, -1
for yy := 0; yy < rowBoxH; yy++ {
for xx := 0; xx < workW; xx++ {
if tmp.GrayAt(xx, yy).Y < 128 {
if xx < minX {
minX = xx
}
if xx > maxX {
maxX = xx
}
}
}
}
if maxX < minX {
continue
}
glyphW := maxX - minX + 1
dstX := 0
switch strings.ToLower(strings.TrimSpace(el.Align)) {
case "center":
dstX = (workW - glyphW) / 2
case "right":
dstX = workW - glyphW
default:
dstX = 0
}
if dstX < 0 {
dstX = 0
}
dstY := startY + i*(rowBoxH+lineGap)
draw.Draw(boxCanvas, image.Rect(dstX, dstY, dstX+glyphW, dstY+rowBoxH), tmp, image.Pt(minX, 0), draw.Src)
}

if angle != 0 {
boxCanvas = rotateGray(boxCanvas, angle)
}

blitElement(img, image.Rect(x0, y0, x0+boxCanvas.Bounds().Dx(), y0+boxCanvas.Bounds().Dy()), boxCanvas, el.Transparent)
return nil
}

func drawTextElementLegacy(img *image.Gray, el Element, text string) error {
fontPath, err := pickFont(el.Family, el.Bold)
if err != nil {
return err
}
lines := strings.Split(text, "\n")
fontMM := el.FontMM
if fontMM <= 0 {
boxW := mmToDotsMin(el.W)
if boxW <= 0 {
boxW = img.Bounds().Dx() - mmToDots(el.X)
}
boxH := mmToDotsMin(el.H)
if boxH <= 0 {
boxH = img.Bounds().Dy() - mmToDots(el.Y)
}
if el.Heavy {
boxW -= 2
}
size, err := fitFontSize(fontPath, lines, boxW, boxH)
if err != nil {
return err
}
fontMM = size / dotsPerMM
}

face, err := loadFace(fontPath, fontMM*dotsPerMM)
if err != nil {
return err
}
defer face.Close()

metrics := face.Metrics()
ascent := metrics.Ascent.Ceil()
lineH := metrics.Height.Ceil()
if el.LineMM > 0 {
lineH = int(math.Round(el.LineMM * dotsPerMM))
}
if lineH <= 0 {
lineH = int(fontMM*dotsPerMM) + 2
}

x0 := int(math.Round(el.X * dotsPerMM))
y0 := int(math.Round(el.Y * dotsPerMM))
boxW := int(math.Round(el.W * dotsPerMM))

if el.H > 0 {
boxH := mmToDotsMin(el.H)
blockH := lineH * len(lines)
if boxH > blockH {
y0 += (boxH - blockH) / 2
}
}

if angle := ((el.Rotate % 360) + 360) % 360; angle != 0 {
blockW, blockH := 0, lineH*len(lines)
for _, line := range lines {
if w := font.MeasureString(face, line).Ceil(); w > blockW {
blockW = w
}
}
if blockW < 1 {
blockW = 1
}
tmp := whiteCanvas(blockW, blockH)
drawTextLines(tmp, face, lines, 0, 0, lineH, blockW, ascent, "left", 0, el.Heavy)
rot := rotateGray(tmp, angle)

rw, rh := rot.Bounds().Dx(), rot.Bounds().Dy()
px := x0
switch strings.ToLower(el.Align) {
case "center":
if boxW > 0 {
px = x0 + (boxW-rw)/2
} else {
px = x0 - rw/2
}
case "right":
if boxW > 0 {
px = x0 + boxW - rw
} else {
px = x0 - rw
}
}
draw.Draw(img, image.Rect(px, y0, px+rw, y0+rh), rot, image.Point{}, draw.Src)
return nil
}

drawTextLines(img, face, lines, x0, y0, lineH, boxW, ascent, el.Align, 1, el.Heavy)
return nil
}

// fitMarginDots — запас при автоподборе кегля, в точках.
//
// Без него автоподбор упирает текст в самые края рамки: на печати это
// означает обрезку, если этикетка ляжет чуть со сдвигом. 6 точек = 0,75 мм.
const fitMarginDots = 6

// fitFontSize подбирает наибольший кегль, при котором текст влезает
// в рамку boxW × boxH (в точках). Размер возвращается в точках изображения.
func fitFontSize(fontPath string, lines []string, boxW, boxH int) (float64, error) {
	boxW -= 2 * fitMarginDots
	if boxW < 4 || boxH < 4 {
		return 4, nil
	}
	best := 4.0
	for size := float64(boxH); size >= 4; size -= 0.5 {
		face, err := loadFace(fontPath, size)
		if err != nil {
			continue
		}
		w, h := measureBlock(face, lines)
		face.Close()
		if w <= boxW && h <= boxH {
			best = size
			break
		}
	}
	return best, nil
}

// drawTextLines рисует строки надписи с выравниванием внутри рамки boxW.
// step — шаг между строками в единицах lineH (нужен, когда строка одна
// и рисовать её надо на своём холсте).
func drawTextLines(dst *image.Gray, face font.Face, lines []string,
	x0, y0, lineH, boxW, ascent int, align string, step int, heavy bool) {
	if step < 1 {
		step = 1
	}
	for i, line := range lines {
		lineW := font.MeasureString(face, line).Ceil()
		x := x0
		switch strings.ToLower(align) {
		case "center":
			if boxW > 0 {
				x = x0 + (boxW-lineW)/2
			} else {
				x = x0 - lineW/2
			}
		case "right":
			if boxW > 0 {
				x = x0 + boxW - lineW
			} else {
				x = x0 - lineW
			}
		}
		y := y0 + ascent + i*lineH*step
		d := &font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(color.Black),
			Face: face,
			Dot:  fixed.P(x, y),
		}
		d.DrawString(line)
		if heavy {
			// Обводка: та же строка со сдвигом на точку по кругу. Шрифт уже
			// жирный, а этого мало — на этикетке хочется ещё плотнее.
			for _, off := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				d.Dot = fixed.P(x+off[0], y+off[1])
				d.DrawString(line)
			}
		}
	}
}

// mmToDots переводит миллиметры в точки печати.
func mmToDots(mm float64) int {
	return int(math.Round(mm * dotsPerMM))
}

// mmToDotsMin переводит миллиметры в точки, но не меньше одной:
// тонкая рамка 0,2 мм должна остаться видимой.
func mmToDotsMin(mm float64) int {
	n := mmToDots(mm)
	if n < 1 {
		n = 1
	}
	return n
}

// fillRect заливает прямоугольник чёрным, обрезая по краям холста.
func fillRect(img *image.Gray, x, y, w, h int) {
	r := image.Rect(x, y, x+w, y+h).Intersect(img.Bounds())
	if r.Empty() {
		return
	}
	draw.Draw(img, r, image.NewUniform(color.Black), image.Point{}, draw.Src)
}

// drawLineElement рисует залитый прямоугольник: разделитель между строками
// или сплошную плашку. Тонкая линия — это тот же прямоугольник с малой высотой.
func drawLineElement(img *image.Gray, el Element) error {
	w, h := mmToDotsMin(el.W), mmToDotsMin(el.H)
	if el.W <= 0 || el.H <= 0 {
		return fmt.Errorf("у линии не заданы размеры (w и h в мм)")
	}
	fillRect(img, mmToDots(el.X), mmToDots(el.Y), w, h)
	return nil
}

// drawFrameElement рисует рамку: четыре полосы по сторонам прямоугольника.
func drawFrameElement(img *image.Gray, el Element) error {
	if el.W <= 0 || el.H <= 0 {
		return fmt.Errorf("у рамки не заданы размеры (w и h в мм)")
	}
	t := el.Thickness
	if t <= 0 {
		t = 0.3
	}
	x, y := mmToDots(el.X), mmToDots(el.Y)
	w, h, tw := mmToDots(el.W), mmToDots(el.H), mmToDotsMin(t)

	fillRect(img, x, y, w, tw)      // верх
	fillRect(img, x, y+h-tw, w, tw) // низ
	fillRect(img, x, y, tw, h)      // левая
	fillRect(img, x+w-tw, y, tw, h) // правая
	return nil
}

func drawCircleElement(img *image.Gray, el Element) error {
if el.W <= 0 || el.H <= 0 {
return fmt.Errorf("у окружности не заданы размеры (w и h в мм)")
}
t := el.Thickness
if t <= 0 {
t = 0.3
}
x0, y0 := mmToDots(el.X), mmToDots(el.Y)
w, h := mmToDots(el.W), mmToDots(el.H)
tw := float64(mmToDotsMin(t))

rxOuter := float64(w) / 2.0
ryOuter := float64(h) / 2.0
if rxOuter < 1 || ryOuter < 1 {
return nil
}
rxInner := math.Max(0, rxOuter-tw)
ryInner := math.Max(0, ryOuter-tw)

cx := float64(x0) + rxOuter
cy := float64(y0) + ryOuter
bounds := img.Bounds()

for py := y0; py <= y0+h; py++ {
if py < bounds.Min.Y || py >= bounds.Max.Y {
continue
}
dy := (float64(py) + 0.5) - cy
for px := x0; px <= x0+w; px++ {
if px < bounds.Min.X || px >= bounds.Max.X {
continue
}
dx := (float64(px) + 0.5) - cx
dOut := (dx*dx)/(rxOuter*rxOuter) + (dy*dy)/(ryOuter*ryOuter)
if dOut > 1.0 {
continue
}
if rxInner > 0 && ryInner > 0 {
dIn := (dx*dx)/(rxInner*rxInner) + (dy*dy)/(ryInner*ryInner)
if dIn < 1.0 {
if !el.Transparent {
img.SetGray(px, py, color.Gray{Y: 255})
}
continue
}
}
img.SetGray(px, py, color.Gray{Y: 0})
}
}
return nil
}

// rotateGray поворачивает изображение на 90, 180 или 270 градусов.
func rotateGray(src *image.Gray, angle int) *image.Gray {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	switch ((angle % 360) + 360) % 360 {
	case 90:
		dst := image.NewGray(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				// по часовой стрелке
				dst.SetGray(h-1-y, x, src.GrayAt(x, y))
			}
		}
		return dst
	case 180:
		dst := image.NewGray(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.SetGray(w-1-x, h-1-y, src.GrayAt(x, y))
			}
		}
		return dst
	case 270:
		dst := image.NewGray(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.SetGray(y, w-1-x, src.GrayAt(x, y))
			}
		}
		return dst
	default:
		return src
	}
}

// pickFont выбирает файл шрифта для элемента.
//
// Пустое семейство — встроенный шрифт по умолчанию. Если названного
// семейства в системе нет, это ошибка: молча подменить шрифт нельзя,
// иначе этикетка выйдет не такой, как задумано, и заметит это только
// человек, глядя на бумагу.
func pickFont(family string, bold bool) (string, error) {
	if strings.TrimSpace(family) == "" {
		return findFontFile()
	}
	path, ok := resolveFontFamily(family, bold)
	if !ok {
		return "", fmt.Errorf("шрифт %q не найден в системе — выберите другой в конструкторе", family)
	}
	return path, nil
}

// drawImageElement рисует картинку в её рамке (с сохранением пропорций или растяжением).
func drawImageElement(img *image.Gray, el Element) error {
if el.Image == "" {
return fmt.Errorf("у картинки не указан источник")
}
src, err := loadImageSource(el.Image)
if err != nil {
return err
}
wMM, hMM := el.W, el.H
if wMM <= 0 || hMM <= 0 {
return fmt.Errorf("у картинки не заданы размеры (w и h в мм)")
}
boxW := int(math.Round(wMM * dotsPerMM))
boxH := int(math.Round(hMM * dotsPerMM))
if boxW < 1 || boxH < 1 {
return fmt.Errorf("картинка получается меньше точки")
}

angle := ((el.Rotate % 360) + 360) % 360
workW, workH := boxW, boxH
if angle == 90 || angle == 270 {
workW, workH = boxH, boxW
}

sb := src.Bounds()
sw, sh := sb.Dx(), sb.Dy()
if sw < 1 || sh < 1 {
return fmt.Errorf("пустое изображение")
}

dw, dh := workW, workH
if !el.Stretch {
scale := math.Min(float64(workW)/float64(sw), float64(workH)/float64(sh))
dw = maxInt(1, int(math.Round(float64(sw)*scale)))
dh = maxInt(1, int(math.Round(float64(sh)*scale)))
}

scaled := image.NewGray(image.Rect(0, 0, dw, dh))
drawScaled(scaled, src)

box := whiteCanvas(workW, workH)
offX := (workW - dw) / 2
offY := (workH - dh) / 2
draw.Draw(box, image.Rect(offX, offY, offX+dw, offY+dh), scaled, image.Point{}, draw.Src)

if angle != 0 {
box = rotateGray(box, angle)
}

x0 := int(math.Round(el.X * dotsPerMM))
y0 := int(math.Round(el.Y * dotsPerMM))
blitElement(img, image.Rect(x0, y0, x0+box.Bounds().Dx(), y0+box.Bounds().Dy()), box, el.Transparent)
return nil
}

// drawScaled уменьшает или увеличивает картинку в подготовленный холст.
func drawScaled(dst *image.Gray, src image.Image) {
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
}

// loadImageSource читает картинку из файла или из data:URL с base64.
//
// data:URL удобен конструктору: картинку можно вставить прямо в шаблон,
// и он останется самодостаточным — одним файлом.
func loadImageSource(src string) (image.Image, error) {
	if strings.HasPrefix(src, "data:") {
		comma := strings.Index(src, ",")
		if comma < 0 {
			return nil, fmt.Errorf("повреждённый data:URL картинки")
		}
		raw, err := base64.StdEncoding.DecodeString(src[comma+1:])
		if err != nil {
			return nil, fmt.Errorf("разбор base64 картинки: %w", err)
		}
		img, _, err := image.Decode(strings.NewReader(string(raw)))
		if err != nil {
			return nil, fmt.Errorf("разбор картинки из шаблона: %w", err)
		}
		return img, nil
	}
	f, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("разбор картинки %s: %w", src, err)
	}
	return img, nil
}

// Подстановки даты и времени. Порядок важен: сначала самая длинная,
// иначе «{дата-время}» превратится в «18.09.2026-время».
var (
	reDateTime = regexp.MustCompile(`(?i)\{дата-время\}`)
	reDate     = regexp.MustCompile(`(?i)\{дата\}`)
	reTime     = regexp.MustCompile(`(?i)\{время\}`)
)

// substituteFields заменяет поля данных и дату со временем.
//
// Столбцы данных: {1}, {2}… Дата и время подставляются в момент печати:
// {дата} → 18.09.2026, {время} → 00:20, {дата-время} → 18.09.2026 00:20.
//
// Ссылка на столбец, которого нет, — ошибка: иначе на этикетке напечаталось
// бы литеральное «{3}», и брак был бы виден только на бумаге.
var reFmtField = regexp.MustCompile(`\{([1-9][0-9]*)(?:([LlRr:])([1-9]|1[0-9]|20))\}`)
var reCounterTag = regexp.MustCompile(`\{([CcNn])(?:([LlRr:])([1-9]|1[0-9]|20))?\}`)

func formatFieldVal(val, dir string, width int) string {
r := []rune(strings.TrimSpace(val))
if width < 1 {
return string(r)
}
if width > 20 {
width = 20
}
if strings.EqualFold(dir, "R") {
if len(r) >= width {
return string(r[:width])
}
return string(r) + strings.Repeat("0", width-len(r))
}
// По умолчанию ('L' или ':') — дополняем нулями слева или обрезаем слева
if len(r) >= width {
return string(r[len(r)-width:])
}
return strings.Repeat("0", width-len(r)) + string(r)
}

func substituteFields(text string, row []string, now time.Time) (string, error) {
return substituteFieldsCtx(text, row, now, 1, 1)
}
func substituteFieldsCtx(text string, row []string, now time.Time, counter, total int) (string, error) {
text = reCounterTag.ReplaceAllStringFunc(text, func(m string) string {
sub := reCounterTag.FindStringSubmatch(m)
raw := strconv.Itoa(counter)
if strings.EqualFold(sub[0+1], "N") { raw = strconv.Itoa(total) }
if sub[0+2] != "" && sub[0+3] != "" { w, _ := strconv.Atoi(sub[0+3]); return formatFieldVal(raw, sub[0+2], w) }
return raw
})
text = reDateTime.ReplaceAllString(text, now.Format("02.01.2006 15:04"))
text = reDate.ReplaceAllString(text, now.Format("02.01.2006"))
text = reTime.ReplaceAllString(text, now.Format("15:04"))

var fmtErr error
text = reFmtField.ReplaceAllStringFunc(text, func(m string) string {
if fmtErr != nil {
return m
}
sub := reFmtField.FindStringSubmatch(m)
colIdx, _ := strconv.Atoi(sub[1])
dir := sub[2]
width, _ := strconv.Atoi(sub[3])
if colIdx < 1 || colIdx > len(row) {
fmtErr = fmt.Errorf("элемент использует %s, а в данных только %d столбц(а/ов)", m, len(row))
return m
}
return formatFieldVal(row[colIdx-1], dir, width)
})
if fmtErr != nil {
return "", fmtErr
}

for i, v := range row {
text = strings.ReplaceAll(text, fmt.Sprintf("{%d}", i+1), v)
}
if bad := rePlaceholder.FindString(text); bad != "" {
return "", fmt.Errorf(
"элемент использует %s, а в данных только %d столбц(а/ов)", bad, len(row))
}
return text, nil
}

// drawBarcodeElement рисует штрихкод или QR-код строго внутри рамки (X, Y, W, H).
func drawBarcodeElementCtx(img *image.Gray, el Element, row []string, now time.Time, counter, total int) error {
codeVal, err := substituteFieldsCtx(el.Text, row, now, counter, total)
if err != nil {
return err
}
if strings.TrimSpace(codeVal) == "" {
return nil
}

cfg := LoadConfig()
wMM := el.W
if wMM <= 0 {
wMM = float64(img.Bounds().Dx()) / dotsPerMM
}
hMM := el.H
if hMM <= 0 {
hMM = labelHeightMM
}
targetH := mmToDotsMin(hMM)

// Строим код на его реальную высоту targetH (без искусственного сжатия после рендера)
origHead := printheadPixels
_ = origHead
safeBc := cfg.BcSafeMode; if el.BcSafeMode != nil { safeBc = *el.BcSafeMode }
opts := BarcodeOptions{
SafeMode:       safeBc,
Data:           codeVal,
Format:         el.BcFormat,
LengthMM:       wMM,
FontName:       el.Family,
IsMainTextBold: el.Bold,
FontSize:       el.FontSize,
SideMarginPx:   0,
MarginPx:       0,
GapPx:          maxInt(0, el.GapPx),
TextGapPx:      0,
}

bcImg, err := RenderBarcodeLabel(opts, cfg)
if err != nil {
return err
}

x0, y0 := mmToDots(el.X), mmToDots(el.Y)
if bcImg.Bounds().Dy() != targetH {
scaled := image.NewGray(image.Rect(0, 0, bcImg.Bounds().Dx(), targetH))
drawScaled(scaled, bcImg)
bcImg = scaled
}
blitElement(img, image.Rect(x0, y0, x0+bcImg.Bounds().Dx(), y0+bcImg.Bounds().Dy()), bcImg, el.Transparent)
return nil
}

func drawDiagElement(img *image.Gray, el Element) error {
t := el.Thickness
if t <= 0 {
t = 0.4
}
x1, y1 := el.X*dotsPerMM, el.Y*dotsPerMM
x2, y2 := el.X2*dotsPerMM, el.Y2*dotsPerMM
halfW := math.Max(0.5, (t*dotsPerMM)/2.0)

minX := int(math.Floor(math.Min(x1, x2) - halfW - 1))
maxX := int(math.Ceil(math.Max(x1, x2) + halfW + 1))
minY := int(math.Floor(math.Min(y1, y2) - halfW - 1))
maxY := int(math.Ceil(math.Max(y1, y2) + halfW + 1))

dx, dy := x2-x1, y2-y1
segLenSq := dx*dx + dy*dy
halfWSq := halfW * halfW
bounds := img.Bounds()

for py := minY; py <= maxY; py++ {
if py < bounds.Min.Y || py >= bounds.Max.Y {
continue
}
cy := float64(py) + 0.5
for px := minX; px <= maxX; px++ {
if px < bounds.Min.X || px >= bounds.Max.X {
continue
}
cx := float64(px) + 0.5
var distSq float64
if segLenSq == 0 {
distSq = (cx-x1)*(cx-x1) + (cy-y1)*(cy-y1)
} else {
u := ((cx-x1)*dx + (cy-y1)*dy) / segLenSq
if u < 0 {
u = 0
} else if u > 1 {
u = 1
}
projX := x1 + u*dx
projY := y1 + u*dy
distSq = (cx-projX)*(cx-projX) + (cy-projY)*(cy-projY)
}
if distSq <= halfWSq {
img.SetGray(px, py, color.Gray{Y: 0})
}
}
}
return nil
}
