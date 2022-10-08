package job

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"x-ui/database/model"
	"x-ui/logger"
	"x-ui/web/service"

	"github.com/google/uuid"
)

type CheckBusinessStatusJob struct {
	inboundService  service.InboundService
	businessService service.BusinessService
	xrayService     service.XrayService
}

func NewCheckBusinessStatusJob() *CheckBusinessStatusJob {
	return new(CheckBusinessStatusJob)
}

func (j *CheckBusinessStatusJob) Run() {
	data, err := j.businessService.GetBusinessInfo()
	if err != nil {
		logger.Error("获取业务信息错误：", err)
		return
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
		}

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
			inbound.Protocol = model.VLESS
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
				return
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
			inbound.Down = 0
			inbound.Up = 0
			inbound.Total = int64(total * 1024 * 1024 * 1024)
			err := j.inboundService.UpdateInbound(inbound)
			if err != nil {
				logger.Error("更新入站出错：", err)
				return
			}

			if err == nil {
				j.xrayService.SetToNeedRestart()
			}

			logger.Info("更新入站成功")

		}
	}

}
