package service

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"x-ui/logger"
)

type Result struct {
	Result  bool   `json:"result"`
	Message string `json:"msg"`
	Data    struct {
		Id      string `json:"id"`
		Guid    string `json:"guid"`
		UserId  string `json:"user_id"`
		OrderId string `json:"order_id"`
		Name    string `json:"name"`
		BuyTime string `json:"buy_time"`
		EndTime string `json:"end_time"`
		Status  string `json:"status"`
		Area    string `json:"area"`
		Flow    string `json:"flow"`
	}
}

type BusinessService struct {
	settingService SettingService
}

func (j *BusinessService) GetBusinessInfo() (*Result, error) {
	data := &Result{}
	allSetting, err := j.settingService.GetAllSetting()
	if err != nil {
		logger.Warning("获取全部配置信息失败", err)
	}
	if allSetting.ApiUrl != "" && allSetting.ApiKey != "" && allSetting.BusinessId > 0 {
		url := allSetting.ApiUrl + "/index/getInfo?key=" + allSetting.ApiKey + "&id=" + strconv.Itoa(allSetting.BusinessId)
		resp, err := http.Get(url)
		if err != nil {
			logger.Warning("http请求错误", err)
			return nil, err
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Warning("http的body解析错误", err)
			return nil, err
		}

		err = json.Unmarshal(body, &data)
		if err != nil {
			logger.Warning("json解析失败:", err)
			return nil, err
		}

	}
	return data, err
}
