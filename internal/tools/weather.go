package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func weatherQuery(params map[string]any) (any, error) {
	city := ""
	if c, ok := params["city"].(string); ok && c != "" {
		city = c
	} else {
		city = getIPLocation()
	}

	url := fmt.Sprintf("https://wttr.in/%s?format=j1", city)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("weather API failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if !strings.Contains(string(body), "weather") {
		return "无法获取天气信息，请检查城市名称", nil
	}

	var data struct {
		CurrentCondition []struct {
			TempC      string `json:"temp_C"`
			WeatherCode string `json:"weatherCode"`
			Humidity   string `json:"humidity"`
			WindspeedKmph string `json:"windspeedKmph"`
			Winddir   string `json:"winddir"`
			FeelsLikeC string `json:"FeelsLikeC"`
		} `json:"current_condition"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return string(body), nil
	}

	if len(data.CurrentCondition) == 0 {
		return "无法获取天气信息", nil
	}

	cur := data.CurrentCondition[0]
	result := fmt.Sprintf("📍 %s 天气\n🌡️ 温度: %s°C (体感 %s°C)\n🌤️ 天气: %s\n💧 湿度: %s%%\n💨 风速: %s km/h %s",
		city, cur.TempC, cur.FeelsLikeC, cur.WeatherCode, cur.Humidity, cur.WindspeedKmph, cur.Winddir)

	return result, nil
}

func getIPLocation() string {
	resp, err := http.Get("https://ipapi.co/json/")
	if err != nil {
		return "Beijing"
	}
	defer resp.Body.Close()

	var data struct {
		City string `json:"city"`
	}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &data)
	if data.City != "" {
		return data.City
	}
	return "Beijing"
}