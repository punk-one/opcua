// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

func TestLocalAddrOption(t *testing.T) {
	tests := []struct {
		name      string
		localAddr string
		wantErr   bool
	}{
		{
			name:      "valid IPv4 address with port",
			localAddr: "192.168.1.10:0",
			wantErr:   false,
		},
		{
			name:      "valid IPv4 address without port",
			localAddr: "192.168.1.10",
			wantErr:   true, // 应该包含端口
		},
		{
			name:      "empty address",
			localAddr: "",
			wantErr:   false, // 空地址应该被接受（使用系统默认）
		},
		{
			name:      "invalid address format",
			localAddr: "invalid-address",
			wantErr:   true,
		},
		{
			name:      "IPv6 address",
			localAddr: "[::1]:0",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试LocalAddr选项是否能正确解析地址
			opts := []opcua.Option{
				opcua.LocalAddr(tt.localAddr),
				opcua.SecurityPolicy(ua.SecurityPolicyURINone),
				opcua.SecurityModeString("None"),
			}

			// 创建客户端（不连接）
			_, err := opcua.NewClient("opc.tcp://localhost:4840", opts...)

			if (err != nil) != tt.wantErr {
				t.Errorf("LocalAddr option error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateCustomDialer(t *testing.T) {
	localAddr := "127.0.0.1:0"
	opt := createCustomDialer(localAddr)

	if opt == nil {
		t.Error("createCustomDialer returned nil")
	}
}

func TestNetworkInterfaceDetection(t *testing.T) {
	// 获取系统中可用的网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("Failed to get network interfaces: %v", err)
	}

	t.Logf("Available network interfaces:")
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
					t.Logf("  %s: %s", iface.Name, ipnet.IP.String())
				}
			}
		}
	}
}

func TestConnectionWithTimeout(t *testing.T) {
	// 测试连接超时设置
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	opts := []opcua.Option{
		opcua.LocalAddr("127.0.0.1:0"),
		opcua.SecurityPolicy(ua.SecurityPolicyURINone),
		opcua.SecurityModeString("None"),
		opcua.DialTimeout(500 * time.Millisecond),
		opcua.AutoReconnect(false), // 禁用自动重连以便测试
	}

	client, err := opcua.NewClient("opc.tcp://192.0.2.1:4840", opts...) // 使用测试用的不可达地址
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 尝试连接应该超时
	err = client.Connect(ctx)
	if err == nil {
		t.Error("Expected connection to fail, but it succeeded")
		client.Close(ctx)
	}
}
