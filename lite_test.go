package main

import (
"testing"
)

func TestResolveFastLabelPriority(t *testing.T) {
cfgDefault := DefaultConfig()
got, err := resolveFastLabel(nil, "", cfgDefault)
if err != nil || got != "withgaps" {
t.Fatalf("ожидали withgaps по умолчанию, получили %q (err=%v)", got, err)
}

cfgEmpty := AppConfig{Label: ""}
gotTpl, err := resolveFastLabel(nil, "transparent", cfgEmpty)
if err != nil || gotTpl != "transparent" {
t.Fatalf("ожидали transparent из шаблона, получили %q (err=%v)", gotTpl, err)
}

cfgFlag := AppConfig{Label: "continuous"}
gotFlag, err := resolveFastLabel(nil, "transparent", cfgFlag)
if err != nil || gotFlag != "continuous" {
t.Fatalf("ожидали continuous из флага/конфига, получили %q (err=%v)", gotFlag, err)
}
}

func TestLiteBarcodeAndTemplateRender(t *testing.T) {
cfg := DefaultConfig()
for _, tc := range []struct{ f, d string }{
{"auto", "123498756000099999"},
{"ean13", "460700465001"},
{"code39", "INV-2026"},
{"pdf417", "LITE-TEST"},
{"dml", "DM-LITE"},
} {
img, err := RenderBarcodeLabel(BarcodeOptions{
Data: tc.d, Format: tc.f, LengthMM: 50, TopText: "Lite", SafeMode: true,
}, cfg)
if err != nil || img.Bounds().Dx() != 400 || img.Bounds().Dy() != 96 {
t.Fatalf("ошибка рендера %s: %v", tc.f, err)
}
}

tpl := LabelTemplate{
LengthMM: 50,
Elements: []Element{
{Kind: "text", X: 1, Y: 1, W: 48, H: 4, Text: "Позиция {1}", FontSize: "-1"},
{Kind: "barcode", X: 1, Y: 5, W: 48, H: 6.5, Text: "{2}", BcFormat: "code128", BcSafeMode: func() *bool { b := true; return &b }()},
},
}
imgTpl, err := RenderTemplateCtx(tpl, []string{"Склад 1", "INV-01"}, 1, 1)
if err != nil || imgTpl.Bounds().Dx() != 400 {
t.Fatalf("ошибка рендера шаблона: %v", err)
}
}

func TestEnsureUTF8FromCP1251(t *testing.T) {
// Байты слова "Насос" в кодировке Windows-1251: CD E0 F1 EE F1
win1251Bytes := string([]byte{0xCD, 0xE0, 0xF1, 0xEE, 0xF1})
if got := ensureUTF8(win1251Bytes, false); got != "Насос" {
t.Fatalf("ожидали 'Насос', получили %q", got)
}
if got := ensureUTF8("Склад №1", false); got != "Склад №1" {
t.Fatalf("UTF-8 строка не должна искажаться, получили %q", got)
}
}
