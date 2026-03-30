package aihelper

import appconfig "GopherAI/config"

func BuildModelConfig(modelType string) map[string]interface{} {
	conf := appconfig.GetConfig()

	switch modelType {
	case "2":
		return map[string]interface{}{
			"baseURL":   conf.OllamaConfig.BaseURL,
			"modelName": conf.OllamaConfig.ModelName,
		}
	default:
		return map[string]interface{}{}
	}
}
