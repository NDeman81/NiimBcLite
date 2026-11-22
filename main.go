package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/text/encoding/charmap"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type SavedTemplate struct {
	Name         string     `json:"name"`
	Length       float64    `json:"length"`
	Font         float64    `json:"font,omitempty"`
	Density      int        `json:"density,omitempty"`
	Label        string     `json:"label,omitempty"`
	Copies       int        `json:"copies,omitempty"`
	Flip         bool       `json:"flip,omitempty"`
	Threshold    int        `json:"threshold,omitempty"`
	BcFormat     string     `json:"bcFormat,omitempty"`
	BcSafeMode   *bool      `json:"bcSafeMode,omitempty"`
	GapPx        *int       `json:"gapPx,omitempty"`
	TextGapPx    *int       `json:"textGapPx,omitempty"`
	FontName     string     `json:"fontName,omitempty"`
	FontSize     string     `json:"fontSize,omitempty"`
	Bold         bool       `json:"bold,omitempty"`
	MarginPx     *int       `json:"marginPx,omitempty"`
	SideMarginPx *int       `json:"sideMarginPx,omitempty"`
	OffsetPx     *int       `json:"offsetPx,omitempty"`
	Template     string     `json:"template,omitempty"`
	CounterStart *int       `json:"counterStart,omitempty"`
	Elements     []Element  `json:"elements,omitempty"`
	System       *AppConfig `json:"system,omitempty"`
}

var activeCfg AppConfig
var rePlaceholder = regexp.MustCompile("\\{\\d+\\}")

func buildTextOptionsFromConfig() TextRenderOptions {
	return TextRenderOptions{
		FontName:     activeCfg.FontName,
		Bold:         activeCfg.IsMainTextBold,
		FontSize:     activeCfg.FontSize,
		MarginPx:     activeCfg.MarginPx,
		SideMarginPx: activeCfg.SideMarginPx,
		OffsetPx:     activeCfg.OffsetPx,
	}
}

func ensureUTF8(s string, forceCP1251 bool) string {
	if s == "" {
		return s
	}
	if !forceCP1251 && utf8.ValidString(s) {
		return s
	}
	dec, err := charmap.Windows1251.NewDecoder().String(s)
	if err != nil {
		return s
	}
	return dec
}

func main() {
	cfg := LoadConfig()
	activeCfg = cfg

	fs := flag.NewFlagSet("niimbc-cli", flag.ExitOnError)
	addrFlag := fs.String("address", "", "printer Bluetooth MAC address")
	lenFlag := fs.Float64("length", 0, "label length in mm")
	densFlag := fs.Int("density", 0, "print density 1..3")
	lblFlag := fs.String("label", "", "label type (default: withgaps; auto = read RFID)")
	copFlag := fs.Int("copies", 0, "number of copies")
	flipFlag := fs.Bool("flip", false, "rotate 180 degrees")
	fmtFlag := fs.String("format", "", "code format: auto, code128, code39, ean13, ean8, pdf417, qr, qrl, qrr, datamatrix, dml, dmr")
	topFlag := fs.String("top", "", "top caption text")
	botFlag := fs.String("bot", "", "bottom caption text")
	safeFlag := fs.Bool("safe", false, "barcode safe mode")
.\.git\fontNameFlag := fs.String("font-name", "", "font family name (e.g. Arial, Segoe UI)")
.\.git\fontSizeFlag := fs.String("font-size", "", "main text size (-1=auto, 0=hide, only_text, 24, 30p)")
.\.git\boldFlag := fs.Bool("bold", false, "bold main text")
.\.git\marginFlag := fs.Int("margin", -1, "top/bottom margin in px")
.\.git\sideMarginFlag := fs.Int("side-margin", -1, "left/right margin in px")
.\.git\gapFlag := fs.Int("gap", -1, "gap between barcode and main text in px")
.\.git\textGapFlag := fs.Int("text-gap", -1, "gap between top/bottom captions and center in px")
.\.git\offsetFlag := fs.Int("offset", 0, "horizontal offset in px")
	outFlag := fs.String("out", "", "output PNG path for preview")
	quietFlag := fs.Bool("quiet", false, "quiet mode (errors only)")
	cp1251Flag := fs.Bool("cp1251", false, "decode input arguments from Windows-1251 (CP1251)")
	saveCfgFlag := fs.Bool("save-config", false, "save flags to config.json")

	var flagsArgs, posArgs []string
	rawArgs := os.Args[1:]
	for i := 0; i < len(rawArgs); i++ {
		a := rawArgs[i]
		if strings.HasPrefix(a, "-") && a != "-" {
			flagsArgs = append(flagsArgs, a)
			if !strings.Contains(a, "=") && i+1 < len(rawArgs) {
				name := strings.TrimLeft(a, "-")
				if name != "flip" && name != "safe" && name != "cp1251" && name != "bold" && name != "bold" && name != "quiet" && name != "save-config" && name != "h" && name != "help" {
					flagsArgs = append(flagsArgs, rawArgs[i+1])
					i++
				}
			}
		} else {
			posArgs = append(posArgs, a)
		}
	}
	_ = fs.Parse(flagsArgs)

	for i := range posArgs {
		posArgs[i] = ensureUTF8(posArgs[i], *cp1251Flag)
	}
	*topFlag = ensureUTF8(*topFlag, *cp1251Flag)
	*botFlag = ensureUTF8(*botFlag, *cp1251Flag)
	if *quietFlag {
		SetLogLevel("error")
	} else {
		SetLogLevel(cfg.LogLevel)
	}

	if *addrFlag != "" {
		cfg.Address = strings.TrimSpace(*addrFlag)
	}
	if *lenFlag > 0 {
		cfg.Length = *lenFlag
	}
	if *densFlag >= 1 && *densFlag <= 3 {
		cfg.Density = *densFlag
	}
	if *lblFlag != "" {
		cfg.Label = strings.TrimSpace(*lblFlag)
	}
	if *copFlag > 0 {
		cfg.Copies = *copFlag
	}
	if *flipFlag {
		cfg.Flip = true
	}
	if *fmtFlag != "" {
		cfg.BcFormat = strings.TrimSpace(*fmtFlag)
	}
	if *safeFlag {
		cfg.BcSafeMode = true
	}
	activeCfg = cfg

	if *saveCfgFlag {
		if err := SaveConfig(cfg); err != nil {
			fatal(err)
		}
		LogKey("Settings saved to config.json")
		if len(posArgs) == 0 {
			return
		}
	}

	if len(posArgs) == 0 {
		printUsage()
		return
	}

	cmd := strings.ToLower(posArgs[0])
	args := posArgs[1:]

	switch cmd {
	case "scan":
		mac, err := ScanPrinter("N1-", 12*time.Second, !*quietFlag)
		if err != nil {
			fatal(err)
		}
		cfg.Address = mac
		if p, cErr := ConnectMAC(mac, false, true); cErr == nil {
			if rf, rErr := p.ReadRfid(false); rErr == nil && rf.TagPresent && rf.LabelTypeName() != "" {
				cfg.Label = rf.LabelTypeName()
			}
			_ = p.Close()
		}
		if strings.TrimSpace(cfg.Label) == "" {
			cfg.Label = "withgaps"
		}
		_ = SaveConfig(cfg)
		LogKey("Printer found: %s (label type: %s) - saved to config.json", mac, cfg.Label)

	case "info":
		p := mustConnect(cfg, !*quietFlag)
		defer func() { _ = p.Close() }()
		info, err := p.Handshake()
		if err != nil {
			fatal(err)
		}
		fmt.Printf("Model: %d | S/N: %s | Firmware: %s | Battery: %d\n", info.ModelID, info.Serial, info.Firmware, info.Battery)
		for _, isRib := range []bool{false, true} {
			name := "Label roll"
			if isRib {
				name = "Ribbon"
			}
			if rf, err := p.ReadRfid(isRib); err == nil && rf.TagPresent {
				fmt.Printf("%s: type=%s, total=%d, used=%d, remaining=%d\n", name, rf.LabelTypeName(), rf.AllPaper, rf.UsedPaper, rf.Leftover())
			} else {
				fmt.Printf("%s: no RFID tag detected\n", name)
			}
		}

	case "barcode":
		if len(args) == 0 {
			fatal(fmt.Errorf("missing barcode data"))
		}
		img, err := RenderBarcodeLabel(BarcodeOptions{
			Data: args[0], Format: cfg.BcFormat, LengthMM: cfg.Length,
			TopText: *topFlag, BottomText: *botFlag, FontName: cfg.FontName,
			IsMainTextBold: cfg.IsMainTextBold, FontSize: cfg.FontSize,
			OffsetPx: cfg.OffsetPx, SideMarginPx: cfg.SideMarginPx,
			MarginPx: cfg.MarginPx, GapPx: cfg.GapPx, TextGapPx: cfg.TextGapPx,
			SafeMode: cfg.BcSafeMode,
		}, cfg)
		if err != nil {
			fatal(err)
		}
		printSingleImage(img, cfg, "", !*quietFlag)

	case "text":
		if len(args) == 0 {
			fatal(fmt.Errorf("missing text lines"))
		}
		img, err := RenderTextWithOptions(args, cfg.Length, buildTextOptionsFromConfig())
		if err != nil {
			fatal(err)
		}
		printSingleImage(img, cfg, "", !*quietFlag)

	case "image":
		if len(args) == 0 {
			fatal(fmt.Errorf("missing image file path"))
		}
		img, err := loadAndFitImage(args[0], cfg)
		if err != nil {
			fatal(err)
		}
		printSingleImage(img, cfg, "", !*quietFlag)

	case "templates":
		dir, _ := os.UserConfigDir()
		tplFile := filepath.Join(dir, "niimbot", "templates.json")
		data, err := os.ReadFile(tplFile)
		if err != nil {
			fmt.Printf("Templates file not found (%s)\n", tplFile)
			return
		}
		var list []SavedTemplate
		if err := json.Unmarshal(data, &list); err != nil || len(list) == 0 {
			fmt.Println("No saved templates found.")
			return
		}
		fmt.Printf("Saved templates (%s):\n", tplFile)
		for i, t := range list {
			fmt.Printf("  %d. %s (length: %.0f mm, elements: %d)\n", i+1, t.Name, t.Length, len(t.Elements))
		}
	case "template":
		if len(args) == 0 {
			dir, _ := os.UserConfigDir()
			tplFile := filepath.Join(dir, "niimbot", "templates.json")
			data, err := os.ReadFile(tplFile)
			if err != nil {
				fmt.Printf("Templates file not found (%s)\n", tplFile)
				return
			}
			var list []SavedTemplate
			if err := json.Unmarshal(data, &list); err != nil || len(list) == 0 {
				fmt.Println("No saved templates found.")
				return
			}
			fmt.Printf("Saved templates (%s):\n", tplFile)
			for i, t := range list {
				fmt.Printf("  %d. %s (length: %.0f mm, elements: %d)\n", i+1, t.Name, t.Length, len(t.Elements))
			}
			return
		}
		dataRow := ""
		if len(args) > 1 {
			dataRow = args[1]
		}
		img, tplLabel, err := renderSingleTemplate(args[0], dataRow, cfg)
		if err != nil {
			fatal(err)
		}
		printSingleImage(img, cfg, tplLabel, !*quietFlag)

	case "preview":
		if len(args) == 0 {
			fatal(fmt.Errorf("specify preview target: barcode | text | image | template"))
		}
		outPath := strings.TrimSpace(*outFlag)
		if outPath == "" {
			outPath = "preview.png"
		}
		img, err := buildPreviewAny(args[0], args[1:], *topFlag, *botFlag, cfg)
		if err != nil {
			fatal(err)
		}
		if err := saveGrayPng(outPath, BinarizeAt(img, uint8(cfg.Threshold))); err != nil {
			fatal(err)
		}
		LogKey("Preview saved: %s (%dx%d px)", outPath, img.Bounds().Dx(), img.Bounds().Dy())

	default:
		printUsage()
		os.Exit(1)
	}
}

func resolveFastLabel(p *Printer, tplLabel string, cfg AppConfig) (string, error) {
	// Приоритет: явный флаг/конфиг (если не auto) -> тип из шаблона -> withgaps по умолчанию
	cand := strings.ToLower(strings.TrimSpace(cfg.Label))
	if cand == "" {
		cand = strings.ToLower(strings.TrimSpace(tplLabel))
	}
	if cand == "" {
		cand = "withgaps"
	}
	if cand == "auto" {
		rf, err := p.ReadRfid(false)
		if err != nil {
			return "", fmt.Errorf("failed to read label RFID: %w", err)
		}
		if !rf.TagPresent {
			return "", fmt.Errorf("no label RFID tag found; specify label type explicitly (--label withgaps)")
		}
		cand = rf.LabelTypeName()
	}
	lt, ok := labelTypes[cand]
	if !ok {
		return "", fmt.Errorf("unknown label type %q", cand)
	}
	if lt != 1 && lt != 3 && lt != 5 && lt != 10 && lt != 11 {
		return "", fmt.Errorf("label type %q is not supported by NIIMBOT N1", cand)
	}
	return cand, nil
}

func mustConnect(cfg AppConfig, verbose bool) *Printer {
	mac := strings.TrimSpace(cfg.Address)
	foundByName := false
	if mac == "" {
		found, err := ScanPrinter("N1-", 12*time.Second, verbose)
		if err != nil {
			fatal(err)
		}
		mac = found
		foundByName = true
	}
	p, err := ConnectMAC(mac, verbose, foundByName)
	if err != nil {
		fatal(err)
	}
	p.printMode = cfg.PrintMode
	return p
}

func printSingleImage(img *image.Gray, cfg AppConfig, tplLabel string, verbose bool) {
	rows := ImageToRowsAt(img, cfg.Flip, uint8(cfg.Threshold))
	p := mustConnect(cfg, verbose)
	defer func() { _ = p.Close() }()

	lblName, err := resolveFastLabel(p, tplLabel, cfg)
	if err != nil {
		fatal(err)
	}
	copies := cfg.Copies
	if copies < 1 {
		copies = 1
	}
	if err := p.PrintPages([][]Row{rows}, byte(cfg.Density), labelTypes[lblName], copies, nil); err != nil {
		fatal(err)
	}
}

func loadAndFitImage(path string, cfg AppConfig) (*image.Gray, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	src, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("unsupported image format: %w", err)
	}
	res := FitToLabelEx(src, cfg.Length, cfg.MarginPx, cfg.SideMarginPx, cfg.ImgStretch)
	if cfg.OffsetPx != 0 {
		shiftImageX(res, cfg.OffsetPx)
	}
	return res, nil
}

func findTemplateByNameOrPath(nameOrPath string) (*SavedTemplate, error) {
	if data, err := os.ReadFile(nameOrPath); err == nil {
		var single SavedTemplate
		if json.Unmarshal(data, &single) == nil && (len(single.Elements) > 0 || single.Template != "") {
			return &single, nil
		}
		var arr []SavedTemplate
		if json.Unmarshal(data, &arr) == nil && len(arr) > 0 {
			return &arr[0], nil
		}
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	tplFile := filepath.Join(dir, "niimbot", "templates.json")
	data, err := os.ReadFile(tplFile)
	if err != nil {
		return nil, fmt.Errorf("template %q not found as file or in %s", nameOrPath, tplFile)
	}
	var list []SavedTemplate
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	for _, t := range list {
		if strings.EqualFold(strings.TrimSpace(t.Name), strings.TrimSpace(nameOrPath)) {
			cp := t
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("template %q not found in templates.json", nameOrPath)
}

func splitRowColumns(s string) []string {
	sep := ";"
	if strings.Contains(s, "\t") {
		sep = "\t"
	}
	parts := strings.Split(s, sep)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func renderSingleTemplate(nameOrPath, dataRow string, cfg AppConfig) (*image.Gray, string, error) {
	tpl, err := findTemplateByNameOrPath(nameOrPath)
	if err != nil {
		return nil, "", err
	}
	length := tpl.Length
	if length <= 0 {
		length = cfg.Length
	}
	cols := splitRowColumns(dataRow)
	startCnt := 1
	if tpl.CounterStart != nil {
		startCnt = *tpl.CounterStart
	}

	if len(tpl.Elements) > 0 {
		img, err := RenderTemplateCtx(LabelTemplate{LengthMM: length, Elements: tpl.Elements}, cols, startCnt, 1)
		if err != nil {
			return nil, "", err
		}
		off := cfg.OffsetPx
		if tpl.OffsetPx != nil {
			off = *tpl.OffsetPx
		}
		if off != 0 {
			shiftImageX(img, off)
		}
		return img, tpl.Label, nil
	}

	rawTpl := tpl.Template
	if sub, sErr := substituteFieldsCtx(rawTpl, cols, time.Now(), startCnt, 1); sErr == nil {
		rawTpl = sub
	} else {
		return nil, "", sErr
	}
	lines := strings.Split(rawTpl, "\n")
	for i, v := range cols {
		for l := range lines {
			lines[l] = strings.ReplaceAll(lines[l], fmt.Sprintf("{%d}", i+1), v)
		}
	}
	opts := buildTextOptionsFromConfig()
	if tpl.FontName != "" {
		opts.FontName = tpl.FontName
	}
	if tpl.FontSize != "" {
		opts.FontSize = tpl.FontSize
	}
	opts.Bold = tpl.Bold
	img, err := RenderTextWithOptions(lines, length, opts)
	return img, tpl.Label, err
}

func buildPreviewAny(subCmd string, args []string, top, bot string, cfg AppConfig) (*image.Gray, error) {
	switch strings.ToLower(subCmd) {
	case "barcode":
		if len(args) == 0 {
			return nil, fmt.Errorf("missing barcode data")
		}
		return RenderBarcodeLabel(BarcodeOptions{
			Data: args[0], Format: cfg.BcFormat, LengthMM: cfg.Length,
			TopText: top, BottomText: bot, FontName: cfg.FontName,
			IsMainTextBold: cfg.IsMainTextBold, FontSize: cfg.FontSize,
			OffsetPx: cfg.OffsetPx, SideMarginPx: cfg.SideMarginPx,
			MarginPx: cfg.MarginPx, GapPx: cfg.GapPx, TextGapPx: cfg.TextGapPx,
			SafeMode: cfg.BcSafeMode,
		}, cfg)
	case "text":
		if len(args) == 0 {
			return nil, fmt.Errorf("missing text lines")
		}
		return RenderTextWithOptions(args, cfg.Length, buildTextOptionsFromConfig())
	case "image":
		if len(args) == 0 {
			return nil, fmt.Errorf("missing image file path")
		}
		return loadAndFitImage(args[0], cfg)
	case "template":
		if len(args) == 0 {
			return nil, fmt.Errorf("missing template name or .niimbot.json path")
		}
		row := ""
		if len(args) > 1 {
			row = strings.Join(args[1:], " ")
		}
		img, _, err := renderSingleTemplate(args[0], row, cfg)
		return img, err
	default:
		return nil, fmt.Errorf("unknown preview target: %s", subCmd)
	}
}

func saveGrayPng(path string, img *image.Gray) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return png.Encode(f, img)
}

func fatal(err error) {
	LogErr("%v", err)
	os.Exit(1)
}

func printUsage() {
	fmt.Println(`niimbc-cli — быстрая консольная печать для NIIMBOT N1

Команды:
  scan                                  Найти принтер и сохранить адрес в config.json
  info                                  Принудительно опросить модель, заряд и RFID расходников
  barcode "DATA"                        Print a single barcode or 2D code
  text "Line 1" ["Line 2"]              Print a text label
  image файл.png                        Напечатать картинку (PNG/JPEG/GIF)
  template "Name" ["col1;col2"]         Print a single label using a saved or .niimbot.json template
  preview barcode|text|image|template   Сохранить PNG-макет без подключения к принтеру (--out)

Флаги (указываются перед командой):
  --length 50        Длина этикетки в мм (по умолчанию из config.json)
  --format auto      Формат: auto, code128, code39, ean13, ean8, pdf417, qr, qrl, qrr, datamatrix, dml, dmr
  --top "Text"       Top caption for barcode
  --bot "Text"       Bottom caption for barcode
  --safe             Безопасный режим штрихкода
  --label withgaps   Тип labels (по умолчанию withgaps; auto — по RFID)
  --copies 1         Количество копий
  --out label.png    Путь сохранения PNG для preview
  --quiet            Тихий режим (выводить только ошибки)
  --save-config      Сохранить переданные флаги в config.json`)
}
