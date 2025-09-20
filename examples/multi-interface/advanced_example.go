// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// 高级多网卡连接示例
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// DeviceConnection 表示一个设备连接配置
type DeviceConnection struct {
	Name      string // 设备名称
	Endpoint  string // OPC UA端点
	LocalAddr string // 本地网卡地址
	Client    *opcua.Client
	Connected bool
	LastError error
	mu        sync.RWMutex
}

// MultiInterfaceManager 管理多个网卡的连接
type MultiInterfaceManager struct {
	devices []*DeviceConnection
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func NewMultiInterfaceManager() *MultiInterfaceManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &MultiInterfaceManager{
		devices: make([]*DeviceConnection, 0),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// AddDevice 添加一个设备连接配置
func (m *MultiInterfaceManager) AddDevice(name, endpoint, localAddr string) {
	device := &DeviceConnection{
		Name:      name,
		Endpoint:  endpoint,
		LocalAddr: localAddr,
	}
	m.devices = append(m.devices, device)
}

// ConnectAll 连接所有设备
func (m *MultiInterfaceManager) ConnectAll() error {
	for _, device := range m.devices {
		if err := m.connectDevice(device); err != nil {
			log.Printf("连接设备 %s 失败: %v", device.Name, err)
			device.setError(err)
		}
	}
	return nil
}

// connectDevice 连接单个设备
func (m *MultiInterfaceManager) connectDevice(device *DeviceConnection) error {
	opts := []opcua.Option{
		opcua.SecurityPolicy(ua.SecurityPolicyURINone),
		opcua.SecurityModeString("None"),
		opcua.LocalAddr(device.LocalAddr), // 指定网卡
		opcua.AutoReconnect(true),
		opcua.ReconnectInterval(5 * time.Second),
		opcua.DialTimeout(10 * time.Second),
	}

	client, err := opcua.NewClient(device.Endpoint, opts...)
	if err != nil {
		return fmt.Errorf("创建客户端失败: %v", err)
	}

	if err := client.Connect(m.ctx); err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}

	device.mu.Lock()
	device.Client = client
	device.Connected = true
	device.LastError = nil
	device.mu.Unlock()

	fmt.Printf("✓ 设备 %s 已通过 %s 连接到 %s\n",
		device.Name, device.LocalAddr, device.Endpoint)
	return nil
}

// setError 设置设备错误状态
func (d *DeviceConnection) setError(err error) {
	d.mu.Lock()
	d.Connected = false
	d.LastError = err
	d.mu.Unlock()
}

// IsConnected 检查设备是否已连接
func (d *DeviceConnection) IsConnected() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Connected
}

// GetClient 获取设备的客户端连接
func (d *DeviceConnection) GetClient() *opcua.Client {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Client
}

// StartMonitoring 开始监控所有设备
func (m *MultiInterfaceManager) StartMonitoring() {
	for _, device := range m.devices {
		m.wg.Add(1)
		go m.monitorDevice(device)
	}
}

// monitorDevice 监控单个设备
func (m *MultiInterfaceManager) monitorDevice(device *DeviceConnection) {
	defer m.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if !device.IsConnected() {
				continue
			}

			client := device.GetClient()
			if client == nil {
				continue
			}

			// 读取服务器状态作为健康检查
			if err := m.healthCheck(client, device.Name); err != nil {
				log.Printf("设备 %s 健康检查失败: %v", device.Name, err)
			}
		}
	}
}

// healthCheck 执行设备健康检查
func (m *MultiInterfaceManager) healthCheck(client *opcua.Client, deviceName string) error {
	// 读取服务器状态节点
	nodeID := ua.NewNumericNodeID(0, 2256) // Server_ServerStatus

	req := &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{
			{NodeID: nodeID},
		},
	}

	resp, err := client.Read(m.ctx, req)
	if err != nil {
		return err
	}

	if len(resp.Results) == 0 || resp.Results[0].Status != ua.StatusOK {
		return fmt.Errorf("服务器状态读取失败")
	}

	fmt.Printf("✓ 设备 %s 健康检查通过\n", deviceName)
	return nil
}

// Close 关闭所有连接
func (m *MultiInterfaceManager) Close() {
	m.cancel()
	m.wg.Wait()

	for _, device := range m.devices {
		if device.Client != nil {
			device.Client.Close(context.Background())
		}
	}
}

// GetAvailableInterfaces 获取系统可用的网络接口
func GetAvailableInterfaces() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // 跳过未启用的接口
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					result = append(result, ipnet.IP.String())
				}
			}
		}
	}
	return result, nil
}

func main() {
	// 显示可用的网络接口
	interfaces, err := GetAvailableInterfaces()
	if err != nil {
		log.Fatalf("获取网络接口失败: %v", err)
	}

	fmt.Println("可用的网络接口:")
	for _, ip := range interfaces {
		fmt.Printf("  - %s\n", ip)
	}

	// 创建多网卡管理器
	manager := NewMultiInterfaceManager()
	defer manager.Close()

	// 添加设备配置
	manager.AddDevice("设备A", "opc.tcp://192.168.100.1:4840", "192.168.100.10:0")
	manager.AddDevice("设备B", "opc.tcp://192.168.100.1:4840", "192.168.100.20:0")

	// 连接所有设备
	fmt.Println("\n开始连接设备...")
	if err := manager.ConnectAll(); err != nil {
		log.Printf("连接过程中出现错误: %v", err)
	}

	// 开始监控
	fmt.Println("\n开始监控设备...")
	manager.StartMonitoring()

	// 运行30秒后退出
	time.Sleep(30 * time.Second)
	fmt.Println("\n程序结束")
}
