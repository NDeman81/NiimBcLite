package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// AppConfig хранит настройки принтера, этикетки, штрихкодов и режимов работы по умолчанию.
type AppConfig struct {
	FastBatchSize        int `json:"fastBatchSize"`
	FastBatchDelayMs     int `json:"fastBatchDelayMs"`
	StatusPollTimeoutMs  int `json:"statusPollTimeoutMs"`
	KeepAliveSec         int    `json:"keepAliveSec"`
	RfidLowAction        string `json:"rfidLowAction"`
	BcSafeMode bool `json:"bcSafeMode"`
	// Параметры принтера и печати
	Address   string  `json:"address"`   // MAC-адрес или UUID принтера (пусто = автопоиск N1-)
	Length    float64 `json:"length"`    // длина этикетки вдоль подачи, мм (по умолчанию 50)
	Height    float64 `json:"height"`    // ширина/высота ленты поперёк подачи, мм (по умолчанию 14)
	Density   int     `json:"density"`   // плотность прожига головки 1..3 (по умолчанию 2)
	Label     string  `json:"label"`     // тип этикетки (пусто = автоопределение по RFID)
	Flip      bool    `json:"flip"`      // переворот изображения на 180°
	Threshold int     `json:"threshold"` // порог чёрно-белого растра 1..255 (по умолчанию 200)
	Copies    int     `json:"copies"`    // количество копий (по умолчанию 1)

	// Режим сервера/очереди и каталоги
	KeepAlive bool   `json:"keepAlive"`
	LogLevel  string `json:"logLevel"`
	PrintMode string `json:"printMode"` // fast | safe (по умолчанию fast) // error | info | debug (по умолчанию info) // true = Keep-Alive (удерживать связь), false = On-Demand
	SaveDir   string `json:"saveDir"`
	SaveFile  string `json:"saveFile"`
	AutoFileName bool `json:"autoFileName"`
	ImgStretch   bool `json:"imgStretch"`   // true = растягивать картинку до полей без сохранения пропорций // true = формировать имя PNG из данных штрихкода/текста  // имя PNG-файла превью (по умолчанию niimbot_label_preview.png)   // папка для PNG-превью (пусто = папка программы)
	WatchDir    string `json:"watchDir"`    // папка мониторинга очереди (пусто = NiimbotQueue рядом с exe)
SaveHistory bool   `json:"saveHistory"` // сохранять журнал печати в history.json (по умолчанию true)

	// Параметры штрихкодов и QR-кодов
	BcFormat       string `json:"bcFormat"`       // auto | code128 | qr | qrl | qrr
	DefaultQrSide  string `json:"defaultQrSide"`  // left | right (используется при bcFormat == "qr")
	FontName       string `json:"fontName"`       // имя системного шрифта
	IsMainTextBold bool   `json:"isMainTextBold"` // полужирный шрифт основной подписи
	FontSize       string `json:"fontSize"`       // -1 (авто), 0 (скрыть), N (pt), Np (%)
	OffsetPx       int    `json:"offsetPx"`       // горизонтальный сдвиг в точках
	SideMarginPx   int    `json:"sideMarginPx"`   // боковые поля / тихие зоны в точках
	MarginPx       int    `json:"marginPx"`       // верхнее и нижнее поля в точках
	GapPx          int    `json:"gapPx"`          // зазор между кодом и подписью в точках
	TextGapPx      int    `json:"textGapPx"`      // зазор до верхнего и нижнего текста в точках
}

// DefaultConfig возвращает заводские настройки по умолчанию.
func DefaultConfig() AppConfig {
	return AppConfig{
		Address:        "",
		Length:         50.0,
		Height:         14.0,
		Density:        2,
		Label:          "withgaps",
		Flip:           false,
		Threshold:      200,
		Copies:         1,
		KeepAlive:           true,
		FastBatchSize:       6,
		FastBatchDelayMs:    6,
		StatusPollTimeoutMs: 250,
		KeepAliveSec:        8,
		RfidLowAction:       "ask",
		LogLevel:       "info",
		PrintMode:      "fast",
		SaveDir:        "",
		SaveFile:       "niimbot_label_preview.png",
		WatchDir:       "",
		SaveHistory:    true,
		BcFormat:       "auto",
		DefaultQrSide:  "left",
		FontName:       "",
		IsMainTextBold: false,
		FontSize:       "-1",
		OffsetPx:       0,
		SideMarginPx:   10,
		MarginPx:       4,
		GapPx:          4,
		TextGapPx:      4,
	}
}

// executableDir возвращает каталог исполняемого файла (или пустую строку при запуске через go run).
func executableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	if strings.Contains(strings.ToLower(dir), "go-build") {
		return ""
	}
	return dir
}

var customConfigPath string

// SetConfigPath задаёт альтернативный путь к файлу конфигурации (из флага --config).
func SetConfigPath(path string) {
customConfigPath = strings.TrimSpace(path)
}

// configPath определяет путь к файлу конфигурации.
func configPath() string {
if customConfigPath != "" {
return customConfigPath
}
	if exeDir := executableDir(); exeDir != "" {
		return filepath.Join(exeDir, "config.json")
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "niimbot", "config.json")
}

// normalizeConfig проверяет границы значений и заполняет пустые поля значениями по умолчанию.
func normalizeConfig(cfg *AppConfig) {
	def := DefaultConfig()
	if cfg.Length <= 0 {
		cfg.Length = def.Length
	}
	if cfg.Height <= 0 {
		cfg.Height = def.Height
	}
	if cfg.Density < 1 || cfg.Density > 3 {
		cfg.Density = def.Density
	}
	if cfg.Threshold < 1 || cfg.Threshold > 255 {
		cfg.Threshold = def.Threshold
	}
	if cfg.Copies < 1 {
		cfg.Copies = def.Copies
	}
	if cfg.BcFormat == "" {
		cfg.BcFormat = def.BcFormat
	}
	if cfg.DefaultQrSide != "right" {
		cfg.DefaultQrSide = "left"
	}
	if cfg.FontSize == "" {
		cfg.FontSize = def.FontSize
	}
	if strings.TrimSpace(cfg.SaveFile) == "" {
		cfg.SaveFile = def.SaveFile
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = def.LogLevel
	}
	if cfg.FastBatchSize <= 0 { cfg.FastBatchSize = 6 }
	if cfg.FastBatchDelayMs <= 0 { cfg.FastBatchDelayMs = 6 }
	if cfg.StatusPollTimeoutMs <= 0 { cfg.StatusPollTimeoutMs = 250 }
	if cfg.KeepAliveSec < 0 { cfg.KeepAliveSec = 8 }
	if cfg.RfidLowAction != "block" && cfg.RfidLowAction != "warn" { cfg.RfidLowAction = "ask" }
	if cfg.PrintMode != "safe" {
		cfg.PrintMode = "fast"
	}
}

// LoadConfig читает настройки из config.json. Если файл отсутствует — создаёт его автоматически.
func LoadConfig() AppConfig {
	cfg := DefaultConfig()
	path := configPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			_ = SaveConfig(cfg)
		}
		return cfg
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig()
	}
	normalizeConfig(&cfg)
	return cfg
}

// SaveConfig сохраняет конфигурацию в config.json в форматированном виде.
func SaveConfig(cfg AppConfig) error {
	normalizeConfig(&cfg)
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
