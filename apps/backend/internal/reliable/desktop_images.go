package reliable

func desktopImage(path string) []byte {
	switch path {
	case "/api/v1/reliable/brand/codex.png":
		return codexBrand
	case "/api/v1/reliable/brand/deepseek.png":
		return deepseekBrand
	case "/api/v1/reliable/brand/zcode.png":
		return zcodeBrand
	case "/reliable/icon.png":
		return puddingAvatar
	case "/reliable/background.png":
		return teahouseBackground
	default:
		return nil
	}
}
