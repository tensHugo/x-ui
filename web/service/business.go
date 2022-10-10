package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
	"x-ui/database/model"
	"x-ui/logger"

	"github.com/google/uuid"
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
	xrayService    XrayService
	inboundService InboundService
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

func (j *BusinessService) EenewBusinessInfo() error {
	data, err := j.GetBusinessInfo()
	if err != nil {
		logger.Error("获取业务信息错误：", err)
		return err
	}

	//将到期时间转换成时间戳
	timeLayout := "2006-01-02 15:04:05"
	loc, _ := time.LoadLocation("Local")
	theTime, _ := time.ParseInLocation(timeLayout, data.Data.EndTime, loc)
	sr := theTime.UnixNano() / 1e6

	if data.Result {
		inbo, err := j.inboundService.GetAllInbounds()
		if err != nil {
			logger.Error("获取入站信息失败：", err)
			return err
		}

		//自动设置流量重置日
		startTime, _ := time.ParseInLocation(timeLayout, data.Data.BuyTime, loc)
		j.settingService.SetTrafficResetDay(startTime.Day())

		strArr := strings.Split(data.Data.Flow, "G")
		guid := uuid.New()
		//默认参数
		settings := "{\n  \"clients\": [\n    {\n      \"id\": \"id-key\",\n      \"flow\": \"xtls-rprx-direct\"\n    }\n  ],\n  \"decryption\": \"none\",\n  \"fallbacks\": []\n}"
		streamSettings := "{\n  \"network\": \"ws\",\n  \"security\": \"none\",\n  \"wsSettings\": {\n    \"acceptProxyProtocol\": false,\n    \"path\": \"/jiulingyun\",\n    \"headers\": {}\n  }\n}"
		sniffing := "{\n  \"enabled\": true,\n  \"destOverride\": [\n    \"http\",\n    \"tls\"\n  ]\n}"
		settings = strings.Replace(settings, "id-key", guid.String(), 1)
		total, _ := strconv.Atoi(strArr[0])

		//检查入站是否添加，如果未添加则新增一个入站
		if len(inbo) < 1 {
			inbound := &model.Inbound{}
			inbound.UserId = 1
			inbound.Enable = true
			inbound.Down = 0
			inbound.ExpiryTime = sr
			inbound.Id = 1
			inbound.Port = 80
			inbound.Protocol = model.VMess
			inbound.Remark = data.Data.Area + "-" + data.Data.UserId + "-" + data.Data.Id
			inbound.Settings = settings
			inbound.Sniffing = sniffing
			inbound.StreamSettings = streamSettings
			inbound.Total = int64(total * 1024 * 1024 * 1024)
			inbound.Up = 0
			inbound.Tag = fmt.Sprintf("inbound-%v", inbound.Port)

			err := j.inboundService.AddInbound(inbound)
			if err != nil {
				logger.Error("添加入站失败：", err)
				return err
			}

			if err == nil {
				j.xrayService.SetToNeedRestart()
			}

			logger.Info("添加入站成功")

		}

		//检查平台用户是否续费，如果续费自动启用
		inbound, err := j.inboundService.GetInbound(1)
		if sr > inbound.ExpiryTime && data.Data.Status == "2" {
			inbound.ExpiryTime = sr
			inbound.Enable = true
			inbound.Total = int64(total * 1024 * 1024 * 1024)
			err := j.inboundService.UpdateInbound(inbound)
			if err != nil {
				logger.Error("更新入站出错：", err)
				return err
			}

			if err == nil {
				j.xrayService.SetToNeedRestart()
			}

			logger.Info("更新入站成功")

		}
	}

	return nil
}
