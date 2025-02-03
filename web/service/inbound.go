package service

import (
	"fmt"
	"time"
	"x-ui/database"
	"encoding/json"
	"x-ui/database/model"
	"x-ui/util/common"
	"x-ui/xray"
	"x-ui/logger"

	"gorm.io/gorm"
)

type InboundService struct {
}

func (s *InboundService) GetInbounds(userId int) ([]*model.Inbound, error) {
	db := database.GetDB()
	var inbounds []*model.Inbound
	err := db.Model(model.Inbound{}).Preload("ClientStats").Where("user_id = ?", userId).Find(&inbounds).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return inbounds, nil
}

func (s *InboundService) GetAllInbounds() ([]*model.Inbound, error) {
	db := database.GetDB()
	var inbounds []*model.Inbound
	err := db.Model(model.Inbound{}).Preload("ClientStats").Find(&inbounds).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return inbounds, nil
}

func (s *InboundService) checkPortExist(port int, ignoreId int) (bool, error) {
	db := database.GetDB()
	db = db.Model(model.Inbound{}).Where("port = ?", port)
	if ignoreId > 0 {
		db = db.Where("id != ?", ignoreId)
	}
	var count int64
	err := db.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *InboundService) AddInbound(inbound *model.Inbound) (*model.Inbound,error) {
	exist, err := s.checkPortExist(inbound.Port, 0)
	if err != nil {
		return inbound, err
	}
	if exist {
		return inbound, common.NewError("port already exists:", inbound.Port)
	}
	db := database.GetDB()

	err = db.Save(inbound).Error
	if err == nil {
		s.UpdateClientStat(inbound.Id,inbound.Settings)
	}
	return inbound, err
}

func (s *InboundService) AddInbounds(inbounds []*model.Inbound) error {
	for _, inbound := range inbounds {
		exist, err := s.checkPortExist(inbound.Port, 0)
		if err != nil {
			return err
		}
		if exist {
			return common.NewError("port already exists:", inbound.Port)
		}
	}

	db := database.GetDB()
	tx := db.Begin()
	var err error
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	for _, inbound := range inbounds {
		err = tx.Save(inbound).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *InboundService) DelInbound(id int) error {
	db := database.GetDB()
	return db.Delete(model.Inbound{}, id).Error
}
func (s *InboundService) DelInboundByPort(port int) error {
	db := database.GetDB()
	var inbound model.Inbound
	db.First(&inbound, "port = ?", port)
	return db.Delete(&inbound).Error
}
func (s *InboundService) GetInbound(id int) (*model.Inbound, error) {
	db := database.GetDB()
	inbound := &model.Inbound{}
	err := db.Model(model.Inbound{}).First(inbound, id).Error
	if err != nil {
		return nil, err
	}
	return inbound, nil
}

func (s *InboundService) UpdateInbound(inbound *model.Inbound) (*model.Inbound, error) {
	exist, err := s.checkPortExist(inbound.Port, inbound.Id)
	if err != nil {
		return inbound, err
	}
	if exist {
		return inbound, common.NewError("port already exists:", inbound.Port)
	}

	oldInbound, err := s.GetInbound(inbound.Id)
	if err != nil {
		return inbound, err
	}
	oldInbound.Up = inbound.Up
	oldInbound.Down = inbound.Down
	oldInbound.Total = inbound.Total
	oldInbound.Remark = inbound.Remark
	oldInbound.Enable = inbound.Enable
	oldInbound.ExpiryTime = inbound.ExpiryTime
	oldInbound.Listen = inbound.Listen
	oldInbound.Port = inbound.Port
	oldInbound.Protocol = inbound.Protocol
	oldInbound.Settings = inbound.Settings
	oldInbound.StreamSettings = inbound.StreamSettings
	oldInbound.Sniffing = inbound.Sniffing
	oldInbound.Tag = fmt.Sprintf("inbound-%v", inbound.Port)

	s.UpdateClientStat(inbound.Id,inbound.Settings)
	db := database.GetDB()
	return inbound, db.Save(oldInbound).Error
}

func (s *InboundService) AddTraffic(traffics []*xray.Traffic) (err error) {
	if len(traffics) == 0 {
		return nil
	}
	db := database.GetDB()
	db = db.Model(model.Inbound{})
	tx := db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()
	for _, traffic := range traffics {
		if traffic.IsInbound {
			err = tx.Where("tag = ?", traffic.Tag).
				UpdateColumn("up", gorm.Expr("up + ?", traffic.Up)).
				UpdateColumn("down", gorm.Expr("down + ?", traffic.Down)).
				Error
			if err != nil {
				return
			}
		}
	}
	return
}
func (s *InboundService) AddClientTraffic(traffics []*xray.ClientTraffic) (err error) {
	if len(traffics) == 0 {
		return nil
	}
	db := database.GetDB()
	db = db.Model(xray.ClientTraffic{})
	tx := db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()
	for _, traffic := range traffics {
		inbound := &model.Inbound{}

		err := db.Model(model.Inbound{}).Where("settings like ?", "%" + traffic.Email + "%").First(inbound).Error
		traffic.InboundId = inbound.Id
		if err != nil {
			logger.Warning("AddClientTraffic find model ", err, traffic.Email)
			continue
		}
		// get settings clients
		settings := map[string][]model.Client{}
		json.Unmarshal([]byte(inbound.Settings), &settings)
		clients := settings["clients"]
		for _, client := range clients {
			if traffic.Email == client.Email {
				traffic.ExpiryTime = client.ExpiryTime
				traffic.Total = client.TotalGB
			}
		}
		if tx.Where("inbound_id = ?", inbound.Id).Where("email = ?", traffic.Email).
		UpdateColumn("enable", true).
		UpdateColumn("expiry_time", traffic.ExpiryTime).
		UpdateColumn("total",traffic.Total).
		UpdateColumn("up", gorm.Expr("up + ?", traffic.Up)).
		UpdateColumn("down", gorm.Expr("down + ?", traffic.Down)).RowsAffected == 0 {
			err = tx.Create(traffic).Error
		}
		
		if err != nil {
			logger.Warning("AddClientTraffic update data ", err)
			continue
		}
	
	}
	return
}

func (s *InboundService) DisableInvalidInbounds() (int64, error) {
	db := database.GetDB()
	now := time.Now().Unix() * 1000
	result := db.Model(model.Inbound{}).
		Where("((total > 0 and up + down >= total) or (expiry_time > 0 and expiry_time <= ?)) and enable = ?", now, true).
		Update("enable", false)
	err := result.Error
	count := result.RowsAffected
	return count, err
}
func (s *InboundService) DisableInvalidClients() (int64, error) {
	db := database.GetDB()
	now := time.Now().Unix() * 1000
	result := db.Model(xray.ClientTraffic{}).
		Where("((total > 0 and up + down >= total) or (expiry_time > 0 and expiry_time <= ?)) and enable = ?", now, true).
		Update("enable", false)
	err := result.Error
	count := result.RowsAffected
	return count, err
}
func (s *InboundService) UpdateClientStat(inboundId int, inboundSettings string) (error) {
	db := database.GetDB()

	// get settings clients
	settings := map[string][]model.Client{}
	json.Unmarshal([]byte(inboundSettings), &settings)
	clients := settings["clients"]
	for _, client := range clients {
		result := db.Model(xray.ClientTraffic{}).
		Where("inbound_id = ? and email = ?", inboundId, client.Email).
		Updates(map[string]interface{}{"enable": true, "total": client.TotalGB, "expiry_time": client.ExpiryTime})
		if result.RowsAffected == 0 {
			clientTraffic := xray.ClientTraffic{}
			clientTraffic.InboundId = inboundId
			clientTraffic.Email = client.Email
			clientTraffic.Total = client.TotalGB
			clientTraffic.ExpiryTime = client.ExpiryTime
			clientTraffic.Enable = true
			clientTraffic.Up = 0
			clientTraffic.Down = 0
			db.Create(&clientTraffic)
		}
		err := result.Error
		if err != nil {
			return err
		}
	
	}
	return nil
}

func (s *InboundService) GetInboundClientIps(clientEmail string) (string, error) {
	db := database.GetDB()
	InboundClientIps := &model.InboundClientIps{}
	err := db.Model(model.InboundClientIps{}).Where("client_email = ?", clientEmail).First(InboundClientIps).Error
	if err != nil {
		return "", err
	}
	return InboundClientIps.Ips, nil
}
func (s *InboundService) ClearClientIps(clientEmail string) (error) {
	db := database.GetDB()

	result := db.Model(model.InboundClientIps{}).
		Where("client_email = ?", clientEmail).
		Update("ips", "")
	err := result.Error


	if err != nil {
		return err
	}
	return nil
}
func (s *InboundService) ClearTrafficByPort(port int) error {
	db := database.GetDB()
	Uperr := db.Model(model.Inbound{}).Where("port = ?", port).Update("up", 0).Error
	if Uperr != nil {
		fmt.Println("ClearTrafficByPort error:clear up failed")
		return Uperr
	}
	Downerr := db.Model(model.Inbound{}).Where("port = ?", port).Update("down", 0).Error
	if Downerr != nil {
		fmt.Println("ClearTrafficByPort error:clear down failed")
		return Downerr
	}
	return nil
}

func (s *InboundService) ClearAllInboundTraffic() error {
	inbounds, _ := s.GetAllInbounds()
	for _, inbound := range inbounds {
		err := s.ClearTrafficByPort(inbound.Port)
		if err != nil {
			fmt.Printf("ClearAllInboundTraffic error,ClearTrafficByPort port %d fail", inbound.Port)
			continue
		}
	}
	return nil
}
func (s *InboundService) DisableInboundByPort(port int) error {
	db := database.GetDB()
	return db.Model(model.Inbound{}).Where("port = ?", port).Update("enable", false).Error
}
func (s *InboundService) EnableInboundByPort(port int) error {
	db := database.GetDB()
	return db.Model(model.Inbound{}).Where("port = ?", port).Update("enable", true).Error
}
