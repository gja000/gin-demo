package service

import (
	"crypto/rand"
	"encoding/hex"
	"sync/atomic"
	"time"

	"gin-demo/internal/model"
	"gin-demo/pkg/db"
)

var idCounter uint64

// genID 生成带前缀的唯一 ID（演示用，足够唯一）
func genID(prefix string) string {
	n := atomic.AddUint64(&idCounter, 1)
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b) + "-" + itoa64(n)
}

func itoa64(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// VehicleState 逐车核查状态
type VehicleState struct {
	VehicleID         string  `json:"vehicleId"`
	Name              string  `json:"name"`
	PrevInterval      int     `json:"prevInterval"`
	ExpectedInterval  int     `json:"expectedInterval"`
	EffectiveInterval int     `json:"effectiveInterval"`
	Batch             string  `json:"batch"`
	VersionNo         int     `json:"versionNo"`
	Status            string  `json:"status"` // PUBLISHED/FETCHED/SUCCESS/FAILED/TIMEOUT/NOT_SENT
	FailReason        string  `json:"failReason,omitempty"`
	PublishedAt       *time.Time `json:"publishedAt,omitempty"`
	FetchedAt         *time.Time `json:"fetchedAt,omitempty"`
	ReceiptAt         *time.Time `json:"receiptAt,omitempty"`
	TimedOut          bool    `json:"timedOut"`
}

// GetVehicleStates 计算活动下所有车辆的核查状态（来源：config_version 最新版本 + 车辆原值）
func GetVehicleStates(activityID, fleetID string) ([]VehicleState, error) {
	// 车队全部车辆
	var vehicles []model.Vehicle
	if err := db.DB.Where("fleet_id = ?", fleetID).Find(&vehicles).Error; err != nil {
		return nil, err
	}
	// 该活动全部版本
	var versions []vehicleVersionLite
	db.DB.Table("config_versions").Where("activity_id = ?", activityID).Find(&versions)

	byVeh := map[string][]vehicleVersionLite{}
	for _, v := range versions {
		byVeh[v.VehicleID] = append(byVeh[v.VehicleID], v)
	}

	states := make([]VehicleState, 0, len(vehicles))
	for _, veh := range vehicles {
		vs := VehicleState{
			VehicleID:         veh.ID,
			Name:              veh.Name,
			PrevInterval:      veh.PrevInterval,
			EffectiveInterval: veh.EffectiveInterval,
			Status:            "NOT_SENT",
		}
		vsList := byVeh[veh.ID]
		if len(vsList) > 0 {
			latest := vsList[0]
			for _, v := range vsList[1:] {
				if v.VersionNo > latest.VersionNo {
					latest = v
				}
			}
			vs.ExpectedInterval = latest.ExpectedInterval
			vs.Batch = latest.Batch
			vs.VersionNo = latest.VersionNo
			vs.Status = latest.Status
			vs.FailReason = latest.FailReason
			vs.PublishedAt = latest.PublishedAt
			vs.FetchedAt = latest.FetchedAt
			vs.ReceiptAt = latest.ReceiptAt
			vs.TimedOut = latest.TimedOut
		}
		states = append(states, vs)
	}
	return states, nil
}

type vehicleVersionLite struct {
	VehicleID        string
	VersionNo        int
	ExpectedInterval int
	Batch            string
	Status           string
	FailReason       string
	PublishedAt      *time.Time
	FetchedAt        *time.Time
	ReceiptAt        *time.Time
	TimedOut         bool
}

// overview 概览统计
type Overview struct {
	Total       int `json:"total"`
	Success     int `json:"success"`
	Failed      int `json:"failed"`
	Timeout     int `json:"timeout"`
	Unconfirmed int `json:"unconfirmed"` // PUBLISHED/FETCHED
	NotSent     int `json:"notSent"`
}

func ComputeOverview(states []VehicleState) Overview {
	o := Overview{Total: len(states)}
	for _, s := range states {
		switch s.Status {
		case "SUCCESS":
			o.Success++
		case "FAILED":
			o.Failed++
		case "TIMEOUT":
			o.Timeout++
		case "NOT_SENT":
			o.NotSent++
		default: // PUBLISHED/FETCHED
			o.Unconfirmed++
		}
	}
	return o
}

// UIDFromString 将字符串 uid 转 uint（服务端以会话令牌为准）
func UIDFromString(s string) uint {
	return uidToUint(s)
}
